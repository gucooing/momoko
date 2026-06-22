package sub2api

import (
	"context"
	"strconv"
	"strings"

	"momoko/pkg/imagestore"
)

// ImageResult 单张生成图片落盘后的结果（供 biz 持久化）。
type ImageResult struct {
	Path          string // 相对 imagestore 根的路径
	Filename      string
	SizeBytes     int64
	Width         int
	Height        int
	MimeType      string
	RevisedPrompt string
	Index         int
}

// GenerateAndSave 文生图：调用 sub2api /v1/images/generations，解码 base64 并落盘。
// 所有 sub2api 交互（线格式拼装、HTTP、文件写盘）都在本方法内完成；biz 仅需持久化返回的 ImageResult。
func (c *ImagineClient) GenerateAndSave(ctx context.Context, srcHost, apiKey, userID, generationID string, params ImageGenParams) ([]ImageResult, error) {
	resp, err := c.doImage(ctx, srcHost, apiKey, imagineGenerationsPath, newImageRequest(params))
	if err != nil {
		return nil, ClassifyImagineError(err)
	}
	return saveImages(resp, userID, generationID, params.OutputFormat)
}

// EditAndSave 图生图：调用 sub2api /v1/images/edits，把整张源图（base64 data URL）
// 作为 images[].image_url 直传上游，解码 base64 结果并落盘。
func (c *ImagineClient) EditAndSave(ctx context.Context, srcHost, apiKey, userID, generationID, sourceImage string, params ImageGenParams) ([]ImageResult, error) {
	req := newImageRequest(params)
	req.Images = []imagineImageRef{{ImageURL: sourceImage}}
	resp, err := c.doImage(ctx, srcHost, apiKey, imagineEditsPath, req)
	if err != nil {
		return nil, ClassifyImagineError(err)
	}
	return saveImages(resp, userID, generationID, params.OutputFormat)
}

// saveImages 解析响应、逐张解码并落盘。
func saveImages(resp *imagineImageResponse, userID, generationID, outputFormat string) ([]ImageResult, error) {
	width, height := parseImageSize(resp.Size)
	mime := mimeTypeFor(outputFormat)
	out := make([]ImageResult, 0, len(resp.Data))
	for i, d := range resp.Data {
		data, err := DecodeB64Image(d.B64JSON)
		if err != nil || len(data) == 0 {
			continue
		}
		relPath, filename, err := imagestore.SaveImage(userID, generationID, i, data, outputFormat)
		if err != nil {
			return out, err
		}
		out = append(out, ImageResult{
			Path:          relPath,
			Filename:      filename,
			SizeBytes:     int64(len(data)),
			Width:         width,
			Height:        height,
			MimeType:      mime,
			RevisedPrompt: d.RevisedPrompt,
			Index:         i,
		})
	}
	return out, nil
}

func parseImageSize(s string) (int, int) {
	parts := strings.SplitN(s, "x", 2)
	if len(parts) != 2 {
		return 0, 0
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil {
		return 0, 0
	}
	return w, h
}

func mimeTypeFor(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
