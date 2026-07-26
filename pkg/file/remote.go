package file

import (
	"fmt"
	"path"
	"strings"

	"momoko/pkg/localfs"
)

// invalidRemotePath 是各来源路径助手在遇到越界输入时返回的兜底值：
// 含 NUL 的路径在 S3 / FTP / WebDAV 上都必然请求失败，即「宁可报错也不越界」。
// 正常情况下走不到这里——各导出方法已在入口用 guard 拦截。
const invalidRemotePath = "/\x00invalid"

// guard 在来源的每个导出方法入口校验一批逻辑路径，任一越界即整体拒绝。
// 远端来源的边界只能靠这一处守住，因此校验放在信任边界上，而不是散落在内部拼接处。
func guard(paths ...string) error {
	for _, p := range paths {
		if _, err := remoteRel(p); err != nil {
			return err
		}
	}
	return nil
}

// remoteRel 把来源内的逻辑路径归一化为「相对来源根的干净相对路径」（正斜杠，无 "." 无 ".."）。
// 根目录返回空串。
//
// 远端来源（OSS/FTP/WebDAV）没有 os.Root 那样的内核级边界，配置里的 Prefix/BasePath
// 是唯一的约束面，因此这里必须「拒绝」而不是「折叠」：
// path.Join("data", "../etc") 会静默得到 "etc"，把 BasePath 整个吃掉，
// 于是一个本该锁在 data/ 下的来源就能读到服务器上的其它目录。
func remoteRel(p string) (string, error) {
	if strings.IndexByte(p, 0) >= 0 {
		return "", fmt.Errorf("%w：不能包含 NUL 字节", localfs.ErrInvalidPath)
	}
	segs := strings.FieldsFunc(strings.ReplaceAll(p, `\`, "/"), func(r rune) bool { return r == '/' })
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "." {
			continue
		}
		if s == ".." || strings.Count(s, ".") == len(s) {
			return "", localfs.ErrTraversal
		}
		out = append(out, s)
	}
	return strings.Join(out, "/"), nil
}

// remoteAbs 把逻辑路径解析成远端绝对路径（以 "/" 开头），供 FTP / WebDAV 使用。
func remoteAbs(base, logical string) (string, error) {
	rel, err := remoteRel(logical)
	if err != nil {
		return "", err
	}
	return "/" + path.Join(base, rel), nil
}

// remoteKey 把逻辑路径解析成对象存储 key（不以 "/" 开头），供 OSS 使用。
func remoteKey(root, logical string) (string, error) {
	rel, err := remoteRel(logical)
	if err != nil {
		return "", err
	}
	switch {
	case root == "":
		return rel, nil
	case rel == "":
		return root, nil
	default:
		return root + "/" + rel, nil
	}
}

// remoteLogical 把对象 key 转回逻辑路径（相对 root）。
// 按路径边界剥前缀，避免 root="a" 时把兄弟前缀 "ab/c" 误当成自己的子项。
func remoteLogical(root, key string) string {
	if root == "" {
		return strings.Trim(key, "/")
	}
	if key == root {
		return ""
	}
	if rest, ok := strings.CutPrefix(key, root+"/"); ok {
		return strings.Trim(rest, "/")
	}
	return strings.Trim(key, "/")
}

// remoteRename 校验改名参数并给出目标逻辑路径。
// newName 必须是合法的单层名称——只查 "/" 与 "\" 是不够的，".." 同样能把目标顶到上级目录。
func remoteRename(logical, newName string) (srcRel, dstRel string, err error) {
	if err := localfs.ValidateName(newName); err != nil {
		return "", "", err
	}
	srcRel, err = remoteRel(logical)
	if err != nil {
		return "", "", err
	}
	if srcRel == "" {
		return "", "", localfs.ErrRootScope
	}
	dir := path.Dir(srcRel)
	if dir == "." {
		return srcRel, newName, nil
	}
	return srcRel, dir + "/" + newName, nil
}

// remoteBase 归一化来源配置里的 Prefix / BasePath。
func remoteBase(raw string) string {
	rel, err := remoteRel(raw)
	if err != nil {
		return ""
	}
	return rel
}
