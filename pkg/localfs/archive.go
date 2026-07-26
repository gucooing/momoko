package localfs

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	compressedSuffix = "-compressed.zip"
	extractedSuffix  = "-unzipped"
	// ratioFloor 小于此体积的条目不做压缩比判定：小文件的压缩比天然可以很夸张，
	// 对它们套比例阈值只会产生误报。
	ratioFloor = 64 << 10 // 64 KiB
)

// Compress 把若干文件/目录打包为一个 zip。
// targetVpath 为空时，默认落在第一个源同级目录、名为 "<源名>-compressed.zip"。
// 返回压缩包的真实绝对路径。
//
// 打包先写临时文件再 rename，因此不会留下半截 zip；
// 目标位于任一源内部时拒绝，避免「边写边把自己打进去」的无限增长。
func (f *FS) Compress(vpaths []string, targetVpath string) (string, error) {
	if err := f.checkWritable(); err != nil {
		return "", err
	}
	sources, err := f.collectSources(vpaths)
	if err != nil {
		return "", err
	}
	defer func() {
		for _, s := range sources {
			s.loc.close()
		}
	}()

	if strings.TrimSpace(targetVpath) == "" {
		targetVpath = defaultTargetPath(sources[0].loc.real, compressedSuffix)
	}
	dst, err := f.resolveNew(targetVpath)
	if err != nil {
		return "", err
	}
	defer dst.close()

	if err := f.checkSubtree(dst.real); err != nil {
		return "", err
	}
	for _, s := range sources {
		// 打包会把整棵子树的内容读进 zip，因此保护清单必须连子孙一起判——
		// 否则对着保护目录的父级打包，密钥就随压缩包一起被下载走了。
		if err := f.checkSubtree(s.loc.real); err != nil {
			return "", err
		}
		if overlaps(s.loc.real, dst.real) {
			return "", errors.New("压缩包不能生成在被压缩的目录内部")
		}
	}
	if info, err := dst.root.Lstat(dst.rel); err == nil && info.IsDir() {
		return "", ErrIsDir
	}
	if parent := relParent(dst.rel); parent != "." {
		if err := dst.root.MkdirAll(parent, 0o755); err != nil {
			return "", sanitize(err)
		}
	}

	tmpRel, cleanup, err := f.newTemp(dst)
	if err != nil {
		return "", err
	}
	defer cleanup()

	if err := writeZip(dst.root, tmpRel, sources, &f.policy); err != nil {
		return "", err
	}
	// rename 不覆盖已存在的目录，先按需清掉旧同名文件。
	if info, err := dst.root.Lstat(dst.rel); err == nil {
		if info.IsDir() {
			return "", ErrIsDir
		}
		if err := dst.root.Remove(dst.rel); err != nil {
			return "", sanitize(err)
		}
	}
	if err := dst.root.Rename(tmpRel, dst.rel); err != nil {
		return "", sanitize(err)
	}
	return dst.real, nil
}

// archiveSource 是一个待打包的源。
type archiveSource struct {
	loc  *loc
	info fs.FileInfo
	name string // 在 zip 内的顶层名称
}

// collectSources 解析并去重待压缩的源：按路径长度升序后剔除被其它源包含的项，
// 避免同一份内容被打进去两次。
func (f *FS) collectSources(vpaths []string) ([]archiveSource, error) {
	if len(vpaths) == 0 {
		return nil, errors.New("请选择要压缩的文件或目录")
	}
	var sources []archiveSource
	ok := false
	defer func() {
		if !ok {
			for _, s := range sources {
				s.loc.close()
			}
		}
	}()

	for _, p := range vpaths {
		if strings.TrimSpace(p) == "" {
			continue
		}
		l, err := f.resolve(p)
		if err != nil {
			return nil, err
		}
		info, err := l.root.Lstat(l.rel)
		if err != nil {
			l.close()
			return nil, sanitize(err)
		}
		name := baseName(l.rel)
		if l.isRoot() {
			name = filepath.Base(l.mount)
		}
		if name == "" || name == "." || name == string(filepath.Separator) {
			name = "archive"
		}
		sources = append(sources, archiveSource{loc: l, info: info, name: name})
	}
	if len(sources) == 0 {
		return nil, errors.New("请选择要压缩的文件或目录")
	}

	sort.Slice(sources, func(i, j int) bool {
		if len(sources[i].loc.real) == len(sources[j].loc.real) {
			return sources[i].loc.real < sources[j].loc.real
		}
		return len(sources[i].loc.real) < len(sources[j].loc.real)
	})
	kept := make([]archiveSource, 0, len(sources))
	for _, cand := range sources {
		nested := false
		for _, k := range kept {
			if k.info.IsDir() && withinPath(k.loc.real, cand.loc.real) {
				nested = true
				break
			}
		}
		if nested {
			cand.loc.close()
			continue
		}
		kept = append(kept, cand)
	}
	sources = kept
	ok = true
	return sources, nil
}

