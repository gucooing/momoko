package biz

import (
	"fmt"
	"time"

	"momoko/pkg/nodeauth"
	"momoko/pkg/response"
)

var (
	ErrSystem = func(err error) error {
		return response.BadRequest(500, fmt.Sprintf("系统错误:%v", err))
	}
	ErrAdminNotFound             = response.BadRequest(500, "用户不存在")
	ErrInvalidPassword           = response.BadRequest(500, "密码错误")
	ErrTokenInvalid              = response.BadRequest(401, "token invalid")
	ErrUserNoRole                = response.BadRequest(500, "用户没有角色")
	ErrNoPermission              = response.BadRequest(500, "权限不足")
	ErrUserInactive              = response.BadRequest(500, "账号已停用")
	ErrRegisterDisabled          = response.BadRequest(500, "注册已关闭")
	ErrUsernameLoginDisabled     = response.BadRequest(500, "用户名登录已关闭")
	ErrEmailLoginDisabled        = response.BadRequest(500, "邮箱登录已关闭")
	ErrNotLoginType              = response.BadRequest(500, "不可用的登录方式")
	ErrNotDeviceID               = response.BadRequest(500, "device_id 不能为空")
	ErrEmailNot                  = response.BadRequest(500, "邮箱未启用")
	ErrEmailNotFound             = response.BadRequest(500, "邮件模板未配置")
	ErrEmailInvalid              = response.BadRequest(500, "邮箱格式不正确")
	ErrEmailRegistered           = response.BadRequest(500, "邮箱已注册")
	ErrUsernameEmpty             = response.BadRequest(500, "用户名不能为空")
	ErrPasswordEmpty             = response.BadRequest(500, "密码不能为空")
	ErrEmailEmpty                = response.BadRequest(500, "邮箱不能为空")
	ErrUsernameRegistered        = response.BadRequest(500, "用户名已注册")
	ErrUsernameEmailRegistered   = response.BadRequest(500, "用户名或邮箱已注册")
	ErrEmailCodeInvalid          = response.BadRequest(500, "邮箱验证码错误或已过期")
	ErrEmailTemplateType         = response.BadRequest(500, "邮件模板类型不正确")
	ErrEmailTemplateSubject      = response.BadRequest(500, "邮件主题不能为空")
	ErrEmailTemplateContent      = response.BadRequest(500, "邮件模板内容不能为空")
	ErrInstanceNotFound          = response.BadRequest(500, "实例不存在")
	ErrInstanceAccess            = response.BadRequest(500, "您没有该实例权限")
	ErrInstanceTypeName          = response.BadRequest(500, "类型名称不能为空")
	ErrInstanceTypeID            = response.BadRequest(500, "类型ID不能为空")
	ErrSSHHostInvalid            = response.BadRequest(500, "SSH服务端参数无效")
	ErrSSHAuthInvalid            = response.BadRequest(500, "SSH认证方式无效")
	ErrSSHCredentialInvalid      = response.BadRequest(500, "SSH凭据不能为空")
	ErrSSHStoredCredentialBroken = response.BadRequest(400, "已存凭据无法解密（密钥变更或数据损坏）。请在编辑页重新填写密码/私钥后再测试并保存。")
	ErrSSHHostAccess             = response.BadRequest(500, "您没有该SSH服务端权限")
	ErrAPIKeyNameEmpty           = response.BadRequest(500, "API Key名称不能为空")
	ErrAPIKeyNotFound            = response.BadRequest(500, "API Key不存在")
	ErrAPIKeyGenerate            = response.BadRequest(500, "API Key生成失败")
	ErrAPIKeyPermission          = response.BadRequest(500, "API Key 权限标识不合法")
	ErrFileNotExist              = response.BadRequest(500, "文件不存在")
	ErrFileTaskNotFound          = response.BadRequest(500, "文件任务不存在")
	ErrSign                      = response.BadRequest(500, "签名失败")
	ErrUploadRequestInvalid      = response.BadRequest(500, "上传请求参数无效")
	ErrShareNotFound             = response.BadRequest(500, "分享不存在或已失效")
	ErrShareForbidden            = response.BadRequest(500, "提取码错误或分享不可用")
	ErrUploadSessionNotFound     = response.BadRequest(500, "上传会话不存在")
	ErrUploadSessionConflict     = response.BadRequest(500, "上传会话与当前文件不匹配")
	ErrUploadPartInvalid         = response.BadRequest(500, "上传分片参数无效")
	ErrUploadIncomplete          = response.BadRequest(500, "文件分片尚未全部上传完成")
	ErrUploadCompleted           = response.BadRequest(500, "上传已完成")
	ErrInitializeDone            = response.BadRequest(500, "系统已初始化")
	ErrInitializeDatabaseEmpty   = response.BadRequest(500, "数据库配置不能为空")
	ErrInitializeAdminEmpty      = response.BadRequest(500, "超级管理员不能为空")
	ErrFileSourceNotFound        = response.BadRequest(500, "文件来源不存在")
	ErrFileSourceDisabled        = response.BadRequest(500, "文件来源已停用")
	ErrFileSourceUnsupported     = response.BadRequest(500, "当前文件来源不支持该操作")
	ErrFileSourceType            = response.BadRequest(500, "文件来源类型不合法")
	ErrShareSignInvalid          = response.BadRequest(500, "分享会话已失效，请重新验证")
	ErrTaskNotFound              = response.BadRequest(500, "任务不存在")
	ErrOIDCClientName            = response.BadRequest(500, "OIDC 客户端名称不能为空")
	ErrOIDCRedirectURI           = response.BadRequest(500, "OIDC 回调地址不正确")
	ErrOIDCClientNotFound        = response.BadRequest(500, "OIDC 客户端不存在")
)

