package service

import (
	"context"
	"io"
	"sync"

	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/net/websocket"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

type DockerService struct {
	v1.UnimplementedDockerManagerServer

	uc *biz.DockerUsecase
}

func NewDockerService(uc *biz.DockerUsecase) *DockerService {
	return &DockerService{uc: uc}
}

func (d *DockerService) RegisterWsServer(srv *http.Server) {
	srv.Handle(biz.DockerContainerLogsWSPath, websocket.Handler(d.RunContainerLogsWsConn))
	srv.Handle(biz.DockerContainerStatsWSPath, websocket.Handler(d.RunContainerStatsWsConn))
	srv.Handle(biz.DockerContainerExecWSPath, websocket.Handler(d.RunContainerExecWsConn))
	srv.Handle(biz.DockerTaskWSPath, websocket.Handler(d.RunTaskWsConn))
}

func (d *DockerService) DockerStatus(ctx context.Context, _ *v1.DockerStatusRequest) (*v1.DockerStatusResponse, error) {
	return d.uc.Status(ctx)
}

func (d *DockerService) GetDockerConfig(ctx context.Context, _ *v1.GetDockerConfigRequest) (*v1.GetDockerConfigResponse, error) {
	config, err := d.uc.Config(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetDockerConfigResponse{Config: config}, nil
}

func (d *DockerService) UpdateDockerConfig(ctx context.Context, req *v1.UpdateDockerConfigRequest) (*v1.UpdateDockerConfigResponse, error) {
	config, err := d.uc.UpdateConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDockerConfigResponse{Config: config}, nil
}

func (d *DockerService) TestDockerConfig(ctx context.Context, req *v1.TestDockerConfigRequest) (*v1.TestDockerConfigResponse, error) {
	status, err := d.uc.TestConfig(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.TestDockerConfigResponse{Status: status}, nil
}

func (d *DockerService) ListDockerTasks(ctx context.Context, _ *v1.ListDockerTasksRequest) (*v1.ListDockerTasksResponse, error) {
	tasks, err := d.uc.Tasks(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListDockerTasksResponse{Tasks: tasks}, nil
}

func (d *DockerService) ListDockerContainers(ctx context.Context, req *v1.ListDockerContainersRequest) (*v1.ListDockerContainersResponse, error) {
	return d.uc.ListContainers(ctx, req)
}

func (d *DockerService) GetDockerContainer(ctx context.Context, req *v1.GetDockerContainerRequest) (*v1.GetDockerContainerResponse, error) {
	info, err := d.uc.Container(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetDockerContainerResponse{Info: info}, nil
}

func (d *DockerService) CreateDockerContainer(ctx context.Context, req *v1.CreateDockerContainerRequest) (*v1.CreateDockerContainerResponse, error) {
	id, err := d.uc.CreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateDockerContainerResponse{Id: id}, nil
}

func (d *DockerService) UpdateDockerContainer(ctx context.Context, req *v1.UpdateDockerContainerRequest) (*v1.UpdateDockerContainerResponse, error) {
	info, task, err := d.uc.UpdateContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDockerContainerResponse{Info: info, Task: task}, nil
}

func (d *DockerService) RecreateDockerContainer(ctx context.Context, req *v1.RecreateDockerContainerRequest) (*v1.RecreateDockerContainerResponse, error) {
	task, err := d.uc.RecreateContainer(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.RecreateDockerContainerResponse{Task: task}, nil
}

func (d *DockerService) StartDockerContainer(ctx context.Context, req *v1.StartDockerContainerRequest) (*v1.StartDockerContainerResponse, error) {
	if err := d.uc.StartContainer(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.StartDockerContainerResponse{}, nil
}

func (d *DockerService) StopDockerContainer(ctx context.Context, req *v1.StopDockerContainerRequest) (*v1.StopDockerContainerResponse, error) {
	if err := d.uc.StopContainer(ctx, req.Id, req.TimeoutSeconds); err != nil {
		return nil, err
	}
	return &v1.StopDockerContainerResponse{}, nil
}

func (d *DockerService) RestartDockerContainer(ctx context.Context, req *v1.RestartDockerContainerRequest) (*v1.RestartDockerContainerResponse, error) {
	if err := d.uc.RestartContainer(ctx, req.Id, req.TimeoutSeconds); err != nil {
		return nil, err
	}
	return &v1.RestartDockerContainerResponse{}, nil
}

func (d *DockerService) KillDockerContainer(ctx context.Context, req *v1.KillDockerContainerRequest) (*v1.KillDockerContainerResponse, error) {
	if err := d.uc.KillContainer(ctx, req.Id, req.Signal); err != nil {
		return nil, err
	}
	return &v1.KillDockerContainerResponse{}, nil
}

func (d *DockerService) PauseDockerContainer(ctx context.Context, req *v1.PauseDockerContainerRequest) (*v1.PauseDockerContainerResponse, error) {
	if err := d.uc.PauseContainer(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.PauseDockerContainerResponse{}, nil
}

func (d *DockerService) UnpauseDockerContainer(ctx context.Context, req *v1.UnpauseDockerContainerRequest) (*v1.UnpauseDockerContainerResponse, error) {
	if err := d.uc.UnpauseContainer(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.UnpauseDockerContainerResponse{}, nil
}

func (d *DockerService) RenameDockerContainer(ctx context.Context, req *v1.RenameDockerContainerRequest) (*v1.RenameDockerContainerResponse, error) {
	if err := d.uc.RenameContainer(ctx, req.Id, req.Name); err != nil {
		return nil, err
	}
	return &v1.RenameDockerContainerResponse{}, nil
}

func (d *DockerService) DeleteDockerContainer(ctx context.Context, req *v1.DeleteDockerContainerRequest) (*v1.DeleteDockerContainerResponse, error) {
	if err := d.uc.DeleteContainer(ctx, req.Id, req.Force, req.RemoveVolumes); err != nil {
		return nil, err
	}
	return &v1.DeleteDockerContainerResponse{}, nil
}

func (d *DockerService) ContainerLogs(ctx context.Context, req *v1.ContainerLogsRequest) (*v1.ContainerLogsResponse, error) {
	logs, err := d.uc.ContainerLogs(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ContainerLogsResponse{Logs: logs}, nil
}

func (d *DockerService) ContainerStats(ctx context.Context, req *v1.ContainerStatsRequest) (*v1.ContainerStatsResponse, error) {
	stats, err := d.uc.ContainerStats(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.ContainerStatsResponse{Json: stats}, nil
}

func (d *DockerService) CreateContainerExec(ctx context.Context, req *v1.CreateContainerExecRequest) (*v1.CreateContainerExecResponse, error) {
	execID, err := d.uc.CreateExec(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateContainerExecResponse{ExecId: execID}, nil
}

func (d *DockerService) ListDockerImages(ctx context.Context, req *v1.ListDockerImagesRequest) (*v1.ListDockerImagesResponse, error) {
	return d.uc.ListImages(ctx, req)
}

func (d *DockerService) GetDockerImage(ctx context.Context, req *v1.GetDockerImageRequest) (*v1.GetDockerImageResponse, error) {
	info, err := d.uc.Image(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetDockerImageResponse{Info: info}, nil
}

func (d *DockerService) PullDockerImage(ctx context.Context, req *v1.PullDockerImageRequest) (*v1.PullDockerImageResponse, error) {
	task, err := d.uc.PullImage(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.PullDockerImageResponse{Task: task}, nil
}

func (d *DockerService) UpdateDockerImageTags(ctx context.Context, req *v1.UpdateDockerImageTagsRequest) (*v1.UpdateDockerImageTagsResponse, error) {
	info, err := d.uc.UpdateImageTags(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDockerImageTagsResponse{Info: info}, nil
}

func (d *DockerService) TagDockerImage(ctx context.Context, req *v1.TagDockerImageRequest) (*v1.TagDockerImageResponse, error) {
	if err := d.uc.TagImage(ctx, req.Id, req.Target); err != nil {
		return nil, err
	}
	return &v1.TagDockerImageResponse{}, nil
}

func (d *DockerService) DeleteDockerImage(ctx context.Context, req *v1.DeleteDockerImageRequest) (*v1.DeleteDockerImageResponse, error) {
	if err := d.uc.DeleteImage(ctx, req.Id, req.Force, req.PruneChildren); err != nil {
		return nil, err
	}
	return &v1.DeleteDockerImageResponse{}, nil
}

func (d *DockerService) ImageHistory(ctx context.Context, req *v1.ImageHistoryRequest) (*v1.ImageHistoryResponse, error) {
	items, err := d.uc.ImageHistory(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.ImageHistoryResponse{Items: items}, nil
}

func (d *DockerService) ListDockerNetworks(ctx context.Context, req *v1.ListDockerNetworksRequest) (*v1.ListDockerNetworksResponse, error) {
	return d.uc.ListNetworks(ctx, req)
}

func (d *DockerService) GetDockerNetwork(ctx context.Context, req *v1.GetDockerNetworkRequest) (*v1.GetDockerNetworkResponse, error) {
	info, err := d.uc.Network(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetDockerNetworkResponse{Info: info}, nil
}

func (d *DockerService) CreateDockerNetwork(ctx context.Context, req *v1.CreateDockerNetworkRequest) (*v1.CreateDockerNetworkResponse, error) {
	id, err := d.uc.CreateNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateDockerNetworkResponse{Id: id}, nil
}

func (d *DockerService) UpdateDockerNetwork(ctx context.Context, req *v1.UpdateDockerNetworkRequest) (*v1.UpdateDockerNetworkResponse, error) {
	info, task, err := d.uc.UpdateNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDockerNetworkResponse{Info: info, Task: task}, nil
}

func (d *DockerService) RecreateDockerNetwork(ctx context.Context, req *v1.RecreateDockerNetworkRequest) (*v1.RecreateDockerNetworkResponse, error) {
	task, err := d.uc.RecreateNetwork(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.RecreateDockerNetworkResponse{Task: task}, nil
}

func (d *DockerService) DeleteDockerNetwork(ctx context.Context, req *v1.DeleteDockerNetworkRequest) (*v1.DeleteDockerNetworkResponse, error) {
	if err := d.uc.DeleteNetwork(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteDockerNetworkResponse{}, nil
}

func (d *DockerService) ConnectDockerNetwork(ctx context.Context, req *v1.ConnectDockerNetworkRequest) (*v1.ConnectDockerNetworkResponse, error) {
	if err := d.uc.ConnectNetwork(ctx, req); err != nil {
		return nil, err
	}
	return &v1.ConnectDockerNetworkResponse{}, nil
}

func (d *DockerService) DisconnectDockerNetwork(ctx context.Context, req *v1.DisconnectDockerNetworkRequest) (*v1.DisconnectDockerNetworkResponse, error) {
	if err := d.uc.DisconnectNetwork(ctx, req); err != nil {
		return nil, err
	}
	return &v1.DisconnectDockerNetworkResponse{}, nil
}

func (d *DockerService) PruneDockerNetworks(ctx context.Context, _ *v1.PruneDockerNetworksRequest) (*v1.PruneDockerNetworksResponse, error) {
	task, err := d.uc.PruneNetworks(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.PruneDockerNetworksResponse{Task: task}, nil
}

func (d *DockerService) ListDockerVolumes(ctx context.Context, req *v1.ListDockerVolumesRequest) (*v1.ListDockerVolumesResponse, error) {
	return d.uc.ListVolumes(ctx, req)
}

func (d *DockerService) GetDockerVolume(ctx context.Context, req *v1.GetDockerVolumeRequest) (*v1.GetDockerVolumeResponse, error) {
	info, err := d.uc.Volume(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &v1.GetDockerVolumeResponse{Info: info}, nil
}

func (d *DockerService) CreateDockerVolume(ctx context.Context, req *v1.CreateDockerVolumeRequest) (*v1.CreateDockerVolumeResponse, error) {
	info, err := d.uc.CreateVolume(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateDockerVolumeResponse{Info: info}, nil
}

func (d *DockerService) UpdateDockerVolume(ctx context.Context, req *v1.UpdateDockerVolumeRequest) (*v1.UpdateDockerVolumeResponse, error) {
	info, task, err := d.uc.UpdateVolume(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateDockerVolumeResponse{Info: info, Task: task}, nil
}

func (d *DockerService) RecreateDockerVolume(ctx context.Context, req *v1.RecreateDockerVolumeRequest) (*v1.RecreateDockerVolumeResponse, error) {
	task, err := d.uc.RecreateVolume(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.RecreateDockerVolumeResponse{Task: task}, nil
}

func (d *DockerService) DeleteDockerVolume(ctx context.Context, req *v1.DeleteDockerVolumeRequest) (*v1.DeleteDockerVolumeResponse, error) {
	if err := d.uc.DeleteVolume(ctx, req.Name, req.Force); err != nil {
		return nil, err
	}
	return &v1.DeleteDockerVolumeResponse{}, nil
}

func (d *DockerService) PruneDockerVolumes(ctx context.Context, _ *v1.PruneDockerVolumesRequest) (*v1.PruneDockerVolumesResponse, error) {
	task, err := d.uc.PruneVolumes(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.PruneDockerVolumesResponse{Task: task}, nil
}

func (d *DockerService) ExportDockerVolume(ctx context.Context, req *v1.ExportDockerVolumeRequest) (*v1.ExportDockerVolumeResponse, error) {
	task, err := d.uc.ExportVolume(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ExportDockerVolumeResponse{Task: task}, nil
}

func (d *DockerService) RestoreDockerVolume(ctx context.Context, req *v1.RestoreDockerVolumeRequest) (*v1.RestoreDockerVolumeResponse, error) {
	task, err := d.uc.RestoreVolume(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.RestoreDockerVolumeResponse{Task: task}, nil
}

func (d *DockerService) RunContainerLogsWsConn(conn *websocket.Conn) {
	defer conn.Close()

	req := conn.Request()
	reader, err := d.uc.ContainerLogStream(req.Context(), &v1.ContainerLogsRequest{
		Id:         req.URL.Query().Get("id"),
		Stdout:     req.URL.Query().Get("stdout") != "false",
		Stderr:     req.URL.Query().Get("stderr") != "false",
		Timestamps: req.URL.Query().Get("timestamps") == "true",
		Tail:       req.URL.Query().Get("tail"),
		Details:    req.URL.Query().Get("details") == "true",
	})
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return
	}
	defer reader.Close()

	sendDockerReader(conn, reader)
}

func (d *DockerService) RunContainerStatsWsConn(conn *websocket.Conn) {
	defer conn.Close()

	reader, err := d.uc.ContainerStatsStream(conn.Request().Context(), conn.Request().URL.Query().Get("id"))
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return
	}
	defer reader.Close()

	sendDockerReader(conn, reader)
}

func (d *DockerService) RunContainerExecWsConn(conn *websocket.Conn) {
	defer conn.Close()

	req := conn.Request()
	session, err := d.uc.AttachExec(req.Context(), req.URL.Query().Get("exec_id"), req.URL.Query().Get("tty") != "false")
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return
	}
	if session.Closer != nil {
		defer session.Closer()
	}

	done := make(chan struct{})
	var once sync.Once
	closeDone := func() {
		once.Do(func() {
			close(done)
		})
	}
	go func() {
		defer closeDone()
		sendDockerReader(conn, session.Reader)
	}()
	go func() {
		defer closeDone()
		for {
			var input []byte
			if err := websocket.Message.Receive(conn, &input); err != nil {
				return
			}
			if _, err := session.Writer.Write(input); err != nil {
				return
			}
		}
	}()
	<-done
}

func (d *DockerService) RunTaskWsConn(conn *websocket.Conn) {
	defer conn.Close()

	ctx := conn.Request().Context()
	ch, cancel, err := d.uc.SubscribeTask(ctx, conn.Request().URL.Query().Get("task_id"))
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return
	}
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			if event.Message != "" {
				if err := websocket.Message.Send(conn, event.Message); err != nil {
					return
				}
			}
			if event.Error != "" {
				if err := websocket.Message.Send(conn, event.Error); err != nil {
					return
				}
			}
		}
	}
}

func sendDockerReader(conn *websocket.Conn, reader io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		n, err := reader.Read(buf)
		if n > 0 {
			if sendErr := websocket.Message.Send(conn, buf[:n]); sendErr != nil {
				return
			}
		}
		if err != nil {
			return
		}
	}
}
