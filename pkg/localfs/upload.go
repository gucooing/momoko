package localfs

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
)

const (
	// MinPartSize 分片下限，取 S3 分片上传的最小分片大小，使同一套分片方案对本地与对象存储通用。
	MinPartSize = 5 << 20 // 5 MiB
	// MaxPartSize 分片上限。
	MaxPartSize = 64 << 20 // 64 MiB
	// targetParts 期望的分片数量（据文件大小在上下限之间取值）。
	targetParts = 100
	// BufferPrefix 是分片缓冲文件的名称前缀；以它开头的条目会被列目录/搜索隐藏。
	BufferPrefix = ".momoko-upload-"
	// bufferNameFmt 是缓冲文件名模板。只由服务端生成的会话 id 决定，
	// 绝不掺入用户提供的文件名——那正是旧实现可以借上传写到任意路径的原因。
	bufferNameFmt = BufferPrefix + "%s.part"
)

// Upload 描述一次分片上传的布局。缓冲文件与目标文件都落在 Dir 之内，
// 因此收尾的 rename 必然同卷、瞬时且原子完成。
//
// 缓冲文件名只由服务端生成的会话 id 决定，用户提供的文件名无从参与路径拼接；
// 它以 [BufferPrefix] 开头，因而在列目录与搜索时被自动隐藏，不会碍着用户。
type Upload struct {
	// ID 是服务端生成的上传会话标识，必须通过 [ValidateToken]。
	ID string
	// Dir 是目标目录（本视图内的 vpath），缓冲文件也落在这里。
	Dir string
	// Name 是目标文件名，必须是通过 [ValidateName] 的单层名称。
	Name string
	// Size 是文件总字节数。
	Size uint64
	// PartSize 是每片大小（最后一片可以更小）。
	PartSize uint64
	// Parts 是分片总数；Size 为 0 时必须为 0。
	Parts uint64
}

// CalcPartSize 依文件大小选择分片大小：尽量分成 targetParts 片，并夹在上下限之间。
func CalcPartSize(size uint64) uint64 {
	if size == 0 {
		return MinPartSize
	}
	ps := size / targetParts
	if size%targetParts != 0 {
		ps++
	}
	if ps < MinPartSize {
		ps = MinPartSize
	}
	if ps > MaxPartSize {
		ps = MaxPartSize
	}
	return ps
}

// CalcParts 返回给定大小与分片大小下的分片总数。
// 刻意不写成 (size+partSize-1)/partSize：那个式子会溢出，
// 于是一个精心构造的巨大 size 能算出 Parts=0，绕过 Validate 的自洽检查。
func CalcParts(size, partSize uint64) uint64 {
	if size == 0 || partSize == 0 {
		return 0
	}
	parts := size / partSize
	if size%partSize != 0 {
		parts++
	}
	return parts
}

// Validate 校验布局自洽且各字段安全。任何上传相关操作都必须先过这一关。
// 体积上限另由所属视图的 Policy 把关（见 FS.validateUpload）。
func (u Upload) Validate() error {
	if err := ValidateToken(u.ID); err != nil {
		return fmt.Errorf("上传会话标识非法: %w", err)
	}
	// 目标文件名是历史上最危险的一处输入：它曾被直接 Join 进目标路径，
	// 于是 "../../x" 就能写到磁盘任意位置。这里是唯一的把关点。
	if err := ValidateName(u.Name); err != nil {
		return fmt.Errorf("目标文件名非法: %w", err)
	}
	if u.Size > math.MaxInt64 {
		return fmt.Errorf("%w：声明的文件大小超出范围", ErrTooLarge)
	}
	if u.PartSize == 0 {
		return errors.New("分片大小不能为 0")
	}
	if want := CalcParts(u.Size, u.PartSize); u.Parts != want {
		return fmt.Errorf("分片数与文件大小不符：声明 %d，应为 %d", u.Parts, want)
	}
	return nil
}

