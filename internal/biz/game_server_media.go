package biz

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/auth"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

// 应用图标与应用背景：管理员在服务器「接入设置」里上传，服务器详情页头与启动器同步使用。文件由浏览器直传到对象存储
// （系统设置 · 储存，与头像共用），Momoko 只保存公开链接与对象 key（换文件或删除时删掉旧对象）。
// 通行证的启动目标原样下发链接（PassportLaunchApp.icon / background），启动器按链接缓存。

const (
	AppMediaIcon       = "icon"
	AppMediaBackground = "background"

	appMediaPresignTTL = 30 * time.Minute
)

var (
	ErrAppMediaStorage = response.BadRequest("还没有开启对象存储（系统设置 · 储存）")
	ErrAppMediaKind    = response.BadRequest("只能上传应用图标或应用背景")
	ErrAppMediaSize    = response.BadRequest("文件是空的")
	ErrAppMediaTicket  = response.BadRequest("上传凭证无效或已过期，请重新上传")
	ErrAppMediaMissing = response.BadRequest("文件还没有上传到对象存储，请重新上传")
)

// appMediaTicket 一次直传的上传凭证（加密后交给前端，完成上传时原样带回）。
type appMediaTicket struct {
	Server string `json:"sv"`
	Kind   string `json:"kd"`
	Key    string `json:"k"`
	Size   int64  `json:"s"`
	Exp    int64  `json:"e"`
}

// appMediaExt 对象扩展名：原文件名的扩展名（只留小写字母和数字），没有时为 bin。
func appMediaExt(fileName string) string {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(strings.TrimSpace(fileName)), "."))
	if ext == "" || len(ext) > 16 {
		return "bin"
	}
	for _, c := range ext {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9') {
			return "bin"
		}
	}
	return ext
}

// PresignAppMedia 签发一次图标 / 背景的直传地址与上传凭证。不限文件类型与大小；对象名为
// app/<uuid>.<原扩展名>，Content-Type 用浏览器给出的类型（没有时为 application/octet-stream）。
func (uc *GameServerUsecase) PresignAppMedia(ctx context.Context, req *v1.PresignGameServerAppMediaRequest) (*v1.PresignGameServerAppMediaResponse, error) {
	if _, err := uc.Get(ctx, req.Id); err != nil {
		return nil, err
	}
	if req.Kind != AppMediaIcon && req.Kind != AppMediaBackground {
		return nil, ErrAppMediaKind
	}
	if req.Size <= 0 {
		return nil, ErrAppMediaSize
	}
	contentType := strings.TrimSpace(req.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ext := appMediaExt(req.FileName)
	store, err := objectStoreOf(ctx, uc.config)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrAppMediaStorage
	}
	uploadURL, key, err := store.PresignPut(ctx, "app/"+uuid.NewString()+"."+ext, contentType, req.Size, appMediaPresignTTL)
	if err != nil {
		return nil, response.BadRequest("签发上传地址失败: " + err.Error())
	}
	raw, err := json.Marshal(appMediaTicket{Server: req.Id, Kind: req.Kind, Key: key, Size: req.Size,
		Exp: time.Now().Add(appMediaPresignTTL).Unix()})
	if err != nil {
		return nil, ErrSystem(err)
	}
	token, err := secretbox.New(auth.AuthSecretKey).Encrypt(string(raw))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.PresignGameServerAppMediaResponse{UploadUrl: uploadURL, Token: token}, nil
}

// CompleteAppMedia 校验上传凭证（这台服务器、未过期）与对象（已上传、大小一致），保存公开链接与对象 key，
// 再删掉被替换的旧对象。
func (uc *GameServerUsecase) CompleteAppMedia(ctx context.Context, id, token string) (*gen.GameServer, error) {
	plain, err := secretbox.New(auth.AuthSecretKey).Decrypt(strings.TrimSpace(token))
	if err != nil {
		return nil, ErrAppMediaTicket
	}
	var ticket appMediaTicket
	if err := json.Unmarshal([]byte(plain), &ticket); err != nil || ticket.Server != id || ticket.Key == "" ||
		(ticket.Kind != AppMediaIcon && ticket.Kind != AppMediaBackground) {
		return nil, ErrAppMediaTicket
	}
	// 直传地址过期后对象已无法再写入；多留一会儿给慢速上传收尾。
	if time.Now().Unix() > ticket.Exp+int64(appMediaPresignTTL/time.Second) {
		return nil, ErrAppMediaTicket
	}
	store, err := objectStoreOf(ctx, uc.config)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrAppMediaStorage
	}
	if size, err := store.Size(ctx, ticket.Key); err != nil || size != ticket.Size {
		return nil, ErrAppMediaMissing
	}
	s, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	link := store.URL(ticket.Key)
	var old string
	if ticket.Kind == AppMediaIcon {
		old = s.AppIconKey
		s.AppIcon, s.AppIconKey = link, ticket.Key
	} else {
		old = s.AppBackgroundKey
		s.AppBackground, s.AppBackgroundKey = link, ticket.Key
	}
	updated, err := uc.repo.Update(ctx, s)
	if err != nil {
		uc.deleteAppMediaObjects(ctx, ticket.Key)
		return nil, ErrSystem(err)
	}
	uc.forget(id)
	if old != ticket.Key {
		uc.deleteAppMediaObjects(ctx, old)
	}
	return updated, nil
}

// DeleteAppMedia 清除图标或背景，并删除对象存储里的文件。
func (uc *GameServerUsecase) DeleteAppMedia(ctx context.Context, id, kind string) (*gen.GameServer, error) {
	s, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var old string
	switch kind {
	case AppMediaIcon:
		old = s.AppIconKey
		s.AppIcon, s.AppIconKey = "", ""
	case AppMediaBackground:
		old = s.AppBackgroundKey
		s.AppBackground, s.AppBackgroundKey = "", ""
	default:
		return nil, ErrAppMediaKind
	}
	updated, err := uc.repo.Update(ctx, s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.forget(id)
	uc.deleteAppMediaObjects(ctx, old)
	return updated, nil
}

// deleteAppMediaObjects 尽力删除对象存储里的旧文件（key 为空或存储未开启时跳过，失败只记日志）。
func (uc *GameServerUsecase) deleteAppMediaObjects(ctx context.Context, keys ...string) {
	var todo []string
	for _, key := range keys {
		if key != "" {
			todo = append(todo, key)
		}
	}
	if len(todo) == 0 {
		return
	}
	store, err := objectStoreOf(ctx, uc.config)
	if err != nil || store == nil {
		uc.log.Warnf("删除启动器文件 %v 失败：对象存储不可用", todo)
		return
	}
	for _, key := range todo {
		delCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
		if err := store.Delete(delCtx, key); err != nil {
			uc.log.Warnf("删除启动器文件 %s 失败: %v", key, err)
		}
		cancel()
	}
}
