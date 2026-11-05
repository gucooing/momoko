//go:build !linux && !(windows && (amd64 || arm64))

package sysmon

// 其它平台只走 nvidia-smi。
func platformGPUSources() []gpuSource { return nil }
