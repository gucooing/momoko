import request from '@/utils/request'
import type {
  DockerStatusRequest,
  DockerStatusResponse,
  GetDockerConfigRequest,
  GetDockerConfigResponse,
  UpdateDockerConfigRequest,
  UpdateDockerConfigResponse,
  TestDockerConfigRequest,
  TestDockerConfigResponse,
  ListDockerTasksRequest,
  ListDockerTasksResponse,
  GetDockerTaskRequest,
  GetDockerTaskResponse,
  ListDockerContainersRequest,
  ListDockerContainersResponse,
  GetDockerContainerRequest,
  GetDockerContainerResponse,
  CreateDockerContainerRequest,
  CreateDockerContainerResponse,
  UpdateDockerContainerRequest,
  UpdateDockerContainerResponse,
  RecreateDockerContainerRequest,
  RecreateDockerContainerResponse,
  StartDockerContainerRequest,
  StartDockerContainerResponse,
  StopDockerContainerRequest,
  StopDockerContainerResponse,
  RestartDockerContainerRequest,
  RestartDockerContainerResponse,
  KillDockerContainerRequest,
  KillDockerContainerResponse,
  PauseDockerContainerRequest,
  PauseDockerContainerResponse,
  UnpauseDockerContainerRequest,
  UnpauseDockerContainerResponse,
  RenameDockerContainerRequest,
  RenameDockerContainerResponse,
  DeleteDockerContainerRequest,
  DeleteDockerContainerResponse,
  ContainerLogsRequest,
  ContainerLogsResponse,
  ContainerStatsRequest,
  ContainerStatsResponse,
  CreateContainerExecRequest,
  CreateContainerExecResponse,
  ListDockerImagesRequest,
  ListDockerImagesResponse,
  GetDockerImageRequest,
  GetDockerImageResponse,
  PullDockerImageRequest,
  PullDockerImageResponse,
  BuildDockerImageRequest,
  BuildDockerImageResponse,
  UpdateDockerImageTagsRequest,
  UpdateDockerImageTagsResponse,
  TagDockerImageRequest,
  TagDockerImageResponse,
  DeleteDockerImageRequest,
  DeleteDockerImageResponse,
  PruneDockerImagesRequest,
  PruneDockerImagesResponse,
  ImageHistoryRequest,
  ImageHistoryResponse,
  ListDockerNetworksRequest,
  ListDockerNetworksResponse,
  GetDockerNetworkRequest,
  GetDockerNetworkResponse,
  CreateDockerNetworkRequest,
  CreateDockerNetworkResponse,
  UpdateDockerNetworkRequest,
  UpdateDockerNetworkResponse,
  RecreateDockerNetworkRequest,
  RecreateDockerNetworkResponse,
  DeleteDockerNetworkRequest,
  DeleteDockerNetworkResponse,
  ConnectDockerNetworkRequest,
  ConnectDockerNetworkResponse,
  DisconnectDockerNetworkRequest,
  DisconnectDockerNetworkResponse,
  PruneDockerNetworksRequest,
  PruneDockerNetworksResponse,
  ListDockerVolumesRequest,
  ListDockerVolumesResponse,
  GetDockerVolumeRequest,
  GetDockerVolumeResponse,
  CreateDockerVolumeRequest,
  CreateDockerVolumeResponse,
  UpdateDockerVolumeRequest,
  UpdateDockerVolumeResponse,
  RecreateDockerVolumeRequest,
  RecreateDockerVolumeResponse,
  DeleteDockerVolumeRequest,
  DeleteDockerVolumeResponse,
  PruneDockerVolumesRequest,
  PruneDockerVolumesResponse,
  ExportDockerVolumeRequest,
  ExportDockerVolumeResponse,
  RestoreDockerVolumeRequest,
  RestoreDockerVolumeResponse,
} from '@/types/v1/docker'

// ==================== Status & Config ====================

export const getDockerStatus = (params?: DockerStatusRequest) => {
  return request.get<DockerStatusResponse>('/docker/status', { params })
}

export const getDockerConfig = (params?: GetDockerConfigRequest) => {
  return request.get<GetDockerConfigResponse>('/docker/config', { params })
}

export const updateDockerConfig = (data: UpdateDockerConfigRequest) => {
  return request.put<UpdateDockerConfigResponse>('/docker/config', data)
}

