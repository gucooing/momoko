package docker

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/go-units"
	"gopkg.in/yaml.v3"

	v1 "momoko/api/gen/v1"
)

// compose 文件里本工具能转换成单个容器创建参数的字段；其余字段（build / depends_on / healthcheck …）记为警告。
type composeService struct {
	Image         string    `yaml:"image"`
	ContainerName string    `yaml:"container_name"`
	Command       yaml.Node `yaml:"command"`
	Entrypoint    yaml.Node `yaml:"entrypoint"`
	Environment   yaml.Node `yaml:"environment"`
	Ports         yaml.Node `yaml:"ports"`
	Volumes       yaml.Node `yaml:"volumes"`
	Restart       string    `yaml:"restart"`
	Networks      yaml.Node `yaml:"networks"`
	NetworkMode   string    `yaml:"network_mode"`
	Hostname      string    `yaml:"hostname"`
	User          string    `yaml:"user"`
	WorkingDir    string    `yaml:"working_dir"`
	Labels        yaml.Node `yaml:"labels"`
	Tty           bool      `yaml:"tty"`
	StdinOpen     bool      `yaml:"stdin_open"`
	Privileged    bool      `yaml:"privileged"`
	MemLimit      yaml.Node `yaml:"mem_limit"`
	Cpus          yaml.Node `yaml:"cpus"`
	Platform      string    `yaml:"platform"`
	Deploy        struct {
		Resources struct {
			Limits struct {
				Cpus   yaml.Node `yaml:"cpus"`
				Memory yaml.Node `yaml:"memory"`
			} `yaml:"limits"`
		} `yaml:"resources"`
	} `yaml:"deploy"`
}

var composeSupportedKeys = map[string]bool{
	"image": true, "container_name": true, "command": true, "entrypoint": true, "environment": true,
	"ports": true, "volumes": true, "restart": true, "networks": true, "network_mode": true,
	"hostname": true, "user": true, "working_dir": true, "labels": true, "tty": true, "stdin_open": true,
	"privileged": true, "mem_limit": true, "cpus": true, "platform": true, "deploy": true,
}

