package mgp

import "encoding/json"

// 风险等级：low 直接执行；medium 执行前预览；high 必须填原因；danger 输入玩家昵称确认，并由服务端自动备份。
const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"
	RiskDanger = "danger"
)

// 指令目标：player 单个玩家；players 1–200 个玩家；server 无目标。
const (
	TargetPlayer  = "player"
	TargetPlayers = "players"
	TargetServer  = "server"
)

// MaxTargets 一条玩家指令最多的目标数。
const MaxTargets = 200

// Manifest 是 GET /manifest 中 Momoko 需要理解的部分；其余字段按原样透传给前端（Raw）。
type Manifest struct {
	Protocol int `json:"protocol"`
	Game     struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Theme string `json:"theme"`
		Icon  string `json:"icon"`
	} `json:"game"`
	Server struct {
		Version  string          `json:"version"`
		Resource json.RawMessage `json:"resource"`
	} `json:"server"`
	Capabilities []string          `json:"capabilities"`
	Views        []ManifestEntry   `json:"views"`
	Commands     []Command         `json:"commands"`
	Reports      []ManifestEntry   `json:"reports"`
	Modes        []ManifestEntry   `json:"modes"`
	Catalogs     []ManifestEntry   `json:"catalogs"`
	Configs      []ManifestConfig  `json:"configs"`
	ResourceType map[string]int32  `json:"resourceTypes"`
	Raw          json.RawMessage   `json:"-"`
	byID         map[string]*Command
}

// ManifestEntry 是 manifest 中带 id 与标题的条目（视图、报表、玩法、目录）。
type ManifestEntry struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Kind  string `json:"kind,omitempty"`
}

// ManifestConfig 是一个可在线编辑的配置域。
type ManifestConfig struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	File      string `json:"file"`
	HotReload bool   `json:"hotReload"`
}

// Command 是服务端声明的一条 GM 指令。
type Command struct {
	ID          string          `json:"id"`
	Title       string          `json:"title"`
	Group       string          `json:"group"`
	Risk        string          `json:"risk"`
	Target      string          `json:"target"`
	SelfService bool            `json:"selfService"`
	Params      json.RawMessage `json:"params"`
}

// ParseManifest 解析 manifest 原始 JSON。
func ParseManifest(raw json.RawMessage) (*Manifest, error) {
	m := &Manifest{}
	if err := json.Unmarshal(raw, m); err != nil {
		return nil, err
	}
	m.Raw = raw
	m.byID = make(map[string]*Command, len(m.Commands))
	for i := range m.Commands {
		m.byID[m.Commands[i].ID] = &m.Commands[i]
	}
	return m, nil
}

// Command 按 id 查找指令。
func (m *Manifest) Command(id string) (*Command, bool) {
	if m == nil {
		return nil, false
	}
	c, ok := m.byID[id]
	return c, ok
}

// HasCapability 判断服务端是否声明了某项能力。
func (m *Manifest) HasCapability(name string) bool {
	if m == nil {
		return false
	}
	for _, c := range m.Capabilities {
		if c == name {
			return true
		}
	}
	return false
}

// HasMode 判断服务端是否配置了某个玩法。
func (m *Manifest) HasMode(id string) bool {
	if m == nil {
		return false
	}
	for _, mode := range m.Modes {
		if mode.ID == id {
			return true
		}
	}
	return false
}

// Status 是 GET /status 中 Momoko 需要理解的部分；完整内容按原样透传（Raw）。
type Status struct {
	Online        int    `json:"online"`
	OnlinePeak    int    `json:"onlinePeak"`
	Registered    int64  `json:"registered"`
	NewToday      int64  `json:"newToday"`
	UptimeSeconds int64  `json:"uptimeSeconds"`
	Now           int64  `json:"now"`
	Version       string `json:"version"`
	Maintenance   struct {
		Enabled bool `json:"enabled"`
	} `json:"maintenance"`
	Health []struct {
		Name string `json:"name"`
		OK   bool   `json:"ok"`
	} `json:"health"`
	Raw json.RawMessage `json:"-"`
}