export const testDockerConfig = (data: TestDockerConfigRequest) => {
  return request.post<TestDockerConfigResponse>('/docker/config/test', data)
}

// ==================== Tasks ====================

export const listDockerTasks = (params?: ListDockerTasksRequest) => {
  return request.get<ListDockerTasksResponse>('/docker/tasks', { params })
}

export const getDockerTask = (params: GetDockerTaskRequest) => {
  return request.get<GetDockerTaskResponse>(`/docker/tasks/${params.taskId}`)
}

// ==================== Containers ====================

export const listDockerContainers = (params: ListDockerContainersRequest) => {
  return request.get<ListDockerContainersResponse>('/docker/containers', { params })
}

export const getDockerContainer = (params: GetDockerContainerRequest) => {
  return request.get<GetDockerContainerResponse>(`/docker/containers/${params.id}`)
}

export const createDockerContainer = (data: CreateDockerContainerRequest) => {
  return request.post<CreateDockerContainerResponse>('/docker/containers', data)
}

export const updateDockerContainer = (data: UpdateDockerContainerRequest) => {
  return request.put<UpdateDockerContainerResponse>(`/docker/containers/${data.id}`, data)
}

export const recreateDockerContainer = (data: RecreateDockerContainerRequest) => {
  return request.post<RecreateDockerContainerResponse>(`/docker/containers/${data.id}/recreate`, data)
}

export const startDockerContainer = (data: StartDockerContainerRequest) => {
  return request.post<StartDockerContainerResponse>(`/docker/containers/${data.id}/start`, data)
}

export const stopDockerContainer = (data: StopDockerContainerRequest) => {
  return request.post<StopDockerContainerResponse>(`/docker/containers/${data.id}/stop`, data)
}

export const restartDockerContainer = (data: RestartDockerContainerRequest) => {
  return request.post<RestartDockerContainerResponse>(`/docker/containers/${data.id}/restart`, data)
}

export const killDockerContainer = (data: KillDockerContainerRequest) => {
  return request.post<KillDockerContainerResponse>(`/docker/containers/${data.id}/kill`, data)
}

export const pauseDockerContainer = (data: PauseDockerContainerRequest) => {
  return request.post<PauseDockerContainerResponse>(`/docker/containers/${data.id}/pause`, data)
}

export const unpauseDockerContainer = (data: UnpauseDockerContainerRequest) => {
  return request.post<UnpauseDockerContainerResponse>(`/docker/containers/${data.id}/unpause`, data)
}

export const renameDockerContainer = (data: RenameDockerContainerRequest) => {
  return request.post<RenameDockerContainerResponse>(`/docker/containers/${data.id}/rename`, data)
}

export const deleteDockerContainer = (params: DeleteDockerContainerRequest) => {
  return request.delete<DeleteDockerContainerResponse>(`/docker/containers/${params.id}`, {
    params: { force: params.force, removeVolumes: params.removeVolumes },
  })
}

export const containerLogs = (params: ContainerLogsRequest) => {
  return request.get<ContainerLogsResponse>(`/docker/containers/${params.id}/logs`, { params })
}

export const containerStats = (params: ContainerStatsRequest) => {
  return request.get<ContainerStatsResponse>(`/docker/containers/${params.id}/stats`, { params })
}

export const createContainerExec = (data: CreateContainerExecRequest) => {
  return request.post<CreateContainerExecResponse>(`/docker/containers/${data.containerId}/exec`, data)
}

// ==================== Images ====================

export const listDockerImages = (params: ListDockerImagesRequest) => {
  return request.get<ListDockerImagesResponse>('/docker/images', { params })
}

export const getDockerImage = (params: GetDockerImageRequest) => {
  return request.get<GetDockerImageResponse>(`/docker/images/${params.id}`)
}

export const pullDockerImage = (data: PullDockerImageRequest) => {
  return request.post<PullDockerImageResponse>('/docker/images/pull', data)
}

export const buildDockerImage = (data: BuildDockerImageRequest) => {
  return request.post<BuildDockerImageResponse>('/docker/images/build', data)
}

