package biz

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/auth"
	"momoko/pkg/common"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

// 储存：头像、应用图标与应用背景放在一个对象存储文件来源里（连接配置在文件管理 · 文件源，系统设置 · 储存只选用哪个来源），
// 各自放在来源下的固定目录。浏览器凭预签名地址直传，Momoko 只保存公开链接与文件路径（换文件或删除时删掉旧文件）。

const (
	// AvatarDir 头像所在目录。
	AvatarDir = "avatars"
	// AppMediaDir 应用图标与应用背景所在目录。
	AppMediaDir = "app"
)

// AvatarMaxSize 上传头像的大小上限。
const AvatarMaxSize = 20 << 20

// avatarPresignTTL 头像直传地址的有效期。
const avatarPresignTTL = 10 * time.Minute

// avatarImageTypes 允许上传的头像类型 → 扩展名。
var avatarImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

var (
	ErrAvatarUploadDisabled = response.BadRequest("管理员未开启头像上传")
	ErrAvatarInvalidImage   = response.BadRequest("头像仅支持 jpg、png、webp、gif 格式")
	ErrAvatarEmpty          = response.BadRequest("头像文件是空的")
	ErrAvatarTooLarge       = response.BadRequest("头像大小不能超过 20MB")
	ErrAvatarTicketInvalid  = response.BadRequest("头像上传凭证无效或已过期，请重新上传")
	ErrAvatarNotUploaded    = response.BadRequest("头像还没有上传到储存，请重新上传")
	ErrGameAssetURLInvalid  = response.BadRequest("游戏素材地址需是 http:// 或 https:// 开头的完整地址（不带 ? 和 #）")
)

// ---- 储存设置 ----

// StorageConfig 返回储存设置与可选的来源（启用中的对象存储来源）。选中的来源不在其中（已删除 / 停用）时视为未设置。
func (c *ConfigUsecase) StorageConfig(ctx context.Context) (*v1.StorageConfig, error) {
	id, err := c.config.Get(ctx, common.ConfigStorageSource)
	if err != nil {
		return nil, ErrSystem(err)
	}
	rows, err := c.files.repo.ListFileSources(ctx, "", "oss", true)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.StorageConfig{Sources: make([]*v1.StorageSource, 0, len(rows)), GameAssetUrl: c.GameAssetURL(ctx)}
	for _, r := range rows {
		out.Sources = append(out.Sources, &v1.StorageSource{Id: r.ID, Name: r.Name})
		if r.ID == id {
			out.SourceId = id
		}
	}
	return out, nil
}

// UpdateStorageConfig 选用存放头像与应用图片的来源：须为启用中的对象存储来源；空表示不存放。
func (c *ConfigUsecase) UpdateStorageConfig(ctx context.Context, req *v1.UpdateStorageConfigRequest) (*v1.StorageConfig, error) {
	id := strings.TrimSpace(req.GetSourceId())
	if id != "" {
		if _, err := c.files.publicStoreFor(ctx, id); err != nil {
			return nil, err
		}
	}
	assetURL, err := normalizeGameAssetURL(req.GetGameAssetUrl())
	if err != nil {
		return nil, err
	}
	if err := c.config.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigStorageSource: id,
		common.ConfigGameAssetURL:  assetURL,
	}); err != nil {
		return nil, ErrSystem(err)
	}
	return c.StorageConfig(ctx)
}

// GameAssetURL 游戏素材地址；没设置或读不到时为空串。
func (c *ConfigUsecase) GameAssetURL(ctx context.Context) string {
	return gameAssetURL(ctx, c.config)
}

func gameAssetURL(ctx context.Context, config ConfigRepo) string {
	v, err := config.Get(ctx, common.ConfigGameAssetURL)
	if err != nil {
		return ""
	}
	return v
}

func normalizeGameAssetURL(raw string) (string, error) {
	s := strings.TrimRight(strings.TrimSpace(raw), "/")
	if s == "" {
		return "", nil
	}
	if len(s) > 1024 {
		return "", ErrGameAssetURLInvalid
	}
	u, err := url.Parse(s)
	if err != nil || u.RawQuery != "" || u.Fragment != "" || strings.ContainsAny(s, "?#") {
		return "", ErrGameAssetURLInvalid
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", ErrGameAssetURLInvalid
	}
	return s, nil
}

// mediaStore 返回存放头像与应用图片的来源；没设置时返回 nil。
func (c *ConfigUsecase) mediaStore(ctx context.Context) (publicStore, error) {
	id, err := c.config.Get(ctx, common.ConfigStorageSource)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if id == "" {
		return nil, nil
	}
	return c.files.publicStoreFor(ctx, id)
}

// AvatarUploadEnabled 是否允许用户上传头像（设置了储存且来源可用）。
func (c *ConfigUsecase) AvatarUploadEnabled(ctx context.Context) bool {
	store, err := c.mediaStore(ctx)
	return err == nil && store != nil
}

