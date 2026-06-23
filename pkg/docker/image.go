package docker

import (
	"context"
	"errors"
	"io"
	"strconv"
	"strings"

	imagetypes "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"

	v1 "momoko/api/gen/v1"
)

func (m *Manager) ListImages(ctx context.Context, req *v1.ListDockerImagesRequest) (*v1.ListDockerImagesResponse, error) {
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()

	filters := filtersFromLabels(req.GetLabels())
	if req != nil && req.Dangling != nil {
		filters.Add("dangling", strconv.FormatBool(req.GetDangling()))
	}
	if req.GetKeyword() != "" {
		filters.Add("reference", req.GetKeyword())
	}
	items, err := cli.ImageList(ctx, imagetypes.ListOptions{
		All:     req.GetAll(),
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}
	total := int64(len(items))
	items = pageSlice(items, req.GetPage(), req.GetPageSize())
	result := make([]*v1.DockerImageSummary, 0, len(items))
	for _, item := range items {
		result = append(result, toImageSummary(item))
	}
	return &v1.ListDockerImagesResponse{Items: result, Total: total}, nil
}

func (m *Manager) Image(ctx context.Context, id string) (*v1.DockerImageInfo, error) {
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
	return toImageInfo(data), nil
}

func (m *Manager) PullImage(ctx context.Context, req *v1.PullDockerImageRequest) *v1.DockerTaskInfo {
	return m.tasks.Start(ctx, v1.DockerTaskType_DOCKER_TASK_TYPE_IMAGE_PULL, "拉取镜像 "+req.GetReference(), m.taskTimeout(), func(taskCtx context.Context, emit func(*v1.DockerTaskInfo)) (string, error) {
		cli, err := m.getClient()
		if err != nil {
			return "", err
		}
		authHeader, err := registryAuthHeader(req.GetRegistryAuth())
		if err != nil {
			return "", err
		}
		reader, err := cli.ImagePull(taskCtx, req.GetReference(), imagetypes.PullOptions{
			RegistryAuth: authHeader,
			Platform:     req.GetPlatform(),
		})
		if err != nil {
			return "", err
		}
		defer reader.Close()
		if err := streamTaskOutput(taskCtx, reader, emit); err != nil {
			return "", err
		}
		return req.GetReference(), nil
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

func (m *Manager) UpdateImageTags(ctx context.Context, req *v1.UpdateDockerImageTagsRequest) error {
	if req.GetImageId() == "" {
		return errors.New("镜像 ID 不能为空")
	}
	for _, tag := range req.GetAddTags() {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := m.TagImage(ctx, req.GetImageId(), tag); err != nil {
			return err
		}
	}
	for _, tag := range req.GetDeleteTags() {
		if strings.TrimSpace(tag) == "" {
			continue
		}
		if err := m.DeleteImage(ctx, tag, req.GetForceDelete(), false); err != nil {
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

func (m *Manager) ImageHistory(ctx context.Context, id string) ([]*v1.DockerImageHistoryItem, error) {
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
	result := make([]*v1.DockerImageHistoryItem, 0, len(items))
	for _, item := range items {
		result = append(result, toImageHistory(item))
	}
	return result, nil
}

func streamTaskOutput(ctx context.Context, reader io.Reader, emit func(*v1.DockerTaskInfo)) error {
	return jsonmessage.DisplayJSONMessagesStream(reader, taskEventWriter{ctx: ctx, emit: emit}, 0, true, nil)
}

type taskEventWriter struct {
	ctx  context.Context
	emit func(*v1.DockerTaskInfo)
}

func (w taskEventWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	if len(p) > 0 {
		w.emit(&v1.DockerTaskInfo{Message: string(p)})
	}
	return len(p), nil
}
