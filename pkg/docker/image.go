package docker

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
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
		filters.Add("dangling", strconv.FormatBool(*opts.Dangling))
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
	return m.tasks.Start(ctx, "image_pull", "拉取镜像 "+opts.Reference, m.taskTimeout(), func(taskCtx context.Context, emit func(TaskEvent)) (string, error) {
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
		if err := streamTaskOutput(taskCtx, reader, emit); err != nil {
			return "", err
		}
		return opts.Reference, nil
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

func streamTaskOutput(ctx context.Context, reader io.Reader, emit func(TaskEvent)) error {
	return jsonmessage.DisplayJSONMessagesStream(reader, taskEventWriter{ctx: ctx, emit: emit}, 0, true, nil)
}

type taskEventWriter struct {
	ctx  context.Context
	emit func(TaskEvent)
}

func (w taskEventWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) > 0 {
		w.emit(TaskEvent{Message: string(p)})
	}
	return len(p), nil
}