func writeZip(dstRoot *os.Root, tmpRel string, sources []archiveSource, policy *Policy) error {
	file, err := dstRoot.OpenFile(tmpRel, os.O_WRONLY, 0o600)
	if err != nil {
		return sanitize(err)
	}
	defer file.Close()

	zw := zip.NewWriter(file)
	for _, s := range sources {
		if err := addToZip(zw, s, policy); err != nil {
			_ = zw.Close()
			return err
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("写入压缩包失败: %w", err)
	}
	return sanitize(file.Sync())
}

// addToZip 把一个源写进 zip：目录以其名作为顶层前缀递归遍历，文件直接放在根层。
func addToZip(zw *zip.Writer, s archiveSource, policy *Policy) error {
	if !s.info.IsDir() {
		return zipFile(zw, s.loc.root, s.loc.rel, s.name, s.info)
	}
	return fs.WalkDir(s.loc.root.FS(), s.loc.rel, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return sanitize(err)
		}
		if policy.deniedLexical(filepath.Join(s.loc.mount, filepath.FromSlash(p))) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		entryName := s.name
		if p != s.loc.rel {
			rel, relErr := relTo(s.loc.rel, p)
			if relErr != nil {
				return relErr
			}
			entryName = s.name + "/" + rel
		}
		info, err := d.Info()
		if err != nil {
			return sanitize(err)
		}
		if d.IsDir() {
			return zipDir(zw, entryName, info)
		}
		return zipFile(zw, s.loc.root, p, entryName, info)
	})
}

func zipDir(zw *zip.Writer, name string, info fs.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("创建压缩条目失败: %w", err)
	}
	header.Name = strings.TrimSuffix(name, "/") + "/"
	header.SetMode(info.Mode())
	if _, err := zw.CreateHeader(header); err != nil {
		return fmt.Errorf("写入压缩条目失败: %w", err)
	}
	return nil
}

func zipFile(zw *zip.Writer, root *os.Root, rel, name string, info fs.FileInfo) error {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("创建压缩条目失败: %w", err)
	}
	header.Name = name
	header.Method = zip.Deflate
	header.SetMode(info.Mode())

	w, err := zw.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("写入压缩条目失败: %w", err)
	}
	in, err := root.Open(rel)
	if err != nil {
		return sanitize(err)
	}
	defer in.Close()
	if _, err := io.Copy(w, in); err != nil {
		return fmt.Errorf("写入压缩内容失败: %w", err)
	}
	return nil
}

