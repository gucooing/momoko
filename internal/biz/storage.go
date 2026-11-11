package biz

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/auth"
	"momoko/pkg/objectstore"
	"momoko/pkg/response"
	"momoko/pkg/secretbox"
)

// AvatarMaxSize 上传头像的大小上限。
const AvatarMaxSize = 20 << 20

// avatarPresignTTL 头像直传地址的有效期。
const avatarPresignTTL = 10 * time.Minute

// avatarImageTypes 允许上传的头像类型 → 对象扩展名。
var avatarImageTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

var (
	ErrAvatarUploadDisabled = response.BadRequest(400, "管理员未开启头像上传")
	ErrAvatarInvalidImage   = response.BadRequest(400, "头像仅支持 jpg、png、webp、gif 格式")
	ErrAvatarEmpty          = response.BadRequest(400, "头像文件是空的")
	ErrAvatarTooLarge       = response.BadRequest(400, "头像大小不能超过 20MB")
	ErrAvatarTicketInvalid  = response.BadRequest(400, "头像上传凭证无效或已过期，请重新上传")
	ErrAvatarNotUploaded    = response.BadRequest(400, "头像还没有上传到对象存储，请重新上传")
)

// ---- 储存设置 ----

// StorageConfig 返回储存设置（Secret Key 不回传，只告知是否已保存）。
func (c *ConfigUsecase) StorageConfig(ctx context.Context) (*v1.StorageConfig, error) {
	avatar, err := c.config.AvatarStorage(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.StorageConfig{Avatar: maskAvatarStorage(avatar)}, nil
}

// UpdateStorageConfig 保存储存设置：Secret Key 留空表示保留原值，新值加密落库。
func (c *ConfigUsecase) UpdateStorageConfig(ctx context.Context, req *v1.UpdateStorageConfigRequest) (*v1.StorageConfig, error) {
	next := req.GetAvatar()
	if next == nil {
		next = &v1.AvatarStorageConfig{}
	}
	current, err := c.config.AvatarStorage(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}

	cfg := &v1.AvatarStorageConfig{
		Enabled:   next.Enabled,
		Endpoint:  strings.TrimSpace(next.Endpoint),
		Region:    strings.TrimSpace(next.Region),
		Bucket:    strings.TrimSpace(next.Bucket),
		AccessKey: strings.TrimSpace(next.AccessKey),
		UseSsl:    next.UseSsl,
		PathStyle: next.PathStyle,
		Prefix:    strings.Trim(strings.TrimSpace(next.Prefix), "/"),
		PublicUrl: strings.TrimRight(strings.TrimSpace(next.PublicUrl), "/"),
		SecretKey: current.SecretKey, // 已是加密态
	}
	box := secretbox.New(auth.AuthSecretKey)
	if next.SecretKey != "" {
		if cfg.SecretKey, err = box.Encrypt(next.SecretKey); err != nil {
			return nil, ErrSystem(err)
		}
	}

	if cfg.Enabled {
		if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
			return nil, response.BadRequest(400, "开启头像储存需要填写服务地址、存储桶、Access Key 和 Secret Key")
		}
	}
	// 填了就校验格式（未开启也校验，避免存进一份打不开的配置）。
	if cfg.Endpoint != "" || cfg.Bucket != "" {
		if _, err := objectstore.New(toObjectStoreConfig(cfg, "")); err != nil {
			return nil, response.BadRequest(400, err.Error())
		}
	}

	if err := c.config.SaveAvatarStorage(ctx, cfg); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.StorageConfig{Avatar: maskAvatarStorage(cfg)}, nil
}

// AvatarUploadEnabled 是否允许用户上传头像（开启了头像储存）。
func (c *ConfigUsecase) AvatarUploadEnabled(ctx context.Context) bool {
	cfg, err := c.config.AvatarStorage(ctx)
	return err == nil && cfg.Enabled
}

