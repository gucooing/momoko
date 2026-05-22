<template>
  <div class="pagination-container">
    <el-pagination
      class="table-pager"
      :current-page="currentPage"
      :page-size="pageSize"
      :total="total"
      background
      size="small"
      :page-sizes="resolvedPageSizes"
      :layout="isMobile ? mobileLayout : desktopLayout"
      :pager-count="
        isMobile ? PAGINATION_CONFIG.mobilePagerCount : PAGINATION_CONFIG.desktopPagerCount
      "
      @update:current-page="onCurrentPageUpdate"
      @update:page-size="onPageSizeUpdate"
      @change="onPageChange"
    />
  </div>
</template>

<script setup lang="ts">
import { PAGINATION_CONFIG } from '@/config/elementConfig'

const props = defineProps<{
  currentPage: number
  pageSize: number
  total: number
  isMobile: boolean
  pageSizes?: number[]
}>()

const emit = defineEmits<{
  'update:currentPage': [value: number]
  'update:pageSize': [value: number]
  change: []
}>()

const desktopLayout = PAGINATION_CONFIG.desktopLayout
const mobileLayout = PAGINATION_CONFIG.mobileLayout
const resolvedPageSizes = computed(() =>
  props.pageSizes?.length ? props.pageSizes : PAGINATION_CONFIG.pageSizes,
)

const onCurrentPageUpdate = (value: number) => {
  if (value === props.currentPage) return
  emit('update:currentPage', value)
}

const onPageSizeUpdate = (value: number) => {
  if (value === props.pageSize) return
  emit('update:pageSize', value)
}

const onPageChange = () => {
  emit('change')
}
</script>

<style lang="scss" scoped>
.table-pager {
  margin-left: auto;
}
</style>
