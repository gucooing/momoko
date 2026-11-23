//go:build dev

package main

// devBuild 开发构建（-tags dev，前端走 vite）：不起 Windows 守护进程，方便调试器直接挂在服务进程上。
const devBuild = true