// ParseCompose 把 docker compose 文件里的每个服务转换成容器创建参数。
func ParseCompose(content string) ([]*v1.DockerComposeService, error) {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(content), &root); err != nil {
		return nil, fmt.Errorf("YAML 解析失败: %w", err)
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return nil, errors.New("不是有效的 compose 文件")
	}
	doc := root.Content[0]
	services := mappingValue(doc, "services")
	if services == nil {
		// 旧版（v1）compose：服务直接写在顶层
		services = doc
	}
	if services.Kind != yaml.MappingNode {
		return nil, errors.New("compose 文件里没有 services")
	}
	result := make([]*v1.DockerComposeService, 0, len(services.Content)/2)
	for i := 0; i+1 < len(services.Content); i += 2 {
		name := services.Content[i].Value
		node := services.Content[i+1]
		if node.Kind != yaml.MappingNode {
			continue
		}
		var svc composeService
		if err := node.Decode(&svc); err != nil {
			return nil, fmt.Errorf("服务 %s 解析失败: %w", name, err)
		}
		item := &v1.DockerComposeService{Name: name}
		for k := 0; k+1 < len(node.Content); k += 2 {
			if key := node.Content[k].Value; !composeSupportedKeys[key] {
				item.Warnings = append(item.Warnings, key)
			}
		}
		item.Options = svc.toOptions(name, &item.Warnings)
		result = append(result, item)
	}
	if len(result) == 0 {
		return nil, errors.New("compose 文件里没有服务")
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (s *composeService) toOptions(service string, warnings *[]string) *v1.DockerContainerCreateOptions {
	opts := &v1.DockerContainerCreateOptions{
		Name:       strings.TrimSpace(s.ContainerName),
		Image:      strings.TrimSpace(s.Image),
		Hostname:   s.Hostname,
		User:       s.User,
		WorkingDir: s.WorkingDir,
		Tty:        s.Tty,
		OpenStdin:  s.StdinOpen,
		Privileged: s.Privileged,
		Platform:   s.Platform,
		Start:      true,
	}
	if opts.Name == "" {
		opts.Name = service
	}
	opts.Cmd = commandList(&s.Command)
	opts.Entrypoint = commandList(&s.Entrypoint)
	opts.Env = envList(&s.Environment)
	opts.Labels = labelMap(&s.Labels)
	opts.Ports = composePorts(&s.Ports, warnings)
	opts.Mounts = composeMounts(&s.Volumes)

	restart := strings.TrimSpace(s.Restart)
	if name, retries, ok := strings.Cut(restart, ":"); ok {
		restart = name
		if n, err := strconv.Atoi(retries); err == nil {
			opts.RestartMaxRetries = int32(n)
		}
	}
	if restart != "no" {
		opts.RestartPolicy = restart
	}

	switch {
	case s.NetworkMode != "":
		opts.Network = s.NetworkMode
	case s.Networks.Kind == yaml.SequenceNode && len(s.Networks.Content) > 0:
		opts.Network = s.Networks.Content[0].Value
		if len(s.Networks.Content) > 1 {
			*warnings = append(*warnings, "networks")
		}
	case s.Networks.Kind == yaml.MappingNode && len(s.Networks.Content) >= 2:
		opts.Network = s.Networks.Content[0].Value
		if len(s.Networks.Content) > 2 {
			*warnings = append(*warnings, "networks")
		}
	}

	memory := scalarValue(&s.MemLimit)
	if memory == "" {
		memory = scalarValue(&s.Deploy.Resources.Limits.Memory)
	}
	if memory != "" {
		if n, err := units.RAMInBytes(memory); err == nil {
			opts.Memory = n
		} else {
			*warnings = append(*warnings, "mem_limit")
		}
	}
	cpus := scalarValue(&s.Cpus)
	if cpus == "" {
		cpus = scalarValue(&s.Deploy.Resources.Limits.Cpus)
	}
	if cpus != "" {
		if n, err := strconv.ParseFloat(cpus, 64); err == nil && n > 0 {
			opts.NanoCpus = int64(math.Round(n * 1e9))
		} else {
			*warnings = append(*warnings, "cpus")
		}
	}
	return opts
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode || node.Tag == "!!null" {
		return ""
	}
	return strings.TrimSpace(node.Value)
}

// commandList 支持字符串（按 shell 规则拆分）与列表两种写法。
func commandList(node *yaml.Node) []string {
	switch node.Kind {
	case yaml.ScalarNode:
		if node.Tag == "!!null" || strings.TrimSpace(node.Value) == "" {
			return nil
		}
		return SplitCommand(node.Value)
	case yaml.SequenceNode:
		out := make([]string, 0, len(node.Content))
		for _, item := range node.Content {
			out = append(out, item.Value)
		}
		return out
	}
	return nil
}

// envList 支持 KEY: value 映射与 - KEY=value 列表两种写法。
func envList(node *yaml.Node) []string {
	out := []string{}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			value := node.Content[i+1]
			if value.Tag == "!!null" {
				out = append(out, node.Content[i].Value+"=")
				continue
			}
			out = append(out, node.Content[i].Value+"="+value.Value)
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if v := strings.TrimSpace(item.Value); v != "" {
				if !strings.Contains(v, "=") {
					v += "="
				}
				out = append(out, v)
			}
		}
	}
	return out
}

func labelMap(node *yaml.Node) map[string]string {
	out := map[string]string{}
	switch node.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			out[node.Content[i].Value] = node.Content[i+1].Value
		}
	case yaml.SequenceNode:
		for _, item := range node.Content {
			k, v, _ := strings.Cut(item.Value, "=")
			if k = strings.TrimSpace(k); k != "" {
				out[k] = v
			}
		}
	}
	return out
}

