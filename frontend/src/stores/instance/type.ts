import { defineStore } from 'pinia'
import { deleteInstanceType, getInstanceTypes } from '@/api/instance'
import type { InstanceTypeInfo } from '@/types/v1/instance'

export const useInstanceTypeStore = defineStore('instance-type', () => {
  const loading = ref(false)
  const instanceTypeList = ref<InstanceTypeInfo[]>([])

  const getInstanceTypeList = async () => {
    loading.value = true

    try {
      const { data: res } = await getInstanceTypes({})
      instanceTypeList.value = res?.types || []
    } finally {
      loading.value = false
    }
  }

  const deleteTypeById = async (id: string) => {
    await deleteInstanceType({ id })
    await getInstanceTypeList()
  }

  return {
    loading,
    instanceTypeList,
    getInstanceTypeList,
    deleteTypeById,
  }
})
