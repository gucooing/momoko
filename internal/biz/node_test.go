package biz

import (
	"errors"
	"testing"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"momoko/pkg/response"
)

func TestNormalizeNodePermissions(t *testing.T) {
	got, err := normalizeNodePermissions([]string{"docker:view", "system:status", "docker:view"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "system:status" || got[1] != "docker:view" {
		t.Fatalf("unexpected normalized permissions: %v", got)
	}
	if _, err := normalizeNodePermissions([]string{"user:delete"}); !errors.Is(err, ErrAPIKeyPermission) {
		t.Fatalf("non-node permission must be rejected, got %v", err)
	}
}

func TestNodeCallError(t *testing.T) {
	asResponse := func(err error) response.Error {
		t.Helper()
		var re response.Error
		if !errors.As(err, &re) {
			t.Fatalf("expected response.Error, got %T", err)
		}
		return re
	}

	business := kerrors.New(500, NodeErrorReason, "实例不存在").WithMetadata(map[string]string{"code": "500"})
	if re := asResponse(NodeCallError("n1", business)); re.Code != 500 || re.Message != "实例不存在" {
		t.Fatalf("business error not preserved: %+v", re)
	}

	unauthorized := kerrors.New(401, NodeErrorReason, "API Key 已过期").WithMetadata(map[string]string{"code": "401"})
	re := asResponse(NodeCallError("n1", unauthorized))
	if re.Code == 401 {
		t.Fatal("node auth failures must not surface as 401 to the browser")
	}

	offline := status.Error(codes.Unavailable, "connection refused")
	if re := asResponse(NodeCallError("n1", offline)); re.Code != 500 {
		t.Fatalf("offline node must map to 500, got %+v", re)
	}

	state, _ := classifyNodeStatus("n1", unauthorized)
	if state.String() != "NodeState_Unauthorized" {
		t.Fatalf("unexpected state %v", state)
	}
	state, _ = classifyNodeStatus("n1", offline)
	if state.String() != "NodeState_Offline" {
		t.Fatalf("unexpected state %v", state)
	}
}
