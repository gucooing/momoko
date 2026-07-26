package file

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"momoko/pkg/localfs"
)

// UploadSeed 是新建一次上传会话所需的参数（业务层据此落库）。
// 纯数据结构，不含任何 ORM 类型，好让 pkg 与 internal/data 都能引用。
type UploadSeed struct {
	// Hash 客户端给出的文件指纹，用于命中同一目标的续传会话。
	Hash string
	// Dir 目标目录：本地来源为真实绝对路径（便于收尾时同卷原子 rename），远端来源为来源内逻辑路径。
	Dir string
	// Name 目标文件名，必须是通过 localfs.ValidateName 的单层名称。
	Name string
	// Size 文件总字节数。
	Size uint64
	// PartSize / Parts 分片布局。
	PartSize uint64
	Parts    uint64
	// SourceID 来源 id，空=本地。
	SourceID string
}

// NewUploadSeed 校验并构造一次上传会话的参数。
//
// 这里是上传目标文件名唯一的把关点：历史实现把它直接 Join 进目标路径，
// 于是 name="../../x" 就能写到磁盘任意位置。
//
// dir 允许为空：远端来源（OSS/FTP/WebDAV）的根目录，其逻辑路径就是空串。
// 本地来源的 dir 由调用方经 FS.RealPath 固化成绝对路径，天然非空。
func NewUploadSeed(hash, dir, name string, size uint64, sourceID string) (*UploadSeed, error) {
	if err := localfs.ValidateName(name); err != nil {
		return nil, fmt.Errorf("上传目标文件名非法: %w", err)
	}
	partSize := localfs.CalcPartSize(size)
	return &UploadSeed{
		Hash:     hash,
		Dir:      dir,
		Name:     name,
		Size:     size,
		PartSize: partSize,
		Parts:    localfs.CalcParts(size, partSize),
		SourceID: sourceID,
	}, nil
}

// Upload 是一次分片上传会话的通用载体：biz 在 PrepareUpload/CompleteUpload/CancelUpload 间盲传它，
// 各来源在接口内部自行处理机制（OSS=预签名直传多分片；本地/FTP/WebDAV=分片经 momoko 签名端点落本地缓冲后放置）。
type Upload struct {
	// Target 是来源内的目标目录（逻辑路径）：本地来源为真实绝对路径，远端来源为来源内路径。
	Target string
	// Spec 是本地缓冲与落地的布局，其 Dir 指向「缓冲文件所在目录」：
	// 本地来源即目标目录本身（收尾就是同卷原子 rename），缓冲型远端来源为统一缓冲目录的根（空）。
	// 对象存储不落本地缓冲，只用其中的体积/分片/文件名信息。
	Spec localfs.Upload
	// Buffer 是承载分片缓冲文件的本地视图；对象存储为 nil。
	Buffer *localfs.FS

	// UserID 发起用户。
	UserID string

	// Ref 是来源侧续传句柄（如 S3 multipart uploadID），对调用方不透明；为空表示尚未建立。
	// 续传/收尾/取消前由 biz 预填入；来源在 PrepareUpload 内按需建立并经 SaveRef 持久化。
	Ref string

	// SignPart 由 biz 注入：返回某分片经 momoko 签名的 PUT URL（缓冲型来源用，OSS 忽略）。
	SignPart func(part uint64) string
	// SaveRef 由 biz 注入：幂等持久化来源续传句柄并返回最终生效值（并发竞态下的胜出者）。
	SaveRef func(ctx context.Context, ref string) (string, error)

	// PartURLs 由来源在 PrepareUpload 内填充：分片号→PUT 直传地址（前端逐片直传，来源透明）。
	PartURLs map[uint64]string
	// Uploaded 已完成分片号→标识(hash/etag)：biz 先以本地分片表预填，来源可在 PrepareUpload 内补充（OSS 以远端为准）。
	Uploaded map[uint64]string
}

// ID 返回上传会话 id。
func (u *Upload) ID() string { return u.Spec.ID }

// Done 返回已完成的分片数。
func (u *Upload) Done() uint64 { return uint64(len(u.Uploaded)) }

// TargetPath 返回来源内的目标逻辑路径（正斜杠拼接）。
func (u *Upload) TargetPath() string {
	dir := strings.Trim(strings.ReplaceAll(u.Target, `\`, "/"), "/")
	if dir == "" {
		return u.Spec.Name
	}
	return dir + "/" + u.Spec.Name
}

// ---- 缓冲型上传共用助手（本地 / FTP / WebDAV）----

// fillSignedParts 用注入的 SignPart 把各分片填成 momoko 签名 PUT URL（前端逐片直传到 momoko 端点）。
func fillSignedParts(u *Upload) {
	u.PartURLs = make(map[uint64]string, u.Spec.Parts)
	if u.SignPart == nil {
		return
	}
	for i := uint64(1); i <= u.Spec.Parts; i++ {
		u.PartURLs[i] = u.SignPart(i)
	}
}

// completeBufferedRemote 把本地缓冲整流推送到远端来源，成功后删缓冲。供 FTP/WebDAV 等缓冲型远端共用。
func completeBufferedRemote(ctx context.Context, dst Store, u *Upload) error {
	if u.Buffer == nil {
		return errors.New("缺少上传缓冲视图")
	}
	if u.Spec.Parts == 0 { // 空文件直接写空对象
		return dst.Write(ctx, u.TargetPath(), strings.NewReader(""))
	}
	buf, err := u.Buffer.OpenUploadBuffer(u.Spec, u.Done())
	if err != nil {
		return err
	}
	if err := dst.Write(ctx, u.TargetPath(), buf); err != nil {
		_ = buf.Close()
		return fmt.Errorf("上传到来源失败: %w", err)
	}
	if err := buf.Close(); err != nil {
		return err
	}
	return u.Buffer.DiscardUpload(u.Spec)
}

// cancelBufferedRemote 删除缓冲型上传的本地缓冲。
func cancelBufferedRemote(u *Upload) error {
	if u.Buffer == nil {
		return nil
	}
	return u.Buffer.DiscardUpload(u.Spec)
}
