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
  type InstanceEnvRow,
  type InstanceRecord,
  type InstanceTypeOption,
  type QueryFormValue,
} from '@/stores/instance/types'
import { useUserStore } from '@/stores/user'
import type {
  CreateInstanceRequest,
  GetInstancesRequest,
  InstanceEnvVar,
  InstanceInfo,
  InstanceStatusCount,
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
  autoRestart: false,
  stopTimeout: 0,
  env: [],
})

const createEmptyCounts = (): InstanceStatusCount => ({ all: 0, running: 0, stopped: 0, maintenance: 0 })

const parsePositiveNumber = (value: unknown, fallback: number) => {
  const nextValue = Number(value)
  return Number.isFinite(nextValue) && nextValue > 0 ? nextValue : fallback
}

/** 接口里的变量 -> 编辑器行（密钥变量不返回值，标记为已保存） */
export const toEnvRows = (env: InstanceEnvVar[] | undefined): InstanceEnvRow[] =>
  (env || []).map((item) => ({
    key: item.key,
    value: item.isSecret ? '' : item.value,
    secret: item.isSecret,
    saved: item.isSecret,
  }))

/** 编辑器行 -> 接口变量（密钥变量的值放在 secret 字段：只写，留空保持原值） */
const toEnvVars = (rows: InstanceEnvRow[]): InstanceEnvVar[] =>
  rows
    .filter((row) => row.key.trim())
    .map((row) => ({
      key: row.key.trim(),
      value: row.secret ? '' : row.value,
      isSecret: row.secret,
      secret: row.secret ? row.value : '',
    }))

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
  autoRestart: Boolean(info.autoRestart),
  stopTimeout: Number(info.stopTimeout || 0),
  env: toEnvRows(info.env),
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
  autoRestart: form.autoRestart,
  stopTimeout: Math.max(0, Math.round(Number(form.stopTimeout) || 0)),
  env: toEnvVars(form.env),
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
  autoRestart: form.autoRestart,
  stopTimeout: Math.max(0, Math.round(Number(form.stopTimeout) || 0)),
  env: { items: toEnvVars(form.env) },
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
  // 各状态数量（按关键字/类型筛选，不按状态筛选），用于状态 chips 与页头计数
  const statusCounts = ref<InstanceStatusCount>(createEmptyCounts())

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
      .map((item) => ({
        label: item.name,
        value: item.id,
        description: item.description || '',
        startCommand: item.startCommand || '',
        stopCommand: item.stopCommand || '',
      })),
  )

  const statusOptions = computed(() =>
    (Object.keys(statusMeta) as InstanceStatus[]).map((key) => ({
      value: key,
      label: translate(statusMeta[key].labelKey),
    })),
  )

  const pagedInstances = computed(() => instanceList.value)

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

  const canBatchRestart = computed(() =>
    selectedIds.value.some(
      (id) => instanceStatusMap.value.get(id) !== InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
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
      statusCounts.value = response?.counts || createEmptyCounts()
    } finally {
      loading.value = false
    }
  }

  const getInstanceDetail = async (id: string): Promise<InstanceInfo | undefined> => {
    const { data: response } = await getInstanceInfoRequest({ id })
    return response?.info
  }

  const createInstance = async (payload: CreateInstanceRequest) => {
    const { data } = await createInstanceRequest(payload)
    await getInstanceList()
    return data?.info
  }

  const updateInstance = async (payload: UpdateInstanceRequest) => {
    const { data } = await updateInstanceRequest(payload)
    await getInstanceList()
    return data?.info
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

  /** 保存编辑器；返回保存后的应用（「保存并启动」用它的 id 启动） */
  const submitInstanceEditor = async (): Promise<InstanceInfo | undefined> => {
    if (instanceEditorSubmitting.value) return

    instanceEditorSubmitting.value = true

    try {
      let saved: InstanceInfo | undefined
      if (instanceEditorMode.value === 'create') {
        const userId = userStore.userInfo?.userId?.trim() || ''
        if (!userId) {
          throw new Error('Current user id is missing')
        }

        saved = await createInstance(buildCreatePayload(instanceEditorForm.value, userId))
      } else {
        if (!instanceEditorForm.value.id) {
          throw new Error('Instance id is missing')
        }

        saved = await updateInstance(buildUpdatePayload(instanceEditorForm.value))
      }

      closeInstanceEditor()
      return saved
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

  // 批量启动跳过维护中的应用（后端会拒绝启动）
  const batchChangeStatus = async (targetStatus: SwitchableInstanceStatus) => {
    const ids =
      targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING
        ? selectedIds.value.filter(
            (id) => instanceStatusMap.value.get(id) !== InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
          )
        : [...selectedIds.value]
    return runBatchAction(
      ids,
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

  // 批量重启：维护中的应用跳过
  const batchRestart = async () => {
    const ids = selectedIds.value.filter(
      (id) => instanceStatusMap.value.get(id) !== InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
    )
    return runBatchAction(ids, (id) => restartInstanceById(id), true)
  }

  // 进入 / 结束维护（进入维护时后端会先停止应用）
  const setMaintenance = async (id: string, maintenance: boolean) => {
    await updateInstanceRequest({ id, maintenance, env: undefined })
    await getInstanceList()
  }

  // 启动维护中的应用：先结束维护再启动
  const endMaintenanceAndStart = async (id: string) => {
    await updateInstanceRequest({ id, maintenance: false, env: undefined })
    return changeInstanceStatus(id, InstanceStatus.INSTANCE_STATUS_RUNNING)
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
    statusCounts,
    selectedIdSet,
    typeOptions,
    statusOptions,
    pagedInstances,
    isCurrentPageAllSelected,
    canBatchStart,
    canBatchStop,
    canBatchRestart,
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
    handlePageChange,
    refreshStatus,
    batchChangeStatus,
    batchDeleteInstances,
    changeInstanceStatus,
    deleteInstances,
    restartInstance,
    batchRestart,
    setMaintenance,
    endMaintenanceAndStart,
    findInstanceById,
    resolveTypeLabel,
    getInstanceTypeList,
    initialize,
  }
})
