import type { Directive, DirectiveBinding } from 'vue'

const checkPermission = (value: string | string[]): boolean => {
  if (!value) return false

  const menuStore = useMenuStore()
  return menuStore.hasButtonPermission(value)
}

const applyPermission = (el: HTMLElement, binding: DirectiveBinding<string | string[]>) => {
  el.style.display = checkPermission(binding.value) ? '' : 'none'
}

export const permissionDirective: Directive<HTMLElement, string | string[]> = {
  mounted: applyPermission,
  updated: applyPermission,
}