// 智商检测
var (
	ErrIQCheckPromptTitle    = response.BadRequest(500, "提示词标题不能为空")
	ErrIQCheckPromptContent  = response.BadRequest(500, "提示词内容不能为空")
	ErrIQCheckPromptNotFound = response.BadRequest(500, "提示词不存在")
	ErrIQCheckSiteName       = response.BadRequest(500, "站点名称不能为空")
	ErrIQCheckSiteAPIKey     = response.BadRequest(500, "API Key 不能为空")
	ErrIQCheckSiteProtocol   = response.BadRequest(500, "协议不合法")
	ErrIQCheckSiteModels     = response.BadRequest(500, "请先勾选需要检测的模型")
	ErrIQCheckSiteNotFound   = response.BadRequest(500, "检测站点不存在")
	ErrIQCheckEffort         = response.BadRequest(500, "推理强度不合法")
	ErrIQCheckRunning        = response.BadRequest(500, "该站点正在检测中，请稍后再试")
	ErrIQCheckNoPrompt       = response.BadRequest(500, "没有启用的检测提示词，请先在提示词配置中添加")
	ErrIQCheckPublicNotFound = response.BadRequest(404, "检测公示页不存在")
	ErrIQCheckEndpoint       = func(err error) error {
		return response.BadRequest(500, fmt.Sprintf("站点地址无效：%v", err))
	}
	ErrIQCheckUpstream = func(err error) error {
		return response.BadRequest(500, fmt.Sprintf("请求上游失败：%v", err))
	}
)

// 多节点
var (
	ErrNodeNameEmpty            = response.BadRequest(500, "节点名称不能为空")
	ErrNodeAddressInvalid       = response.BadRequest(500, "节点地址格式应为 host:port")
	ErrNodeAPIKeyEmpty          = response.BadRequest(500, "节点 API Key 不能为空")
	ErrNodeNotFound             = response.BadRequest(500, "节点不存在")
	ErrNodeKeyBroken            = response.BadRequest(500, "节点 API Key 无法解密（主密钥已变更），请编辑节点重新填写")
	ErrNodeOperationUnsupported = response.BadRequest(500, "该功能暂不支持在远程节点上使用")
	ErrNodeOperationForbidden   = response.BadRequest(500, "该操作不对远程主控开放")
	// 以下鉴权错误由节点返回给主控；主控转发给浏览器前会改写状态码，避免触发主控登录态失效。
	ErrNodeAuthFailed          = response.BadRequest(401, "API Key 签名校验失败")
	ErrNodeAPIKeyExpired       = response.BadRequest(401, "API Key 已过期")
	ErrNodeAPIKeyOwnerInactive = response.BadRequest(401, "API Key 所属用户不存在或已停用")
	ErrNodeRequestReplay       = response.BadRequest(401, "请求已被使用（疑似重放）")
	ErrNodeClockSkew           = func(now time.Time) error {
		return response.BadRequest(401, fmt.Sprintf("请求时间与节点时间偏差超过 %s，请同步系统时间（节点时间 %s）", nodeauth.MaxClockSkew, now.Format(time.DateTime)))
	}
	ErrNodePermissionDenied = func(perm string) error {
		return response.BadRequest(500, fmt.Sprintf("API Key 未授予该节点权限：%s", perm))
	}
)
