package biz

import (
	"fmt"

	"momoko/pkg/response"
)

var (
	ErrSystem = func(err error) error {
		return response.BadRequest(501, fmt.Sprintf("系统错误,请联系管理员:%v", err))
	}
	ErrAdminNotFound    = response.BadRequest(501, "用户不存在")
	ErrInvalidPassword  = response.BadRequest(501, "密码错误")
	ErrTokenInvalid     = response.BadRequest(401, "token invalid")
	ErrUserNoRole       = response.BadRequest(501, "用户没有角色")
	ErrNoPermission     = response.BadRequest(501, "权限不足")
	ErrUserInactive     = response.BadRequest(501, "账号已停用")
	ErrInstanceNotFound = response.BadRequest(501, "实例不存在")
)