// avatarStore 按当前头像储存设置构造对象存储客户端；未开启返回 nil。
func (c *ConfigUsecase) avatarStore(ctx context.Context) (*objectstore.Store, error) {
	cfg, err := c.config.AvatarStorage(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !cfg.Enabled {
		return nil, nil
	}
	secret, err := secretbox.New(auth.AuthSecretKey).Decrypt(cfg.SecretKey)
	if err != nil {
		return nil, ErrSystem(err)
	}
	store, err := objectstore.New(toObjectStoreConfig(cfg, secret))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return store, nil
}

func maskAvatarStorage(cfg *v1.AvatarStorageConfig) *v1.AvatarStorageConfig {
	return &v1.AvatarStorageConfig{
		Enabled:      cfg.Enabled,
		Endpoint:     cfg.Endpoint,
		Region:       cfg.Region,
		Bucket:       cfg.Bucket,
		AccessKey:    cfg.AccessKey,
		SecretKeySet: cfg.SecretKey != "",
		UseSsl:       cfg.UseSsl,
		PathStyle:    cfg.PathStyle,
		Prefix:       cfg.Prefix,
		PublicUrl:    cfg.PublicUrl,
	}
}

func toObjectStoreConfig(cfg *v1.AvatarStorageConfig, secret string) objectstore.Config {
	return objectstore.Config{
		Endpoint:  cfg.Endpoint,
		Region:    cfg.Region,
		Bucket:    cfg.Bucket,
		AccessKey: cfg.AccessKey,
		SecretKey: secret,
		UseSSL:    cfg.UseSsl,
		PathStyle: cfg.PathStyle,
		Prefix:    cfg.Prefix,
		PublicURL: cfg.PublicUrl,
	}
}

// ---- 头像（只能经对象存储直传）----

// avatarTicket 是一次头像直传的上传凭证内容（加密后交给前端，完成上传时原样带回）。
type avatarTicket struct {
	User string `json:"u"` // 申请人
	Key  string `json:"k"` // 对象 key
	Size int64  `json:"s"` // 申请时声明的大小（已签进直传签名）
	Exp  int64  `json:"e"` // 过期时间（unix 秒）
}

// presignAvatar 签发一次头像直传：校验类型与大小，返回预签名 PUT 地址和上传凭证。对象名是随机 uuid。
func (c *ConfigUsecase) presignAvatar(ctx context.Context, userID, contentType string, size int64) (uploadURL, token string, err error) {
	store, err := c.avatarStore(ctx)
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
	uploadURL, key, err := store.PresignPut(ctx, uuid.NewString()+"."+ext, contentType, size, avatarPresignTTL)
	if err != nil {
		return "", "", response.BadRequest(400, "签发头像上传地址失败: "+err.Error())
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

// completeAvatar 校验上传凭证（本人、未过期）并确认对象已在对象存储里且大小一致，返回头像公开链接与对象 key。
func (c *ConfigUsecase) completeAvatar(ctx context.Context, userID, token string) (link, key string, err error) {
	plain, err := secretbox.New(auth.AuthSecretKey).Decrypt(strings.TrimSpace(token))
	if err != nil {
		return "", "", ErrAvatarTicketInvalid
	}
	var ticket avatarTicket
	if err := json.Unmarshal([]byte(plain), &ticket); err != nil || ticket.User != userID || ticket.Key == "" {
		return "", "", ErrAvatarTicketInvalid
	}
	// 直传地址过期后对象已无法再写入；多留一会儿给慢速上传收尾。
	if time.Now().Unix() > ticket.Exp+int64(avatarPresignTTL/time.Second) {
		return "", "", ErrAvatarTicketInvalid
	}
	store, err := c.avatarStore(ctx)
	if err != nil {
		return "", "", err
	}
	if store == nil {
		return "", "", ErrAvatarUploadDisabled
	}
	size, err := store.Size(ctx, ticket.Key)
	if err != nil || size != ticket.Size {
		return "", "", ErrAvatarNotUploaded
	}
	return store.URL(ticket.Key), ticket.Key, nil
}

// deleteAvatarObject 按对象 key 尽力删除旧头像（key 为空或头像储存未开启则跳过，失败只记日志）。
func (c *ConfigUsecase) deleteAvatarObject(ctx context.Context, key string) {
	if key == "" {
		return
	}
	store, err := c.avatarStore(ctx)
	if err != nil || store == nil {
		return
	}
	delCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	if err := store.Delete(delCtx, key); err != nil {
		log.Warnf("删除旧头像失败 %s: %v", key, err)
	}
}
