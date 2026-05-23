package biz

import (
	"fmt"

	"momoko/pkg/response"
)

var (
	ErrSystem = func(err error) error {
		return response.BadRequest(500, fmt.Sprintf("系统错误:%v", err))
	}
	ErrAdminNotFound         = response.BadRequest(500, "用户不存在")
	ErrInvalidPassword       = response.BadRequest(500, "密码错误")
	ErrTokenInvalid          = response.BadRequest(401, "token invalid")
	ErrUserNoRole            = response.BadRequest(500, "用户没有角色")
	ErrNoPermission          = response.BadRequest(500, "权限不足")
	ErrUserInactive          = response.BadRequest(500, "账号已停用")
	ErrRegisterDisabled      = response.BadRequest(403, "注册已关闭")
	ErrUsernameLoginDisabled = response.BadRequest(403, "用户名登录已关闭")
	ErrEmailLoginDisabled    = response.BadRequest(403, "邮箱登录已关闭")
	ErrInstanceNotFound      = response.BadRequest(500, "实例不存在")
	ErrInstanceAccess        = response.BadRequest(500, "您没有该实例权限")
	ErrInstanceTypeName      = response.BadRequest(500, "类型名称不能为空")
	ErrInstanceTypeID        = response.BadRequest(500, "类型ID不能为空")
	ErrSSHHostInvalid        = response.BadRequest(400, "SSH服务端参数无效")
	ErrSSHAuthInvalid        = response.BadRequest(400, "SSH认证方式无效")
	ErrSSHCredentialInvalid  = response.BadRequest(400, "SSH凭据不能为空")
	ErrSSHHostAccess         = response.BadRequest(500, "您没有该SSH服务端权限")
	ErrFileNotExist          = response.BadRequest(500, "文件不存在")
	ErrFileTaskNotFound      = response.BadRequest(404, "文件任务不存在")
	ErrSign                  = response.BadRequest(500, "签名失败")
	ErrUploadRequestInvalid  = response.BadRequest(400, "上传请求参数无效")
	ErrUploadSessionNotFound = response.BadRequest(404, "上传会话不存在")
	ErrUploadSessionConflict = response.BadRequest(400, "上传会话与当前文件不匹配")
	ErrUploadPartInvalid     = response.BadRequest(400, "上传分片参数无效")
	ErrUploadIncomplete      = response.BadRequest(400, "文件分片尚未全部上传完成")
	ErrUploadCompleted       = response.BadRequest(400, "上传已完成")
)