// composePorts 支持短写法 [HOST_IP:][HOST:]CONTAINER[/PROTO] 与长写法 {target, published, protocol, host_ip}。
func composePorts(node *yaml.Node, warnings *[]string) []*v1.DockerPortBinding {
	out := []*v1.DockerPortBinding{}
	if node.Kind != yaml.SequenceNode {
		return out
	}
	for _, item := range node.Content {
		if item.Kind == yaml.MappingNode {
			target := scalarValue(mappingValue(item, "target"))
			if target == "" {
				continue
			}
			proto := scalarValue(mappingValue(item, "protocol"))
			if proto == "" {
				proto = "tcp"
			}
			out = append(out, &v1.DockerPortBinding{
				ContainerPort: target + "/" + strings.ToLower(proto),
				HostPort:      scalarValue(mappingValue(item, "published")),
				HostIp:        scalarValue(mappingValue(item, "host_ip")),
			})
			continue
		}
		spec := strings.TrimSpace(item.Value)
		if spec == "" {
			continue
		}
		proto := "tcp"
		if base, p, ok := strings.Cut(spec, "/"); ok {
			spec, proto = base, strings.ToLower(p)
		}
		parts := strings.Split(spec, ":")
		binding := &v1.DockerPortBinding{}
		switch len(parts) {
		case 1:
			binding.ContainerPort = parts[0]
		case 2:
			binding.HostPort, binding.ContainerPort = parts[0], parts[1]
		default:
			binding.HostIp = strings.Join(parts[:len(parts)-2], ":")
			binding.HostPort, binding.ContainerPort = parts[len(parts)-2], parts[len(parts)-1]
		}
		if strings.Contains(binding.ContainerPort, "-") || strings.Contains(binding.HostPort, "-") {
			*warnings = append(*warnings, "ports "+item.Value)
			continue
		}
		binding.HostIp = strings.Trim(binding.HostIp, "[]")
		binding.ContainerPort += "/" + proto
		out = append(out, binding)
	}
	return out
}

// composeMounts 支持短写法 SRC:DST[:ro] 与长写法 {type, source, target, read_only}；
// 来源不是路径（不以 / . ~ 开头）的视为命名卷。
func composeMounts(node *yaml.Node) []*v1.DockerMount {
	out := []*v1.DockerMount{}
	if node.Kind != yaml.SequenceNode {
		return out
	}
	for _, item := range node.Content {
		if item.Kind == yaml.MappingNode {
			target := scalarValue(mappingValue(item, "target"))
			if target == "" {
				continue
			}
			mountType := scalarValue(mappingValue(item, "type"))
			if mountType == "" {
				mountType = "volume"
			}
			out = append(out, &v1.DockerMount{
				Type:     mountType,
				Source:   scalarValue(mappingValue(item, "source")),
				Target:   target,
				ReadOnly: scalarValue(mappingValue(item, "read_only")) == "true",
			})
			continue
		}
		parts := strings.Split(strings.TrimSpace(item.Value), ":")
		mount := &v1.DockerMount{Type: "volume"}
		switch len(parts) {
		case 1:
			mount.Target = parts[0]
		default:
			mount.Source, mount.Target = parts[0], parts[1]
			if len(parts) > 2 {
				for _, mode := range strings.Split(parts[2], ",") {
					if mode == "ro" {
						mount.ReadOnly = true
					}
				}
			}
		}
		if mount.Target == "" {
			continue
		}
		if strings.HasPrefix(mount.Source, "/") || strings.HasPrefix(mount.Source, ".") || strings.HasPrefix(mount.Source, "~") {
			mount.Type = "bind"
		}
		out = append(out, mount)
	}
	return out
}

// SplitCommand 按简化的 shell 规则拆分命令行（支持单双引号与反斜杠转义）。
func SplitCommand(text string) []string {
	var (
		out     []string
		cur     strings.Builder
		quote   rune
		escaped bool
		has     bool
	)
	for _, r := range text {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\' && quote != '\'':
			escaped = true
			has = true
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			has = true
		case r == ' ' || r == '\t' || r == '\n':
			if has || cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
				has = false
			}
		default:
			cur.WriteRune(r)
		}
	}
	if has || cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}
