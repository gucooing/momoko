// Package sharetype 存放文件分享的叶子数据类型：作为 file_share.items 的 JSON 列元素，
// 被 ent schema、pkg/share、internal/biz 共同引用。独立成包以避免
// ent schema → 类型 → ent gen → ent schema 的循环依赖（本包不导入 gen）。
package sharetype

import "time"

// Item 是分享内的一个被分享条目（可跨来源）。
// 除来源 id 与来源内路径外，还缓存创建时探测到的名称/类型/大小/修改时间，
// 使后续浏览分享根目录时无需再逐一访问来源 Stat，显著加快分享页打开速度。
type Item struct {
	// SourceID 文件来源 id，空=本地磁盘。
	SourceID string `json:"source_id"`
	// Path 来源内的真实逻辑路径。
	Path string `json:"path"`
	// Name 展示名（缓存；创建/更新时由服务端 Stat 填充，即该条目在虚拟顶层目录中的名称）。
	Name string `json:"name"`
	// IsDir 是否目录（缓存）。
	IsDir bool `json:"is_dir"`
	// Size 大小（缓存，字节）；目录为其下全部文件的递归总大小。
	Size uint64 `json:"size"`
	// FileCount 包含的文件数（缓存）：文件恒为 1；目录为递归统计，-1 表示目录过大或来源不可达、未能统计。
	FileCount int64 `json:"file_count"`
	// Children 目录的直接子项数（缓存，列分享根目录时展示）；-1 表示未能统计。
	Children int64 `json:"children"`
	// Counted 是否已做过统计（旧记录无此字段，浏览时补统计并回写）。
	Counted bool `json:"counted"`
	// UpdateTime 修改时间（缓存；列根目录时直接展示，避免再次 Stat）。
	UpdateTime time.Time `json:"update_time"`
}
