package biz

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/auth"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

// 启动器的游戏图标与游戏页背景：管理员在服务器「接入设置 → 客户端」里上传。文件由浏览器直传到对象存储
// （系统设置 · 储存，与头像共用），Momoko 只保存公开链接与对象 key（换文件或删除时删掉旧对象）。
// 通行证的启动目标原样下发链接（PassportLaunchApp.icon / background），启动器按链接缓存。

const (
	LauncherMediaIcon       = "icon"
	LauncherMediaBackground = "background"

	launcherIconMaxSize       = 20 << 20
	launcherBackgroundMaxSize = 200 << 20
	launcherMediaPresignTTL   = 30 * time.Minute
)

var (
	launcherImageTypes = map[string]string{"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp", "image/gif": "gif"}
	launcherVideoTypes = map[string]string{"video/mp4": "mp4", "video/webm": "webm"}
)

var (
	ErrLauncherMediaStorage = response.BadRequest(400, "还没有开启对象存储（系统设置 · 储存）")
	ErrLauncherMediaKind    = response.BadRequest(400, "只能上传游戏图标或游戏页背景")
	ErrLauncherMediaType    = response.BadRequest(400, "图标支持 jpg、png、webp、gif；背景还支持 mp4、webm 视频")
	ErrLauncherMediaSize    = response.BadRequest(400, "文件是空的，或超过大小上限（图标 20 MB，背景 200 MB）")
	ErrLauncherMediaTicket  = response.BadRequest(400, "上传凭证无效或已过期，请重新上传")
	ErrLauncherMediaMissing = response.BadRequest(400, "文件还没有上传到对象存储，请重新上传")
)

// launcherMediaTicket 一次直传的上传凭证（加密后交给前端，完成上传时原样带回）。
type launcherMediaTicket struct {
	Server string `json:"sv"`
	Kind   string `json:"kd"`
	Key    string `json:"k"`
	Size   int64  `json:"s"`
	Exp    int64  `json:"e"`
}

// launcherMediaExt 按种类与类型返回对象扩展名，以及这一种的大小上限。
func launcherMediaExt(kind, contentType string) (string, int64, error) {
	switch kind {
	case LauncherMediaIcon:
		if ext, ok := launcherImageTypes[contentType]; ok {
			return ext, launcherIconMaxSize, nil
		}
	case LauncherMediaBackground:
		if ext, ok := launcherImageTypes[contentType]; ok {
			return ext, launcherBackgroundMaxSize, nil
		}
		if ext, ok := launcherVideoTypes[contentType]; ok {
			return ext, launcherBackgroundMaxSize, nil
		}
	default:
		return "", 0, ErrLauncherMediaKind
	}
	return "", 0, ErrLauncherMediaType
}

// PresignLauncherMedia 签发一次图标 / 背景的直传地址与上传凭证。对象名为 launcher/<uuid>.<扩展名>。
func (uc *GameServerUsecase) PresignLauncherMedia(ctx context.Context, req *v1.PresignGameServerLauncherMediaRequest) (*v1.PresignGameServerLauncherMediaResponse, error) {
	if _, err := uc.Get(ctx, req.Id); err != nil {
		return nil, err
	}
	contentType := strings.ToLower(strings.TrimSpace(req.ContentType))
	ext, limit, err := launcherMediaExt(req.Kind, contentType)
	if err != nil {
		return nil, err
	}
	if req.Size <= 0 || req.Size > limit {
		return nil, ErrLauncherMediaSize
	}
	store, err := objectStoreOf(ctx, uc.config)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrLauncherMediaStorage
	}
	uploadURL, key, err := store.PresignPut(ctx, "launcher/"+uuid.NewString()+"."+ext, contentType, req.Size, launcherMediaPresignTTL)
	if err != nil {
		return nil, response.BadRequest(400, "签发上传地址失败: "+err.Error())
	}
	raw, err := json.Marshal(launcherMediaTicket{Server: req.Id, Kind: req.Kind, Key: key, Size: req.Size,
		Exp: time.Now().Add(launcherMediaPresignTTL).Unix()})
	if err != nil {
		return nil, ErrSystem(err)
	}
	token, err := secretbox.New(auth.AuthSecretKey).Encrypt(string(raw))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.PresignGameServerLauncherMediaResponse{UploadUrl: uploadURL, Token: token}, nil
}

// CompleteLauncherMedia 校验上传凭证（这台服务器、未过期）与对象（已上传、大小一致），保存公开链接与对象 key，
// 再删掉被替换的旧对象。
func (uc *GameServerUsecase) CompleteLauncherMedia(ctx context.Context, id, token string) (*gen.GameServer, error) {
	plain, err := secretbox.New(auth.AuthSecretKey).Decrypt(strings.TrimSpace(token))
	if err != nil {
		return nil, ErrLauncherMediaTicket
	}
	var ticket launcherMediaTicket
	if err := json.Unmarshal([]byte(plain), &ticket); err != nil || ticket.Server != id || ticket.Key == "" ||
		(ticket.Kind != LauncherMediaIcon && ticket.Kind != LauncherMediaBackground) {
		return nil, ErrLauncherMediaTicket
	}
	// 直传地址过期后对象已无法再写入；多留一会儿给慢速上传收尾。
	if time.Now().Unix() > ticket.Exp+int64(launcherMediaPresignTTL/time.Second) {
		return nil, ErrLauncherMediaTicket
	}
	store, err := objectStoreOf(ctx, uc.config)
	if err != nil {
		return nil, err
	}
	if store == nil {
		return nil, ErrLauncherMediaStorage
	}
	if size, err := store.Size(ctx, ticket.Key); err != nil || size != ticket.Size {
		return nil, ErrLauncherMediaMissing
	}
	s, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	link := store.URL(ticket.Key)
	var old string
	if ticket.Kind == LauncherMediaIcon {
		old = s.LauncherIconKey
		s.LauncherIcon, s.LauncherIconKey = link, ticket.Key
	} else {
		old = s.LauncherBackgroundKey
		s.LauncherBackground, s.LauncherBackgroundKey = link, ticket.Key
	}
	updated, err := uc.repo.Update(ctx, s)
	if err != nil {
		uc.deleteLauncherObjects(ctx, ticket.Key)
		return nil, ErrSystem(err)
	}
	uc.forget(id)
	if old != ticket.Key {
		uc.deleteLauncherObjects(ctx, old)
	}
	return updated, nil
}

// DeleteLauncherMedia 清除图标或背景，并删除对象存储里的文件。
func (uc *GameServerUsecase) DeleteLauncherMedia(ctx context.Context, id, kind string) (*gen.GameServer, error) {
	s, err := uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var old string
	switch kind {
	case LauncherMediaIcon:
		old = s.LauncherIconKey
		s.LauncherIcon, s.LauncherIconKey = "", ""
	case LauncherMediaBackground:
		old = s.LauncherBackgroundKey
		s.LauncherBackground, s.LauncherBackgroundKey = "", ""
	default:
		return nil, ErrLauncherMediaKind
	}
	updated, err := uc.repo.Update(ctx, s)
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.forget(id)
	uc.deleteLauncherObjects(ctx, old)
	return updated, nil
}

// deleteLauncherObjects 尽力删除对象存储里的旧文件（key 为空或存储未开启时跳过，失败只记日志）。
func (uc *GameServerUsecase) deleteLauncherObjects(ctx context.Context, keys ...string) {
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