// Extract 解压 zip。targetVpath 为空时默认解到压缩包同级的 "<包名>-unzipped" 目录。
// 返回解压目标目录的真实绝对路径。
//
// 三重防护：
//   - 条目名逐段校验并经 os.Root 落盘，zip slip（"../" 与绝对路径）在内核层即被拒绝；
//   - 条目数、总解压体积、单条目压缩比三道闸门拦 zip 炸弹；
//   - 实际写入量按剩余预算 LimitReader 截断——压缩包头里声明的大小是不可信的。
//
// 含符号链接条目的压缩包整体拒绝：解压器创建链接是经典的越权写入跳板，
// 而静默丢弃条目又会造成难以察觉的数据缺失，两者都不可取。
func (f *FS) Extract(vpath, targetVpath string) (string, error) {
	if err := f.checkWritable(); err != nil {
		return "", err
	}
	src, err := f.resolve(vpath)
	if err != nil {
		return "", err
	}
	defer src.close()

	file, err := src.root.Open(src.rel)
	if err != nil {
		return "", sanitize(err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", sanitize(err)
	}
	if info.IsDir() {
		return "", ErrIsDir
	}
	reader, err := zip.NewReader(file, info.Size())
	if err != nil {
		return "", fmt.Errorf("打开压缩包失败: %w", err)
	}
	if len(reader.File) > f.policy.MaxArchiveEntries {
		return "", fmt.Errorf("%w：条目数 %d 超过上限 %d", ErrArchiveLimit, len(reader.File), f.policy.MaxArchiveEntries)
	}
	// 聚合闸门：逐条目比压缩比会被「把炸弹切成一堆小条目」轻易绕开
	//（每条都低于单条目下限，检查一次都不触发），所以还要按整包总量判一次。
	var declared uint64
	for _, entry := range reader.File {
		declared += entry.UncompressedSize64
	}
	if declared > uint64(f.policy.MaxArchiveBytes) {
		return "", fmt.Errorf("%w：声明解压体积 %d 超过上限 %d", ErrArchiveLimit, declared, f.policy.MaxArchiveBytes)
	}
	if f.policy.MaxArchiveRatio > 0 && info.Size() > 0 {
		if ratio := declared / uint64(info.Size()); ratio > uint64(f.policy.MaxArchiveRatio) {
			return "", fmt.Errorf("%w：整包压缩比 %d 异常", ErrArchiveLimit, ratio)
		}
	}

	if strings.TrimSpace(targetVpath) == "" {
		targetVpath = defaultTargetPath(src.real, extractedSuffix)
	}
	dst, err := f.resolveNew(targetVpath)
	if err != nil {
		return "", err
	}
	defer dst.close()
	// 解压会往目标之下写入任意多个条目，保护清单必须连子孙一起判：
	// 否则解压到保护目录的父级，一个名为 configs/auth.secret 的条目就把签名密钥换掉了。
	if err := f.checkSubtree(dst.real); err != nil {
		return "", err
	}
	if overlaps(src.real, dst.real) {
		return "", errors.New("解压目标不能位于压缩包内部")
	}
	if err := dst.root.MkdirAll(dst.rel, 0o755); err != nil {
		return "", sanitize(err)
	}

	// 记录本次真正写出的文件，失败时逐一撤销：没有回滚的话，
	// 换个目标名反复触发「超预算」就能不断把半截内容留在盘上，所谓上限形同虚设。
	var written []string
	rollback := func() {
		for i := len(written) - 1; i >= 0; i-- {
			_ = dst.root.Remove(written[i])
		}
	}

	budget := f.policy.MaxArchiveBytes
	for _, entry := range reader.File {
		rel, err := f.archiveEntryRel(entry)
		if err != nil {
			rollback()
			return "", err
		}
		if rel == "" {
			continue
		}
		target := relJoin(dst.rel, rel)

		if entry.FileInfo().IsDir() {
			if err := dst.root.MkdirAll(target, 0o755); err != nil {
				rollback()
				return "", sanitize(err)
			}
			continue
		}
		if parent := relParent(target); parent != "." {
			if err := dst.root.MkdirAll(parent, 0o755); err != nil {
				rollback()
				return "", sanitize(err)
			}
		}
		n, err := extractEntry(entry, dst.root, target, budget)
		if err != nil {
			rollback()
			return "", err
		}
		written = append(written, target)
		budget -= n
	}
	return dst.real, nil
}

// archiveEntryRel 校验并规整一个压缩包条目名，返回目标内相对路径（空表示应跳过）。
func (f *FS) archiveEntryRel(entry *zip.File) (string, error) {
	mode := entry.Mode()
	if mode&fs.ModeSymlink != 0 {
		return "", fmt.Errorf("%w：包含符号链接条目 %q，为安全起见拒绝解压", ErrBadArchive, entry.Name)
	}
	if !mode.IsRegular() && !mode.IsDir() {
		return "", fmt.Errorf("%w：包含非普通文件条目 %q", ErrBadArchive, entry.Name)
	}
	// zip 规范用 '/'，但实际存在用 '\' 的实现，两者都当分隔符处理。
	name := strings.ReplaceAll(entry.Name, `\`, "/")
	if strings.IndexByte(name, 0) >= 0 {
		return "", fmt.Errorf("%w：条目名含 NUL 字节", ErrBadArchive)
	}
	if path.IsAbs(name) || filepath.VolumeName(filepath.FromSlash(name)) != "" {
		return "", fmt.Errorf("%w：条目 %q 使用了绝对路径", ErrBadArchive, entry.Name)
	}
	segs := strings.FieldsFunc(name, func(r rune) bool { return r == '/' })
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "." {
			continue
		}
		if s == ".." {
			return "", fmt.Errorf("%w：条目 %q 试图跳出解压目录", ErrBadArchive, entry.Name)
		}
		if err := validateArchiveName(s); err != nil {
			return "", fmt.Errorf("%w：条目 %q 名称非法: %v", ErrBadArchive, entry.Name, err)
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return "", nil
	}
	if len(out) > maxPathDepth {
		return "", fmt.Errorf("%w：条目 %q 层级过深", ErrBadArchive, entry.Name)
	}
	// 压缩比判定：头里声明的解压体积虽不可信，但异常比例足以先行拦下明显的炸弹。
	// 阈值按「解压后」体积开闸——炸弹的特征恰恰是压缩后极小，若按压缩后体积设下限，
	// 就会在最该生效的时候把检查关掉。
	if f.policy.MaxArchiveRatio > 0 && entry.UncompressedSize64 >= ratioFloor && entry.CompressedSize64 > 0 {
		if ratio := entry.UncompressedSize64 / entry.CompressedSize64; ratio > uint64(f.policy.MaxArchiveRatio) {
			return "", fmt.Errorf("%w：条目 %q 压缩比 %d 异常", ErrArchiveLimit, entry.Name, ratio)
		}
	}
	return strings.Join(out, "/"), nil
}

// extractEntry 写出单个条目，最多写 budget 字节；超出即判定为炸弹并清掉半成品。
func extractEntry(entry *zip.File, dstRoot *os.Root, target string, budget int64) (int64, error) {
	if budget <= 0 {
		return 0, fmt.Errorf("%w：总解压体积超过上限", ErrArchiveLimit)
	}
	rc, err := entry.Open()
	if err != nil {
		return 0, fmt.Errorf("打开压缩条目失败: %w", err)
	}
	defer rc.Close()

	mode := entry.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}
	// O_EXCL：解压到一个已有内容的目录是常规操作，静默覆盖等同于无声的数据丢失，
	// 与 CreateFile 保持同一套「新建不得误伤既有文件」的约定。
	out, err := dstRoot.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return 0, fmt.Errorf("%w：目标已存在同名文件 %q", ErrExist, entry.Name)
		}
		return 0, sanitize(err)
	}
	// 多读 1 字节以判定是否越预算。
	n, err := io.Copy(out, io.LimitReader(rc, budget+1))
	closeErr := out.Close()
	if err != nil {
		_ = dstRoot.Remove(target)
		return 0, fmt.Errorf("写入解压文件失败: %w", err)
	}
	if closeErr != nil {
		_ = dstRoot.Remove(target)
		return 0, sanitize(closeErr)
	}
	if n > budget {
		_ = dstRoot.Remove(target)
		return 0, fmt.Errorf("%w：总解压体积超过上限", ErrArchiveLimit)
	}
	return n, nil
}

// defaultTargetPath 由源真实路径推出默认目标真实路径（同级目录、去掉扩展名后加后缀）。
func defaultTargetPath(sourceReal, suffix string) string {
	base := filepath.Base(sourceReal)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || stem == "." {
		stem = base
	}
	if stem == "" || stem == "." || stem == string(filepath.Separator) {
		stem = "archive"
	}
	return filepath.Join(filepath.Dir(sourceReal), stem+suffix)
}
