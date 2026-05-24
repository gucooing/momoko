package initialize

import (
	"fmt"
	"momoko/pkg/utils"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"momoko/internal/conf"
)

func ApplyDatabaseConfig(configPath string, database *conf.Data_Database) error {
	if database == nil {
		return fmt.Errorf("database config is required")
	}

	filePath, err := resolveConfigFile(configPath)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(filePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	var root yaml.Node
	if len(content) > 0 {
		if err := yaml.Unmarshal(content, &root); err != nil {
			return err
		}
	} else {
		root = yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{Kind: yaml.MappingNode},
			},
		}
	}
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	if len(root.Content) == 0 || root.Content[0].Kind != yaml.MappingNode {
		return fmt.Errorf("invalid config file: root should be a mapping")
	}

	rootMap := root.Content[0]
	dataNode := mappingValue(rootMap, "data")
	if dataNode == nil {
		dataNode = addMapping(rootMap, "data")
	}
	databaseNode := mappingValue(dataNode, "database")
	if databaseNode == nil {
		databaseNode = addMapping(dataNode, "database")
	}

	setScalar(databaseNode, "driver", database.Driver)
	setScalar(databaseNode, "source", database.Source)

	authNode := mappingValue(rootMap, "auth")
	if authNode == nil {
		authNode = addMapping(rootMap, "auth")
	}
	setScalar(authNode, "secret", utils.GenerateRandomString(12))

	next, err := yaml.Marshal(&root)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, next, 0o644)
}

func resolveConfigFile(configPath string) (string, error) {
	if configPath == "" {
		configPath = "./configs"
	}
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) && filepath.Ext(configPath) != "" {
			return configPath, nil
		}
		if os.IsNotExist(err) {
			return filepath.Join(configPath, "config.yaml"), nil
		}
		return "", err
	}
	if !info.IsDir() {
		return configPath, nil
	}

	filePath := filepath.Join(configPath, "config.yaml")
	return filePath, nil
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func addMapping(node *yaml.Node, key string) *yaml.Node {
	child := &yaml.Node{Kind: yaml.MappingNode}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		child,
	)
	return child
}

func setScalar(node *yaml.Node, key string, value string) {
	if node.Kind != yaml.MappingNode {
		node.Kind = yaml.MappingNode
		node.Content = nil
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1] = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
			return
		}
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value},
	)
}
