<template>
  <HoverAnimateWrapper name="lift">
    <BaseCard class="running-instance-card" @click="emit('open', instance)">
      <div class="instance-title">
        <div class="instance-icon" :style="{ backgroundColor: `${themeColor}20` }">
          <el-icon size="20">
            <component :is="menuStore.iconComponents['HOutline:ServerStackIcon']" :style="{ color: themeColor }" />
          </el-icon>
        </div>
        <div class="title">{{ instance.name || '未命名实例' }}</div>
        <BaseTag :text="statusText" :type="statusTagType" />
      </div>

      <div class="instance-description">
        <div class="instance-meta-line">
          <span class="instance-meta-label">类型</span>
          <TextEllipsis :text="instance.type || '-'" :line="1" />
        </div>
        <div class="instance-meta-line instance-meta-line--tags">
          <span class="instance-meta-label">标签</span>
          <div v-if="tagList.length" class="instance-tag-list">
            <BaseTag v-for="tag in tagList" :key="tag" :text="tag" />
          </div>
          <span v-else class="instance-empty-text">-</span>
        </div>
      </div>

      <div class="instance-footer">
        <div class="instance-member">
          <div
            v-for="placeholderIndex in avatarLine"
            :key="`instance-avatar-${placeholderIndex}`"
            class="avatar avatar-placeholder"
          />
        </div>
        <div class="instance-time">{{ timeText }}</div>
      </div>
    </BaseCard>
  </HoverAnimateWrapper>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'

interface RunningInstanceCardProps {
  instance: InstanceInfo
  avatarLine?: number
}

const props = withDefaults(defineProps<RunningInstanceCardProps>(), {
  avatarLine: 3,
})

const emit = defineEmits<{
  open: [instance: InstanceInfo]
}>()

const menuStore = useMenuStore()

const INSTANCE_CARD_COLORS = ['#10b981', '#4f46e5', '#f59e0b', '#ef4444', '#06b6d4', '#8b5cf6']

const hashText = (value: string) => {
  let hash = 0

  for (const char of value) {
    hash = (hash << 5) - hash + char.charCodeAt(0)
    hash |= 0
  }

  return Math.abs(hash)
}

const themeColor = computed(() => {
  const hashIndex = hashText(props.instance.id || props.instance.name || 'instance')
  return INSTANCE_CARD_COLORS[hashIndex % INSTANCE_CARD_COLORS.length] || '#10b981'
})

const tagList = computed(() =>
  (props.instance.tags || '')
    .split(',')
    .map((tag) => tag.trim())
    .filter(Boolean),
)

const statusTextMap: Record<InstanceInfo['status'], string> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: '运行中',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: '已停止',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: '维护中',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: '未指定',
  [InstanceStatus.UNRECOGNIZED]: '未知状态',
}

const statusTagTypeMap: Record<InstanceInfo['status'], 'success' | 'info' | 'warning' | 'danger'> =
  {
    [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'success',
    [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'info',
    [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'warning',
    [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'info',
    [InstanceStatus.UNRECOGNIZED]: 'danger',
  }

const statusText = computed(
  () => statusTextMap[props.instance.status] || statusTextMap[InstanceStatus.UNRECOGNIZED],
)

const statusTagType = computed(
  () => statusTagTypeMap[props.instance.status] || statusTagTypeMap[InstanceStatus.UNRECOGNIZED],
)

const timeText = computed(() => {
  const rawValue = props.instance.startTime || props.instance.createTime
  if (!rawValue) return '-'

  const parsed = dayjs(rawValue instanceof Date ? rawValue : String(rawValue))
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : String(rawValue)
})
</script>

<style scoped lang="scss">
.running-instance-card {
  height: 100%;
  background: var(--el-bg-color-page);
  padding: 0.25rem;
  cursor: pointer;

  .instance-title {
    display: flex;
    align-items: center;
    gap: 1rem;

    .instance-icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 2.5rem;
      height: 2.5rem;
      border-radius: 0.5rem;
      flex-shrink: 0;
    }

    .title {
      min-width: 0;
      font-weight: 700;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .instance-description {
    margin-top: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
    font-size: 0.875rem;
    color: var(--el-text-color-secondary);
  }

  .instance-meta-line {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;
    min-width: 0;

    :deep(.text-ellipsis) {
      flex: 1;
      min-width: 0;
    }
  }

  .instance-meta-line--tags {
    align-items: center;
  }

  .instance-meta-label {
    flex-shrink: 0;
    font-size: 0.78rem;
    font-weight: 700;
    color: var(--el-text-color-placeholder);
  }

  .instance-tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .instance-empty-text {
    color: var(--el-text-color-placeholder);
  }

  .instance-footer {
    margin-top: 1rem;
    display: flex;
    align-items: center;
    justify-content: space-between;
    color: var(--el-text-color-placeholder);
    font-size: 0.875rem;

    .instance-member {
      display: flex;
      align-items: center;
    }

    .avatar {
      width: 24px;
      height: 24px;
      border-radius: 999px;
      border: 2px solid var(--el-bg-color-page);
      margin-left: -0.625rem;

      &:first-child {
        margin-left: 0;
      }
    }

    .avatar-placeholder {
      background: color-mix(in srgb, var(--el-text-color-placeholder) 12%, var(--el-bg-color));
      border-color: var(--el-bg-color-page);
    }
  }
}
</style>
