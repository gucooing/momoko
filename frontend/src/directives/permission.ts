import { watchEffect, type Directive, type DirectiveBinding } from 'vue'

type PermissionValue = string | string[]
type PermissionElement = HTMLElement & {
  __permissionStop?: ReturnType<typeof watchEffect>
  __permissionValue?: PermissionValue
}

const checkPermission = (value?: PermissionValue): boolean => {
  if (!value) return false
  if (Array.isArray(value) && value.length === 0) return false

  const menuStore = useMenuStore()
  return menuStore.hasButtonPermission(value)
}

const applyPermission = (el: PermissionElement) => {
  el.style.display = checkPermission(el.__permissionValue) ? '' : 'none'
}

export const permissionDirective: Directive<PermissionElement, PermissionValue> = {
  mounted(el, binding: DirectiveBinding<PermissionValue>) {
    el.__permissionValue = binding.value
    el.__permissionStop = watchEffect(() => applyPermission(el))
  },
  updated(el, binding: DirectiveBinding<PermissionValue>) {
    el.__permissionValue = binding.value
    applyPermission(el)
  },
  unmounted(el) {
    el.__permissionStop?.()
    delete el.__permissionStop
    delete el.__permissionValue
  },
}
