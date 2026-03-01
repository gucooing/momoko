package biz

import (
	"momoko/pkg/response"
)

var (
	// ErrAdminNotFound error admin not found.
	ErrAdminNotFound = response.BadRequest(501, "用户不存在")
)
