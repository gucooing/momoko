<!-- 登录日志 -->
<template>
  <BaseCard>
    <el-empty v-if="!loginLogs.length" description="暂无登录日志" />
    <div v-else>
      <el-table
        :data="loginLogs"
        :border="TABLE_CONFIG.border"
        show-overflow-tooltip
        class="custom-modern-table"
      >
        <el-table-column prop="ip" label="IP 地址" min-width="150" />
        <el-table-column prop="userAgent" label="User Agent" min-width="280" />
        <el-table-column prop="operationTime" label="操作时间" min-width="170" />
        <el-table-column label="结果" width="80">
          <template #default="{ row }">
            <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? '成功' : '失败'" />
          </template>
        </el-table-column>
        <el-table-column prop="detail" label="详情" min-width="160" />
      </el-table>

      <TablePagination
        v-model:current-page="loginLogsPagination.page"
        v-model:page-size="loginLogsPagination.pageSize"
        :total="loginLogsPagination.total"
        :is-mobile="menuStore.isMobile"
        @change="userProfileStore.getMyLoginLogs"
      />
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { useUserProfileStore } from '@/stores/user/profile'
import { TABLE_CONFIG } from '@/config/elementConfig'
import TablePagination from '@/components/pagination/TablePagination.vue'

const menuStore = useMenuStore()
const userProfileStore = useUserProfileStore()
const { loginLogs, loginLogsPagination } = storeToRefs(userProfileStore)

onMounted(() => {
  userProfileStore.getMyLoginLogs()
})
</script>

<style scoped lang="scss"></style>
