export const normalizeAuthRedirect = (value: unknown) => {
  if (typeof value !== 'string') return ''

  const nextValue = value.trim()
  if (!nextValue) return ''
  if (!nextValue.startsWith('/')) return ''
  if (nextValue.startsWith('//')) return ''
  if (
    nextValue === '/login' ||
    nextValue.startsWith('/login?') ||
    nextValue.startsWith('/login#')
  ) {
    return ''
  }

  return nextValue
}

export const buildLoginRoute = (redirectPath?: string) => {
  const safeRedirect = normalizeAuthRedirect(redirectPath)

  if (!safeRedirect) {
    return { path: '/login' }
  }

  return {
    path: '/login',
    query: {
      redirect: safeRedirect,
    },
  }
}
