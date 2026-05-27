package docker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	buildtypes "github.com/docker/docker/api/types/build"
	imagetypes "github.com/docker/docker/api/types/image"
)

func (m *Manager) ListImages(ctx context.Context, opts ImageListOptions) ([]ImageSummary, int64, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(opts.Labels)
	if opts.Dangling != nil {
		filters.Add("dangling", strconvBool(*opts.Dangling))
	}
	if opts.Keyword != "" {
		filters.Add("reference", opts.Keyword)
	}
	items, err := cli.ImageList(ctx, imagetypes.ListOptions{
		All:     opts.All,
		Filters: filters,
	})
	if err != nil {
		return nil, 0, err
	}
	total := int64(len(items))
	items = pageSlice(items, opts.Page, opts.PageSize)
	result := make([]ImageSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toImageSummary(item))
	}
	return result, total, nil
}

func (m *Manager) Image(ctx context.Context, id string) (*ImageInfo, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	data, err := cli.ImageInspect(ctx, id)
	if err != nil {
		return nil, err
	}
	info := toImageInfo(data)
	return &info, nil
}

func (m *Manager) PullImage(ctx context.Context, opts PullImageOptions) *Task {
	return m.tasks.Start(ctx, "image_pull", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		authHeader, err := registryAuthHeader(opts.RegistryAuth)
		if err != nil {
			return "", err
		}
		reader, err := cli.ImagePull(taskCtx, opts.Reference, imagetypes.PullOptions{
			RegistryAuth: authHeader,
			Platform:     opts.Platform,
		})
		if err != nil {
			return "", err
		}
		defer reader.Close()
		if err := streamJSONEvents(taskCtx, reader, emit); err != nil {
			return "", err
		}
		return opts.Reference, nil
	})
}

func (m *Manager) BuildImage(ctx context.Context, opts BuildImageOptions) *Task {
	return m.tasks.Start(ctx, "image_build", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		buildCtx, err := makeBuildContext(opts.ContextPath)
		if err != nil {
			return "", err
		}
		defer buildCtx.Close()
		resp, err := cli.ImageBuild(taskCtx, buildCtx, toImageBuildOptions(opts))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if err := streamJSONEvents(taskCtx, resp.Body, emit); err != nil {
			return "", err
		}
		if len(opts.Tags) > 0 {
			return opts.Tags[0], nil
		}
		return "", nil
	})
}

func (m *Manager) TagImage(ctx context.Context, id, target string) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	repo, tag := splitImageRef(target)
	return cli.ImageTag(ctx, id, imageRef(repo, tag))
}

func (m *Manager) UpdateImageTags(ctx context.Context, opts UpdateImageTagsOptions) error {
	if opts.ImageID == "" {
		return errors.New("镜像 ID 不能为空")
	}
	for _, tag := range opts.AddTags {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := m.TagImage(ctx, opts.ImageID, tag); err != nil {
			return err
		}
	}
	for _, tag := range opts.DeleteTags {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := m.DeleteImage(ctx, tag, opts.ForceDelete, false); err != nil {
			return err
		}
	}
	return nil
}

func (m *Manager) DeleteImage(ctx context.Context, id string, force bool, pruneChildren bool) error {
	cli, err := m.getClient()
	if err != nil {
		return err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	_, err = cli.ImageRemove(ctx, id, imagetypes.RemoveOptions{
		Force:         force,
		PruneChildren: pruneChildren,
	})
	return err
}

func (m *Manager) PruneImages(ctx context.Context, danglingOnly bool) *Task {
	return m.tasks.Start(ctx, "image_prune", m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		filters := filtersFromLabels(nil)
		if danglingOnly {
			filters.Add("dangling", "true")
		}
		report, err := cli.ImagesPrune(taskCtx, filters)
		if err != nil {
			return "", err
		}
		raw, _ := json.Marshal(report)
		emit(TaskEvent{Message: string(raw)})
		return "", nil
	})
}

func (m *Manager) ImageHistory(ctx context.Context, id string) ([]ImageHistoryItem, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	items, err := cli.ImageHistory(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]ImageHistoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, toImageHistory(item))
	}
	return result, nil
}

func streamJSONEvents(ctx context.Context, reader io.Reader, emit func(TaskEvent)) error {
	decoder := json.NewDecoder(reader)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		var event struct {
			Status      string `json:"status"`
			Progress    string `json:"progress"`
			ID          string `json:"id"`
			Stream      string `json:"stream"`
			Error       string `json:"error"`
			ErrorDetail struct {
				Message string `json:"message"`
			} `json:"errorDetail"`
			Aux any `json:"aux"`
		}
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		msg := strings.TrimSpace(event.Stream)
		if msg == "" {
			msg = event.Status
		}
		errText := event.Error
		if errText == "" {
			errText = event.ErrorDetail.Message
		}
		emit(TaskEvent{
			Status:   event.Status,
			Progress: event.Progress,
			ID:       event.ID,
			Message:  msg,
			Error:    errText,
		})
		if errText != "" {
			return errors.New(errText)
		}
	}
	return nil
}

func strconvBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

var _ = buildtypes.ImageBuildOptions{}
