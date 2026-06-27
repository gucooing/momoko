<template>
  <div class="file-pager">
    <span class="fp-total">{{ t('fileManager.totalItems', { total }) }}</span>

    <!-- 每页数量：自绘下拉，向上弹出，避免 teleport 逃出浅色作用域 -->
    <div ref="sizeRef" class="fp-size">
      <button type="button" class="fp-size-trigger" @click="sizeOpen = !sizeOpen">
        <span>{{ t('fileManager.perPage', { size: pageSize }) }}</span>
        <el-icon class="fp-caret" :class="{ 'is-open': sizeOpen }"><IconChevronDown /></el-icon>
      </button>
      <transition name="fp-fade">
        <ul v-if="sizeOpen" class="fp-size-menu">
          <li
            v-for="size in pageSizes"
            :key="size"
            class="fp-size-item"
            :class="{ 'is-active': size === pageSize }"
            @click="selectSize(size)"
          >
            {{ t('fileManager.perPage', { size }) }}
          </li>
        </ul>
      </transition>
    </div>

    <!-- 页码 -->
    <nav class="fp-pages">
      <button type="button" class="fp-page-btn" :disabled="page <= 1" @click="goto(page - 1)">
        <el-icon><IconChevronLeft /></el-icon>
      </button>
      <template v-for="(item, index) in pageItems" :key="`${item}-${index}`">
        <span v-if="item === '...'" class="fp-ellipsis">…</span>
        <button
          v-else
          type="button"
          class="fp-page-btn"
          :class="{ 'is-active': item === page }"
          @click="goto(item as number)"
        >
          {{ item }}
        </button>
      </template>
      <button
        type="button"
        class="fp-page-btn"
        :disabled="page >= pageCount"
        @click="goto(page + 1)"
      >
        <el-icon><IconChevronRight /></el-icon>
      </button>
    </nav>

    <!-- 跳转 -->
    <div class="fp-jump">
      <span>{{ t('fileManager.goTo') }}</span>
      <input
        class="fp-jump-input"
        :value="jumpValue"
        inputmode="numeric"
        @input="onJumpInput"
        @keyup.enter="applyJump"
        @blur="applyJump"
      />
      <span>{{ t('fileManager.pageUnit') }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { IconChevronDown, IconChevronLeft, IconChevronRight } from './icons'

const props = withDefaults(
  defineProps<{
    page: number
    pageSize: number
    total: number
    pageSizes?: number[]
  }>(),
  {
    pageSizes: () => [10, 20, 50, 100],
  },
)

const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
  change: []
}>()

const { t } = useI18n()

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

// 窗口化页码：首尾 + 当前页附近，超出用省略号。
const pageItems = computed<(number | '...')[]>(() => {
  const count = pageCount.value
  const current = props.page
  if (count <= 7) {
    return Array.from({ length: count }, (_, i) => i + 1)
  }
  const items: (number | '...')[] = [1]
  const start = Math.max(2, current - 1)
  const end = Math.min(count - 1, current + 1)
  if (start > 2) items.push('...')
  for (let i = start; i <= end; i += 1) items.push(i)
  if (end < count - 1) items.push('...')
  items.push(count)
  return items
})

const goto = (target: number) => {
  const next = Math.min(Math.max(1, target), pageCount.value)
  if (next === props.page) return
  emit('update:page', next)
  emit('change')
}

// 每页数量下拉
const sizeRef = ref<HTMLElement | null>(null)
const sizeOpen = ref(false)
onClickOutside(sizeRef, () => (sizeOpen.value = false))

const selectSize = (size: number) => {
  sizeOpen.value = false
  if (size === props.pageSize) return
  emit('update:pageSize', size)
  emit('change')
}

// 跳转输入
const jumpValue = ref('')
const onJumpInput = (event: Event) => {
  jumpValue.value = (event.target as HTMLInputElement).value.replace(/\D/g, '')
}
const applyJump = () => {
  if (!jumpValue.value) return
  goto(Number(jumpValue.value))
  jumpValue.value = ''
}
</script>

<style scoped>
.file-pager {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  color: var(--fm-text-2);
  font-size: 13px;
  user-select: none;
}

.fp-total {
  color: var(--fm-text-3);
}

/* 每页数量 */
.fp-size {
  position: relative;
}
.fp-size-trigger {
  display: inline-flex;
  align-items: center;
  gap: 0.25rem;
  height: 28px;
  padding: 0 0.5rem;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
  color: var(--fm-text-2);
  cursor: pointer;
  transition: border-color 0.15s;
}
.fp-size-trigger:hover {
  border-color: var(--fm-border-strong);
}
.fp-caret {
  font-size: 14px;
  transition: transform 0.15s;
}
.fp-caret.is-open {
  transform: rotate(180deg);
}
.fp-size-menu {
  position: absolute;
  bottom: calc(100% + 4px);
  left: 0;
  min-width: 100%;
  margin: 0;
  padding: 4px;
  list-style: none;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  box-shadow: var(--fm-shadow);
  z-index: 20;
}
.fp-size-item {
  padding: 0.35rem 0.6rem;
  border-radius: var(--fm-radius-sm);
  white-space: nowrap;
  cursor: pointer;
  color: var(--fm-text-2);
}
.fp-size-item:hover {
  background: var(--fm-hover);
}
.fp-size-item.is-active {
  color: var(--fm-accent);
  font-weight: 600;
}

/* 页码 */
.fp-pages {
  display: flex;
  align-items: center;
  gap: 4px;
}
.fp-page-btn {
  min-width: 28px;
  height: 28px;
  padding: 0 0.4rem;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
  color: var(--fm-text-2);
  font-size: 13px;
  cursor: pointer;
  transition:
    border-color 0.15s,
    color 0.15s,
    background 0.15s;
}
.fp-page-btn:hover:not(:disabled):not(.is-active) {
  border-color: var(--fm-accent);
  color: var(--fm-accent);
}
.fp-page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.fp-page-btn.is-active {
  background: var(--fm-accent);
  border-color: var(--fm-accent);
  color: #fff;
  font-weight: 600;
}
.fp-ellipsis {
  min-width: 20px;
  text-align: center;
  color: var(--fm-text-3);
}

/* 跳转 */
.fp-jump {
  display: flex;
  align-items: center;
  gap: 0.35rem;
}
.fp-jump-input {
  width: 44px;
  height: 28px;
  text-align: center;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-surface);
  color: var(--fm-text);
  outline: none;
  transition: border-color 0.15s;
}
.fp-jump-input:focus {
  border-color: var(--fm-accent);
}

.fp-fade-enter-active,
.fp-fade-leave-active {
  transition:
    opacity 0.15s,
    transform 0.15s;
}
.fp-fade-enter-from,
.fp-fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}
</style>