// validateUpload 在 Upload.Validate 之上叠加本视图的策略上限。
// 体积必须有界：声明一个天文数字的 Size 后只发最后一片，
// io.NewOffsetWriter 会在支持稀疏文件的文件系统上一次铺开整个体积。
func (f *FS) validateUpload(u Upload) error {
	if err := u.Validate(); err != nil {
		return err
	}
	if f.policy.MaxUploadSize > 0 && u.Size > uint64(f.policy.MaxUploadSize) {
		return fmt.Errorf("%w：声明体积 %d 超过上传上限 %d", ErrTooLarge, u.Size, f.policy.MaxUploadSize)
	}
	return nil
}

// bufferName 返回缓冲文件名。
func (u Upload) bufferName() string {
	return fmt.Sprintf(bufferNameFmt, u.ID)
}

// partRange 返回第 part 片在文件内的偏移与长度。part 从 1 计数。
func (u Upload) partRange(part uint64) (offset, size uint64, err error) {
	if part == 0 || part > u.Parts {
		return 0, 0, fmt.Errorf("分片号 %d 超出范围（1..%d）", part, u.Parts)
	}
	offset = (part - 1) * u.PartSize
	size = u.PartSize
	if part == u.Parts {
		size = u.Size - offset
	}
	if offset > math.MaxInt64 || size > math.MaxInt64 {
		return 0, 0, fmt.Errorf("%w：分片偏移超出范围", ErrTooLarge)
	}
	return offset, size, nil
}

// WriteUploadPart 把一个分片写入缓冲文件的对应偏移，返回实际字节数与该片的 SHA-256。
//
// 严格按声明长度收取：读到的字节数必须恰好等于该片应有长度，多一个字节即拒绝。
// 否则客户端可以用「声明小片、实际发大片」把相邻分片覆盖掉，或让缓冲文件超出声明体积。
func (f *FS) WriteUploadPart(u Upload, part uint64, r io.Reader) (uint64, string, error) {
	if err := f.checkWritable(); err != nil {
		return 0, "", err
	}
	if err := f.validateUpload(u); err != nil {
		return 0, "", err
	}
	offset, want, err := u.partRange(part)
	if err != nil {
		return 0, "", err
	}

	l, err := f.resolve(u.Dir)
	if err != nil {
		return 0, "", err
	}
	defer l.close()
	// 目标目录可能尚不存在（客户端直接上传到新路径），按需补建；
	// 与其它建目录的入口一样，逐段校验名称合法性。
	if err := validateRelNames(l.rel); err != nil {
		return 0, "", err
	}
	if err := l.root.MkdirAll(l.rel, 0o755); err != nil {
		return 0, "", sanitize(err)
	}

	bufRel := relJoin(l.rel, u.bufferName())
	file, err := l.root.OpenFile(bufRel, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, "", sanitize(err)
	}
	defer file.Close()

	hasher := sha256.New()
	//nolint:gosec // offset 已在 partRange 内确认不超过 MaxInt64
	w := io.MultiWriter(io.NewOffsetWriter(file, int64(offset)), hasher)
	//nolint:gosec // want 同上
	written, err := io.CopyBuffer(w, io.LimitReader(r, int64(want)), make([]byte, 64<<10))
	if err != nil {
		return 0, "", fmt.Errorf("写入分片失败: %w", err)
	}
	if uint64(written) != want {
		return 0, "", fmt.Errorf("分片长度不符：收到 %d，应为 %d", written, want)
	}
	// 确认客户端没有多发：多出的字节会落到下一片的区间里。
	var extra [1]byte
	if n, err := r.Read(extra[:]); err != nil && !errors.Is(err, io.EOF) {
		return 0, "", fmt.Errorf("读取分片失败: %w", err)
	} else if n > 0 {
		return 0, "", fmt.Errorf("分片长度不符：超出声明的 %d 字节", want)
	}
	if err := file.Close(); err != nil {
		return 0, "", sanitize(err)
	}
	return want, hex.EncodeToString(hasher.Sum(nil)), nil
}