export const updateDockerImageTags = (data: UpdateDockerImageTagsRequest) => {
  return request.put<UpdateDockerImageTagsResponse>(`/docker/images/${data.imageId}/tags`, data)
}

export const tagDockerImage = (data: TagDockerImageRequest) => {
  return request.post<TagDockerImageResponse>(`/docker/images/${data.id}/tag`, data)
}

export const deleteDockerImage = (params: DeleteDockerImageRequest) => {
  return request.delete<DeleteDockerImageResponse>(`/docker/images/${params.id}`, {
    params: { force: params.force, pruneChildren: params.pruneChildren },
  })
}

export const pruneDockerImages = (data: PruneDockerImagesRequest) => {
  return request.post<PruneDockerImagesResponse>('/docker/images/prune', data)
}

export const imageHistory = (params: ImageHistoryRequest) => {
  return request.get<ImageHistoryResponse>(`/docker/images/${params.id}/history`, { params })
}

// ==================== Networks ====================

export const listDockerNetworks = (params: ListDockerNetworksRequest) => {
  return request.get<ListDockerNetworksResponse>('/docker/networks', { params })
}

export const getDockerNetwork = (params: GetDockerNetworkRequest) => {
  return request.get<GetDockerNetworkResponse>(`/docker/networks/${params.id}`)
}

export const createDockerNetwork = (data: CreateDockerNetworkRequest) => {
  return request.post<CreateDockerNetworkResponse>('/docker/networks', data)
}

export const updateDockerNetwork = (data: UpdateDockerNetworkRequest) => {
  return request.put<UpdateDockerNetworkResponse>(`/docker/networks/${data.id}`, data)
}

export const recreateDockerNetwork = (data: RecreateDockerNetworkRequest) => {
  return request.post<RecreateDockerNetworkResponse>(`/docker/networks/${data.id}/recreate`, data)
}

export const deleteDockerNetwork = (params: DeleteDockerNetworkRequest) => {
  return request.delete<DeleteDockerNetworkResponse>(`/docker/networks/${params.id}`)
}

export const connectDockerNetwork = (data: ConnectDockerNetworkRequest) => {
  return request.post<ConnectDockerNetworkResponse>(`/docker/networks/${data.networkId}/connect`, data)
}

export const disconnectDockerNetwork = (data: DisconnectDockerNetworkRequest) => {
  return request.post<DisconnectDockerNetworkResponse>(`/docker/networks/${data.networkId}/disconnect`, data)
}

export const pruneDockerNetworks = (data: PruneDockerNetworksRequest) => {
  return request.post<PruneDockerNetworksResponse>('/docker/networks/prune', data)
}

// ==================== Volumes ====================

export const listDockerVolumes = (params: ListDockerVolumesRequest) => {
  return request.get<ListDockerVolumesResponse>('/docker/volumes', { params })
}

export const getDockerVolume = (params: GetDockerVolumeRequest) => {
  return request.get<GetDockerVolumeResponse>(`/docker/volumes/${params.name}`)
}

export const createDockerVolume = (data: CreateDockerVolumeRequest) => {
  return request.post<CreateDockerVolumeResponse>('/docker/volumes', data)
}

export const updateDockerVolume = (data: UpdateDockerVolumeRequest) => {
  return request.put<UpdateDockerVolumeResponse>(`/docker/volumes/${data.name}`, data)
}

export const recreateDockerVolume = (data: RecreateDockerVolumeRequest) => {
  return request.post<RecreateDockerVolumeResponse>(`/docker/volumes/${data.name}/recreate`, data)
}

export const deleteDockerVolume = (params: DeleteDockerVolumeRequest) => {
  return request.delete<DeleteDockerVolumeResponse>(`/docker/volumes/${params.name}`, {
    params: { force: params.force },
  })
}

export const pruneDockerVolumes = (data: PruneDockerVolumesRequest) => {
  return request.post<PruneDockerVolumesResponse>('/docker/volumes/prune', data)
}

export const exportDockerVolume = (data: ExportDockerVolumeRequest) => {
  return request.post<ExportDockerVolumeResponse>(`/docker/volumes/${data.name}/export`, data)
}

export const restoreDockerVolume = (data: RestoreDockerVolumeRequest) => {
  return request.post<RestoreDockerVolumeResponse>(`/docker/volumes/${data.name}/restore`, data)
}
