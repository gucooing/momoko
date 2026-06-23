import { defineStore } from 'pinia'
import {
  createInstanceRequest,
  deleteInstanceRequest,
  getInstanceInfoRequest,
  getInstances,
  getInstanceTypes,
  restartInstanceRequest,
  startInstanceRequest,
  stopInstanceRequest,
  updateInstanceRequest,
} from '@/api/instance'
import {
  InstanceStatus,
  statusMeta,
  type InstanceEditorFormValue,
  type InstanceEditorMode,
  type InstanceRecord,
  type InstanceTypeOption,
  type OverviewCardItem,
  type QueryFormValue,
} from '@/stores/instance/types'
import { useUserStore } from '@/stores/user'
import type {
  CreateInstanceRequest,
  GetInstancesRequest,
  InstanceInfo,
  InstanceTypeInfo,
  UpdateInstanceRequest,
} from '@/types/v1/instance'
import { translate } from '@/locales'

type SwitchableInstanceStatus =
  | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
  | typeof InstanceStatus.INSTANCE_STATUS_STOPPED

interface BatchActionResult {
  successCount: number
  failedCount: number
}

const createDefaultQueryForm = (): QueryFormValue => ({
  keyword: '',
  type: '',
  status: '',
})

const createDefaultEditorForm = (defaultType = ''): InstanceEditorFormValue => ({
  id: '',
  name: '',
  remark: '',
  tags: '',
  type: defaultType,
  startCommand: '',
  stopCommand: 'exit',
  instancePath: './servers/',
  autoStart: false,
  envText: '',
})

const parsePositiveNumber = (value: unknown, fallback: number) => {
  const nextValue = Number(value)
  return Number.isFinite(nextValue) && nextValue > 0 ? nextValue : fallback
}

const parseEnvLines = (value: string) => {
  return value
    .split(/\r?\n/u)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)
}

const joinEnvLines = (envList: string[] | undefined) => {
  return (envList || []).join('\n')
}

const mapInfoToEditorForm = (info: InstanceInfo, resolveTypeId: (value: string) => string): InstanceEditorFormValue => ({
  id: info.id,
  name: info.name,
  remark: info.remark || '',
  tags: info.tags || '',
  type: resolveTypeId(info.type || ''),
  startCommand: info.startCommand || '',
  stopCommand: info.stopCommand || '',
  instancePath: info.instancePath || '',
  autoStart: Boolean(info.autoStart),
  envText: joinEnvLines(info.env),
})

const buildCreatePayload = (form: InstanceEditorFormValue, userId: string): CreateInstanceRequest => ({
  name: form.name.trim(),
  remark: form.remark.trim(),
  tags: form.tags.trim(),
  type: form.type.trim(),
  userId,
  startCommand: form.startCommand.trim(),
  instancePath: form.instancePath.trim(),
  stopCommand: form.stopCommand.trim(),
  autoStart: form.autoStart,
  env: parseEnvLines(form.envText),
})

const buildUpdatePayload = (
  form: InstanceEditorFormValue,
): UpdateInstanceRequest => ({
  id: form.id,
  name: form.name.trim(),
  remark: form.remark.trim(),
  tags: form.tags.trim(),
  type: form.type.trim(),
  startCommand: form.startCommand.trim(),
  instancePath: form.instancePath.trim(),
  stopCommand: form.stopCommand.trim(),
  autoStart: form.autoStart,
  env: parseEnvLines(form.envText),
})