// CommitUpload 收尾一次本地上传：校验缓冲齐备后原子 rename 到 Dir/Name。
// 返回目标文件的真实绝对路径。doneParts 是调用方（数据库分片表）记录的已完成片数。
//
// 缓冲与目标同目录，因此这里恒为同卷 rename：瞬时、原子、零额外 I/O。
// 把缓冲集中到别处看着更整洁，但那与目标多半不同卷，只能退化成一次全量复制，
// 几个 GB 的上传要为此多读写一遍——不值得。
func (f *FS) CommitUpload(u Upload, doneParts uint64) (string, error) {
	if err := f.checkWritable(); err != nil {
		return "", err
	}
	if err := f.validateUpload(u); err != nil {
		return "", err
	}
	l, err := f.resolve(u.Dir)
	if err != nil {
		return "", err
	}
	defer l.close()

	targetRel := relJoin(l.rel, u.Name)
	targetReal := joinReal(l.mount, targetRel)
	if f.policy.denied(targetReal) {
		return "", ErrDenied
	}
	if info, err := l.root.Lstat(targetRel); err == nil && info.IsDir() {
		return "", ErrIsDir
	}

	// 只有确实是空文件才走这条路。若 Parts 因任何原因为 0 而 Size 不为 0，
	// 落下去就是拿 O_TRUNC 把一个既有文件清零——必须挡住。
	if u.Parts == 0 && u.Size != 0 {
		return "", fmt.Errorf("分片布局非法：声明体积 %d 但分片数为 0", u.Size)
	}
	if u.Parts == 0 { // 空文件没有分片，直接落一个空文件
		file, err := l.root.OpenFile(targetRel, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return "", sanitize(err)
		}
		if err := file.Close(); err != nil {
			return "", sanitize(err)
		}
		_ = f.DiscardUpload(u)
		return targetReal, nil
	}

	bufRel := relJoin(l.rel, u.bufferName())
	if err := verifyBuffer(l.root, bufRel, u, doneParts); err != nil {
		return "", err
	}
	if err := l.root.Rename(bufRel, targetRel); err != nil {
		return "", sanitize(err)
	}
	return targetReal, nil
}

// OpenUploadBuffer 校验后打开缓冲文件用于读取，供「缓冲到本地再整流推送到远端来源」的收尾使用。
// 调用方负责 Close，并在推送成功后调用 [FS.DiscardUpload]。
func (f *FS) OpenUploadBuffer(u Upload, doneParts uint64) (*os.File, error) {
	if err := f.validateUpload(u); err != nil {
		return nil, err
	}
	l, err := f.resolve(u.Dir)
	if err != nil {
		return nil, err
	}
	defer l.close()

	bufRel := relJoin(l.rel, u.bufferName())
	if err := verifyBuffer(l.root, bufRel, u, doneParts); err != nil {
		return nil, err
	}
	file, err := l.root.Open(bufRel)
	if err != nil {
		return nil, sanitize(err)
	}
	return file, nil
}

// DiscardUpload 删除缓冲文件（不存在视为成功）。取消上传与清理残留会话都用它。
func (f *FS) DiscardUpload(u Upload) error {
	if err := ValidateToken(u.ID); err != nil {
		return err
	}
	l, err := f.resolve(u.Dir)
	if err != nil {
		return err
	}
	defer l.close()

	if err := l.root.Remove(relJoin(l.rel, u.bufferName())); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return sanitize(err)
	}
	return nil
}

// verifyBuffer 校验缓冲文件已齐备：分片数齐全，且实际体积恰等于声明体积。
// 每片的偏移与长度已在收片时逐片校验，这里只把关「漏片」与「总长」。
func verifyBuffer(root *os.Root, bufRel string, u Upload, doneParts uint64) error {
	if doneParts != u.Parts {
		return fmt.Errorf("分片未上传完成：%d/%d", doneParts, u.Parts)
	}
	info, err := root.Stat(bufRel)
	if err != nil {
		return fmt.Errorf("上传缓冲文件缺失: %w", sanitize(err))
	}
	if uint64(info.Size()) != u.Size {
		return fmt.Errorf("文件体积不符：实际 %d，声明 %d", info.Size(), u.Size)
	}
	return nil
}