// presignMedia 在储存里为 dir/<uuid>.<ext> 签发直传地址，返回直传地址与文件路径。
func presignMedia(ctx context.Context, store publicStore, dir, ext, contentType string, size int64, ttl time.Duration) (uploadURL, key string, err error) {
	key = dir + "/" + uuid.NewString() + "." + ext
	if uploadURL, err = store.PresignPut(ctx, key, contentType, size, ttl); err != nil {
		return "", "", response.BadRequest("签发上传地址失败: " + err.Error())
	}
	return uploadURL, key, nil
}

// uploaded 储存里的文件是否已上传完且大小一致。
func uploaded(ctx context.Context, store publicStore, key string, size int64) bool {
	info, err := store.Stat(ctx, key)
	return err == nil && !info.IsDir && int64(info.Size) == size
}

// deleteMedia 尽力删除储存里的旧文件（路径为空或没设置储存时跳过，失败只记日志）。
func (c *ConfigUsecase) deleteMedia(ctx context.Context, keys ...string) {
	var todo []string
	for _, key := range keys {
		if key != "" {
			todo = append(todo, key)
		}
	}
	if len(todo) == 0 {
		return
	}
	store, err := c.mediaStore(ctx)
	if err != nil || store == nil {
		log.Warnf("删除储存里的旧文件 %v 失败：储存不可用", todo)
		return
	}
	delCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	for _, r := range store.Delete(delCtx, todo) {
		if !r.Success {
			log.Warnf("删除储存里的旧文件 %s 失败: %s", r.Path, r.Message)
		}
	}
}

// ---- 头像（只能经储存直传）----

// avatarTicket 是一次头像直传的上传凭证内容（加密后交给前端，完成上传时原样带回）。
type avatarTicket struct {
	User string `json:"u"` // 申请人
	Key  string `json:"k"` // 文件路径
	Size int64  `json:"s"` // 申请时声明的大小（已签进直传签名）
	Exp  int64  `json:"e"` // 过期时间（unix 秒）
}

// presignAvatar 签发一次头像直传：校验类型与大小，返回预签名 PUT 地址和上传凭证。文件名是随机 uuid。
func (c *ConfigUsecase) presignAvatar(ctx context.Context, userID, contentType string, size int64) (uploadURL, token string, err error) {
	store, err := c.mediaStore(ctx)
	if err != nil {
		return "", "", err
	}
	if store == nil {
		return "", "", ErrAvatarUploadDisabled
	}
	contentType = strings.ToLower(strings.TrimSpace(contentType))
	ext, ok := avatarImageTypes[contentType]
	if !ok {
		return "", "", ErrAvatarInvalidImage
	}
	if size <= 0 {
		return "", "", ErrAvatarEmpty
	}
	if size > AvatarMaxSize {
		return "", "", ErrAvatarTooLarge
	}
	uploadURL, key, err := presignMedia(ctx, store, AvatarDir, ext, contentType, size, avatarPresignTTL)
	if err != nil {
		return "", "", err
	}
	raw, err := json.Marshal(avatarTicket{User: userID, Key: key, Size: size, Exp: time.Now().Add(avatarPresignTTL).Unix()})
	if err != nil {
		return "", "", ErrSystem(err)
	}
	if token, err = secretbox.New(auth.AuthSecretKey).Encrypt(string(raw)); err != nil {
		return "", "", ErrSystem(err)
	}
	return uploadURL, token, nil
}

// completeAvatar 校验上传凭证（本人、未过期）并确认文件已在储存里且大小一致，返回头像公开链接与文件路径。
func (c *ConfigUsecase) completeAvatar(ctx context.Context, userID, token string) (link, key string, err error) {
	plain, err := secretbox.New(auth.AuthSecretKey).Decrypt(strings.TrimSpace(token))
	if err != nil {
		return "", "", ErrAvatarTicketInvalid
	}
	var ticket avatarTicket
	if err := json.Unmarshal([]byte(plain), &ticket); err != nil || ticket.User != userID || ticket.Key == "" {
		return "", "", ErrAvatarTicketInvalid
	}
	// 直传地址过期后已无法再写入；多留一会儿给慢速上传收尾。
	if time.Now().Unix() > ticket.Exp+int64(avatarPresignTTL/time.Second) {
		return "", "", ErrAvatarTicketInvalid
	}
	store, err := c.mediaStore(ctx)
	if err != nil {
		return "", "", err
	}
	if store == nil {
		return "", "", ErrAvatarUploadDisabled
	}
	if !uploaded(ctx, store, ticket.Key, ticket.Size) {
		return "", "", ErrAvatarNotUploaded
	}
	return store.PublicURL(ticket.Key), ticket.Key, nil
}