export const useInstanceListStore = defineStore('instance-list', () => {
  const loading = ref(false)
  const instanceList = ref<InstanceRecord[]>([])
  const queryForm = ref<QueryFormValue>(createDefaultQueryForm())
  const appliedQuery = ref<QueryFormValue>({ ...queryForm.value })
  const selectedIds = ref<string[]>([])
  const instanceTypeList = ref<InstanceTypeInfo[]>([])
  const pagination = ref({
    page: 1,
    pageSize: 9,
    total: 0,
  })

  const instanceEditorVisible = ref(false)
  const instanceEditorMode = ref<InstanceEditorMode>('create')
  const instanceEditorLoading = ref(false)
  const instanceEditorSubmitting = ref(false)
  const instanceEditorForm = ref<InstanceEditorFormValue>(createDefaultEditorForm())

  const userStore = useUserStore()

  const selectedIdSet = computed(() => new Set(selectedIds.value))

  const typeLabelMap = computed(
    () => new Map(instanceTypeList.value.map((item) => [item.id, item.name] as const)),
  )

  const typeValueMap = computed(
    () => new Map(instanceTypeList.value.map((item) => [item.name, item.id] as const)),
  )

  const typeOptions = computed<InstanceTypeOption[]>(() =>
    instanceTypeList.value
      .filter((item) => item.isEnable)
      .map((item) => ({ label: item.name, value: item.id })),
  )

  const statusOptions = computed(() =>
    (Object.keys(statusMeta) as InstanceStatus[]).map((key) => ({
      value: key,
      label: translate(statusMeta[key].labelKey),
    })),
  )

  const pagedInstances = computed(() => instanceList.value)

  const gridTransitionKey = computed(
    () => pagedInstances.value.map((item) => item.id).join('|') || 'instance-empty',
  )

  const instanceStatusMap = computed(
    () => new Map(instanceList.value.map((item) => [item.id, item.status] as const)),
  )

  const instanceOwnerMap = computed(
    () => new Map(instanceList.value.map((item) => [item.id, item.userId] as const)),
  )

  const currentUserId = computed(() => userStore.userInfo?.userId?.trim() || '')

  const resolveTypeId = (value: string) => {
    const normalizedValue = value.trim()
    if (!normalizedValue) return ''

    if (typeLabelMap.value.has(normalizedValue)) {
      return normalizedValue
    }

    return typeValueMap.value.get(normalizedValue) || normalizedValue
  }

  const resolveTypeLabel = (value: string) => {
    const normalizedValue = value.trim()
    if (!normalizedValue) return '-'

    return typeLabelMap.value.get(normalizedValue) || normalizedValue
  }

  const isCurrentPageAllSelected = computed(
    () =>
      pagedInstances.value.length > 0 &&
      pagedInstances.value.every((item) => selectedIdSet.value.has(item.id)),
  )

  const summaryCards = computed<OverviewCardItem[]>(() => {
    let runningCount = 0
    let stoppedCount = 0

    for (const item of pagedInstances.value) {
      if (item.status === InstanceStatus.INSTANCE_STATUS_RUNNING) runningCount += 1
      if (item.status === InstanceStatus.INSTANCE_STATUS_STOPPED) stoppedCount += 1
    }

    return [
      {
        label: translate('instance.totalInstances'),
        value: pagination.value.total,
        note: translate('instance.currentFilterResult'),
        icon: 'HOutline:ServerStackIcon',
        skin: 'tone-a',
      },
      {
        label: translate('instance.currentRunning'),
        value: runningCount,
        note: translate('instance.currentPageStats'),
        icon: 'HSolid:PlayCircleIcon',
        skin: 'tone-b',
      },
      {
        label: translate('instance.stoppedSummary'),
        value: stoppedCount,
        note: translate('instance.currentPageStats'),
        icon: 'HSolid:StopCircleIcon',
        skin: 'tone-c',
      },
    ]
  })

  const canBatchStart = computed(() =>
    selectedIds.value.some((id) => {
      const status = instanceStatusMap.value.get(id)
      return (
        status === InstanceStatus.INSTANCE_STATUS_STOPPED ||
        status === InstanceStatus.INSTANCE_STATUS_UNSPECIFIED ||
        status === InstanceStatus.UNRECOGNIZED
      )
    }),
  )

  const canBatchStop = computed(() =>
    selectedIds.value.some(
      (id) => instanceStatusMap.value.get(id) === InstanceStatus.INSTANCE_STATUS_RUNNING,
    ),
  )

  const isOwnedInstance = (id: string) => {
    const ownerId = instanceOwnerMap.value.get(id)?.trim() || ''
    return !!ownerId && ownerId === currentUserId.value
  }

  const deletableSelectedIds = computed(() => selectedIds.value.filter((id) => isOwnedInstance(id)))

  const canBatchDelete = computed(() => deletableSelectedIds.value.length > 0)

  const clearSelection = () => {
    selectedIds.value = []
  }

  const setSelection = (id: string, checked: boolean) => {
    const nextSet = new Set(selectedIds.value)
    if (checked) nextSet.add(id)
    else nextSet.delete(id)
    selectedIds.value = Array.from(nextSet)
  }

  const toggleCurrentPageSelection = () => {
    if (isCurrentPageAllSelected.value) {
      const currentIds = new Set(pagedInstances.value.map((item) => item.id))
      selectedIds.value = selectedIds.value.filter((id) => !currentIds.has(id))
      return
    }

    const merged = new Set(selectedIds.value)
    pagedInstances.value.forEach((item) => merged.add(item.id))
    selectedIds.value = Array.from(merged)
  }

  const getInstanceTypeList = async () => {
    const { data: response } = await getInstanceTypes({})
    instanceTypeList.value = response?.types || []

    const enabledTypeSet = new Set(typeOptions.value.map((item) => item.value))
    const normalizedQueryType = resolveTypeId(queryForm.value.type)
    const normalizedAppliedType = resolveTypeId(appliedQuery.value.type)
    const normalizedEditorType = resolveTypeId(instanceEditorForm.value.type)

    queryForm.value.type = enabledTypeSet.has(normalizedQueryType) ? normalizedQueryType : ''

    appliedQuery.value.type = enabledTypeSet.has(normalizedAppliedType) ? normalizedAppliedType : ''

    if (enabledTypeSet.has(normalizedEditorType)) {
      instanceEditorForm.value.type = normalizedEditorType
    } else if (!instanceEditorForm.value.type) {
      instanceEditorForm.value.type = resolveDefaultType()
    }
  }

  const buildListQuery = (): GetInstancesRequest => ({
    page: pagination.value.page,
    pageSize: pagination.value.pageSize,
    keywords: appliedQuery.value.keyword.trim() || undefined,
    type: appliedQuery.value.type || undefined,
    status: appliedQuery.value.status || undefined,
  })

  const getInstanceList = async () => {
    loading.value = true

    try {
      const { data: response } = await getInstances(buildListQuery())
      instanceList.value = response?.infos || []
      pagination.value.page = parsePositiveNumber(response?.page, pagination.value.page)
      pagination.value.pageSize = parsePositiveNumber(response?.pageSize, pagination.value.pageSize)
      pagination.value.total = Math.max(0, Number(response?.total || 0))
    } finally {
      loading.value = false
    }
  }

  const getInstanceDetail = async (id: string): Promise<InstanceInfo | undefined> => {
    const { data: response } = await getInstanceInfoRequest({ id })
    return response?.info
  }

  const createInstance = async (payload: CreateInstanceRequest) => {
    await createInstanceRequest(payload)
    await getInstanceList()
  }

  const updateInstance = async (payload: UpdateInstanceRequest) => {
    await updateInstanceRequest(payload)
    await getInstanceList()
  }

  const startInstanceById = (id: string) => startInstanceRequest({ id })
  const stopInstanceById = (id: string, force = false) => stopInstanceRequest({ id, force })
  const restartInstanceById = (id: string, force = false) => restartInstanceRequest({ id, force })
  const deleteInstanceById = (id: string) => deleteInstanceRequest({ id })

  const resolveDefaultType = () => typeOptions.value[0]?.value || ''

  const openCreateEditor = () => {
    instanceEditorMode.value = 'create'
    instanceEditorLoading.value = false
    instanceEditorSubmitting.value = false
    instanceEditorForm.value = createDefaultEditorForm(resolveDefaultType())
    instanceEditorVisible.value = true
  }

  const openEditEditor = async (id: string) => {
    instanceEditorMode.value = 'edit'
    instanceEditorVisible.value = true
    instanceEditorSubmitting.value = false
    instanceEditorLoading.value = true

    try {
      const detail = await getInstanceDetail(id)
      if (!detail) throw new Error('Instance detail is empty')
      instanceEditorForm.value = mapInfoToEditorForm(detail, resolveTypeId)
    } catch (error) {
      instanceEditorVisible.value = false
      throw error
    } finally {
      instanceEditorLoading.value = false
    }
  }

  const closeInstanceEditor = () => {
    instanceEditorVisible.value = false
    instanceEditorMode.value = 'create'
    instanceEditorLoading.value = false
    instanceEditorSubmitting.value = false
    instanceEditorForm.value = createDefaultEditorForm(resolveDefaultType())
  }

  const submitInstanceEditor = async () => {
    if (instanceEditorSubmitting.value) return

    instanceEditorSubmitting.value = true

    try {
      if (instanceEditorMode.value === 'create') {
        const userId = userStore.userInfo?.userId?.trim() || ''
        if (!userId) {
          throw new Error('Current user id is missing')
        }

        await createInstance(buildCreatePayload(instanceEditorForm.value, userId))
      } else {
        if (!instanceEditorForm.value.id) {
          throw new Error('Instance id is missing')
        }

        await updateInstance(buildUpdatePayload(instanceEditorForm.value))
      }

      closeInstanceEditor()
    } finally {
      instanceEditorSubmitting.value = false
    }
  }

  const applyFilters = async () => {
    appliedQuery.value = { ...queryForm.value }
    pagination.value.page = 1
    clearSelection()
    await getInstanceList()
  }

  const resetFilters = async () => {
    queryForm.value = createDefaultQueryForm()
    appliedQuery.value = { ...queryForm.value }
    pagination.value.page = 1
    clearSelection()
    await getInstanceList()
  }

  const handlePageChange = async () => {
    clearSelection()
    await getInstanceList()
  }

  const runBatchAction = async (
    ids: string[],
    requestAction: (id: string) => Promise<unknown>,
    syncFailedSelection = false,
  ): Promise<BatchActionResult> => {
    if (!ids.length) {
      return {
        successCount: 0,
        failedCount: 0,
      }
    }

    const results = await Promise.allSettled(ids.map((id) => requestAction(id)))

    const successCount = results.filter((item) => item.status === 'fulfilled').length
    const failedIds = ids.filter((_, index) => results[index]?.status === 'rejected')

    if (syncFailedSelection) {
      selectedIds.value = failedIds
    }

    if (successCount > 0) {
      await getInstanceList()
    }

    return {
      successCount,
      failedCount: results.length - successCount,
    }
  }

  const refreshStatus = async () => {
    await getInstanceList()
  }

  const batchChangeStatus = async (targetStatus: SwitchableInstanceStatus) => {
    return runBatchAction(
      [...selectedIds.value],
      targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING
        ? (id) => startInstanceById(id)
        : (id) => stopInstanceById(id),
      true,
    )
  }

  const changeInstanceStatus = async (id: string, targetStatus: SwitchableInstanceStatus) => {
    const { successCount } = await runBatchAction(
      [id],
      targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING
        ? (nextId) => startInstanceById(nextId)
        : (nextId) => stopInstanceById(nextId),
    )

    return successCount > 0
  }

  const deleteInstances = async (ids: string[]) => {
    const deletableIds = ids.filter((id) => isOwnedInstance(id))
    return runBatchAction(deletableIds, (id) => deleteInstanceById(id))
  }

  const batchDeleteInstances = async () => {
    return runBatchAction([...deletableSelectedIds.value], (id) => deleteInstanceById(id), true)
  }

  const restartInstance = async (id: string) => {
    const { successCount } = await runBatchAction([id], (nextId) => restartInstanceById(nextId))
    return successCount > 0
  }

  const findInstanceById = (id: string) => {
    return instanceList.value.find((item) => item.id === id)
  }

  const initialize = async () => {
    await getInstanceTypeList()
    await getInstanceList()
  }

  return {
    loading,
    queryForm,
    selectedIds,
    pagination,
    selectedIdSet,
    typeOptions,
    statusOptions,
    pagedInstances,
    gridTransitionKey,
    isCurrentPageAllSelected,
    summaryCards,
    canBatchStart,
    canBatchStop,
    canBatchDelete,
    instanceEditorVisible,
    instanceEditorMode,
    instanceEditorLoading,
    instanceEditorSubmitting,
    instanceEditorForm,
    setSelection,
    toggleCurrentPageSelection,
    openCreateEditor,
    openEditEditor,
    closeInstanceEditor,
    submitInstanceEditor,
    applyFilters,
    resetFilters,
    handlePageChange,
    refreshStatus,
    batchChangeStatus,
    batchDeleteInstances,
    changeInstanceStatus,
    deleteInstances,
    restartInstance,
    findInstanceById,
    resolveTypeLabel,
    getInstanceTypeList,
    initialize,
  }
})
