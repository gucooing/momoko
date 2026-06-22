package sub2api

import (
	"encoding/json"
	"errors"
)

// Imagine 调用 sub2api 所需的接口路径常量。
const (
	imagineKeysPath        = "/api/v1/keys"
	imagineModelsPath      = "/v1/models"
	imagineGenerationsPath = "/v1/images/generations"
	imagineEditsPath       = "/v1/images/edits"

	imagineImageMaxBytes = 64 << 20 // 单次响应读取上限 64MiB（base64 图片可能较大）
)

// ImagineRequestError 表示 sub2api 网关返回的 OpenAI 形错误。
type ImagineRequestError struct {
	HTTPStatus int
	Type       string // error.type
	Code       string // error.code
	Message    string // error.message
}

func (e *ImagineRequestError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

// 预置错误，供 biz 层做语义映射。
var (
	ErrImagineTokenInvalid    = errors.New("sub2api token 无效或已过期")
	ErrImagineApiKeyForbidden = errors.New("apikey 无权访问（已过期/欠费/未开启生图）")
	ErrImagineQuotaExhausted  = errors.New("apikey 额度已用尽")
)

// sub2api /api/v1/keys 响应。
type imagineKeysResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []imagineKeyItem `json:"items"`
	} `json:"data"`
}

// imagineKeyItem 对应 sub2api APIKey DTO（仅取所需字段，key 原文仅服务端缓存）。
// id / user_id / group_id 用 json.Number 原样承接，全程按字符串处理，避免整型转换与溢出。
type imagineKeyItem struct {
	ID      json.Number `json:"id"`
	UserID  json.Number `json:"user_id"`
	Key     string      `json:"key"`
	Name    string      `json:"name"`
	GroupID json.Number `json:"group_id"`
	Status  string      `json:"status"`
	Group   struct {
		Name     string `json:"name"`
		Platform string `json:"platform"`
	} `json:"group"`
}

// sub2api /v1/models 响应。
type imagineModelsResponse struct {
	Data []imagineModelDTO `json:"data"`
}

type imagineModelDTO struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// ImageGenParams 是调用方（biz）提交生图所需的参数，不含任何 sub2api 线格式细节。
// 线格式请求体的拼装全部发生在本包，调用方只填业务字段。
type ImageGenParams struct {
	Model        string
	Prompt       string
	N            int
	Size         string
	Quality      string
	OutputFormat string
}

// imagineImageRequest 是发往 sub2api 的线格式请求体（OpenAI Images 形），仅本包使用。
type imagineImageRequest struct {
	Model        string            `json:"model"`
	Prompt       string            `json:"prompt"`
	N            int               `json:"n,omitempty"`
	Size         string            `json:"size,omitempty"`
	Quality      string            `json:"quality,omitempty"`
	OutputFormat string            `json:"output_format,omitempty"`
	Images       []imagineImageRef `json:"images,omitempty"`
}

type imagineImageRef struct {
	ImageURL string `json:"image_url"`
}

// imagineImageResponse 是 sub2api 生图响应（OpenAI Images 形），仅本包使用。
type imagineImageResponse struct {
	Data []imagineImageData `json:"data"`
	Size string             `json:"size"`
}

type imagineImageData struct {
	B64JSON       string `json:"b64_json"`
	RevisedPrompt string `json:"revised_prompt"`
}

// newImageRequest 由业务参数构造线格式请求体。
func newImageRequest(p ImageGenParams) *imagineImageRequest {
	return &imagineImageRequest{
		Model:        p.Model,
		Prompt:       p.Prompt,
		N:            p.N,
		Size:         p.Size,
		Quality:      p.Quality,
		OutputFormat: p.OutputFormat,
	}
}

// openAIErrorEnvelope 用于解析生图端点的错误响应。
type openAIErrorEnvelope struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
		Code    string `json:"code"`
		Param   string `json:"param"`
	} `json:"error"`
}
