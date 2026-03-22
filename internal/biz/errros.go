package biz

import (
	"fmt"

	"momoko/pkg/response"
)

var (
	ErrSystem = func(err error) error {
		return response.BadRequest(500, fmt.Sprintf("系统错误,请联系管理员:%v", err))
	}
	ErrAdminNotFound    = response.BadRequest(500, "用户不存在")
	ErrInvalidPassword  = response.BadRequest(500, "密码错误")
	ErrTokenInvalid     = response.BadRequest(401, "token invalid")
	ErrUserNoRole       = response.BadRequest(500, "用户没有角色")
	ErrNoPermission     = response.BadRequest(500, "权限不足")
	ErrUserInactive     = response.BadRequest(500, "账号已停用")
	ErrInstanceNotFound = response.BadRequest(500, "实例不存在")
	ErrInstanceAccess   = response.BadRequest(500, "您没有该实例权限")
	ErrInstanceTypeName = response.BadRequest(500, "类型名称不能为空")
	ErrInstanceTypeID   = response.BadRequest(500, "类型ID不能为空")
)
