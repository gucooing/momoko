package server

import (
	"testing"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/constant"
)

func TestNodeOperationPermission(t *testing.T) {
	cases := []struct {
		operation string
		perm      constant.Permissions
		remote    bool
		scoped    bool
	}{
		{v1.OperationSystemSystemStatus, constant.SystemStatus, true, false},
		{v1.OperationSystemAdminRoles, "", false, false},
		{v1.OperationDockerManagerListDockerContainers, constant.DockerView, true, true},
		{v1.OperationDockerManagerStopDockerContainer, constant.DockerContainerManage, true, true},
		{v1.OperationFileManagerGetFileSystemList, constant.Terminal, true, true},
		{v1.OperationFileManagerCreateFileSource, constant.FileSource, true, true},
		{v1.OperationFileManagerCreateShare, "", false, true},
		{v1.OperationOpenSSHManagerGetSSHHosts, constant.SSHManage, true, true},
		{v1.OperationOpenSSHManagerShareSSHHost, "", false, true},
		{v1.OperationInstanceManagerStartInstance, constant.Instance, true, true},
		{v1.OperationTaskManagerListTasks, constant.TaskManage, true, true},
		{v1.OperationUserServiceListUser, "", false, false},
		{v1.OperationNodeServiceListNodes, "", false, false},
		{"/v1.NodeAgent/Tunnel", "", true, false},
	}
	for _, c := range cases {
		perm, remote := nodeOperationPermission(c.operation)
		if remote != c.remote || perm != c.perm {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", c.operation, perm, remote, c.perm, c.remote)
		}
		if scoped := isNodeScopedOperation(c.operation); scoped != c.scoped {
			t.Errorf("%s: scoped = %v, want %v", c.operation, scoped, c.scoped)
		}
	}
}

func TestNodeProxyReplyType(t *testing.T) {
	p := &NodeProxy{}
	reply, err := p.newReply(v1.OperationDockerManagerListDockerContainers)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reply.(*v1.ListDockerContainersResponse); !ok {
		t.Fatalf("unexpected reply type %T", reply)
	}
	// 第二次走缓存，仍需返回新实例
	again, _ := p.newReply(v1.OperationDockerManagerListDockerContainers)
	if again == reply {
		t.Fatal("reply messages must not be shared between calls")
	}
	if _, err := p.newReply("/v1.Nope/Missing"); err == nil {
		t.Fatal("unknown operation must fail")
	}
}