// ParseStatus 解析 status 原始 JSON。
func ParseStatus(raw json.RawMessage) (*Status, error) {
	s := &Status{}
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, err
	}
	s.Raw = raw
	return s, nil
}

// Healthy 判断全部依赖是否健康。
func (s *Status) Healthy() bool {
	if s == nil {
		return false
	}
	for _, h := range s.Health {
		if !h.OK {
			return false
		}
	}
	return true
}

// CommandRequest 是 POST /commands/{cmd} 的请求体。
type CommandRequest struct {
	Targets   []string        `json:"targets"`
	Params    json.RawMessage `json:"params,omitempty"`
	Reason    string          `json:"reason"`
	RequestID string          `json:"requestId"`
	DryRun    bool            `json:"dryRun"`
	Operator  string          `json:"operator"`
}

// Change 是指令对一个对象的实际变化。
type Change struct {
	Kind       string          `json:"kind"`
	ResourceID string          `json:"resourceId"`
	Before     json.RawMessage `json:"before"`
	After      json.RawMessage `json:"after"`
}

// TargetResult 是指令对一个玩家的结果。
type TargetResult struct {
	PID      string          `json:"pid"`
	Name     string          `json:"name"`
	Status   string          `json:"status"`
	Code     string          `json:"code,omitempty"`
	Message  string          `json:"message,omitempty"`
	Changes  []Change        `json:"changes,omitempty"`
	Detail   json.RawMessage `json:"detail,omitempty"`
	BackupID json.Number     `json:"backupId,omitempty"`
}

// CommandResult 是指令响应的 data。
type CommandResult struct {
	Command   string          `json:"command"`
	RequestID string          `json:"requestId"`
	DryRun    bool            `json:"dryRun"`
	Replayed  bool            `json:"replayed"`
	Results   []TargetResult  `json:"results"`
	Summary   json.RawMessage `json:"summary,omitempty"`
}

// Failed 统计失败的目标数。
func (r *CommandResult) Failed() int {
	n := 0
	for _, t := range r.Results {
		if t.Status != "success" {
			n++
		}
	}
	return n
}

// Player 是玩家列表条目中 Momoko 需要理解的部分（绑定、校验账号时使用）。
type Player struct {
	PID         string `json:"pid"`
	DisplayID   string `json:"displayId"`
	Account     string `json:"account"`
	Name        string `json:"name"`
	Level       int    `json:"level"`
	Online      bool   `json:"online"`
	LastLoginAt int64  `json:"lastLoginAt"`
	BannedUntil int64  `json:"bannedUntil"`
	// AvatarIdolCardID 出借记忆的偶像卡（头像）。
	AvatarIdolCardID string `json:"avatarIdolCardId"`
}

// Page 是 MGP 列表的统一结构。
type Page struct {
	Items      json.RawMessage `json:"items"`
	NextCursor string          `json:"nextCursor"`
	Total      int64           `json:"total"`
	Summary    json.RawMessage `json:"summary,omitempty"`
}

// ConfigInfo 是 GET /configs 列表中的一个配置域。
type ConfigInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	File     string `json:"file"`
	Revision string `json:"revision"`
}

// ConfigDocument 是 GET /configs/{domain} 的结果。
type ConfigDocument struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	File     string          `json:"file"`
	Revision string          `json:"revision"`
	Content  json.RawMessage `json:"content"`
}

// ConfigPut 是 PUT /configs/{domain} 的请求体。
type ConfigPut struct {
	Content  json.RawMessage `json:"content"`
	Revision string          `json:"revision"`
	DryRun   bool            `json:"dryRun"`
	Operator string          `json:"operator"`
	Comment  string          `json:"comment"`
}

// ConfigPutResult 是 PUT /configs/{domain} 的结果（字段按服务端返回透传）。
type ConfigPutResult struct {
	Valid    bool   `json:"valid"`
	Error    string `json:"error"`
	Revision string `json:"revision"`
}
