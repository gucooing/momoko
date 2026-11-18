package server

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"momoko/internal/biz"
)

// 开放给远程节点的每个操作与原始路径，所需权限都必须能授予 API Key，否则远程永远调不通。
func TestNodePolicyPermissionsGrantable(t *testing.T) {
	checked := 0
	protoregistry.GlobalFiles.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		services := fd.Services()
		for i := 0; i < services.Len(); i++ {
			svc := services.Get(i)
			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				operation := "/" + string(svc.FullName()) + "/" + string(methods.Get(j).Name())
				perm, ok := nodeOperationPermission(operation)
				if !ok || perm == "" {
					continue
				}
				checked++
				if !biz.IsNodePermission(perm) {
					t.Errorf("%s 远程需要 %s，但 API Key 不能授予它", operation, perm)
				}
			}
		}
		return true
	})
	if checked == 0 {
		t.Fatal("没有检查到任何远程操作")
	}
	for path, perm := range nodeTunnelPaths {
		if perm != "" && !biz.IsNodePermission(perm) {
			t.Errorf("隧道路径 %s 需要 %s，但 API Key 不能授予它", path, perm)
		}
	}
	for _, tp := range nodeTunnelPrefixes {
		if !biz.IsNodePermission(tp.perm) {
			t.Errorf("隧道路径前缀 %s 需要 %s，但 API Key 不能授予它", tp.prefix, tp.perm)
		}
	}
}
