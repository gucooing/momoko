package biz

import (
	"context"
	"fmt"
	"strconv"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entgameserver "momoko/internal/data/ent/gen/gameserver"
	entgmexecution "momoko/internal/data/ent/gen/gmexecution"
)

// GameHostingDriver 适配器的托管驱动：识别目录、写配置、创建进程（应用实例）、专用设置、更新与备份。
// 由托管用例注册到适配器上（见 game_hosting.go）；没有注册时该游戏只能「仅 GM」接入。
type GameHostingDriver interface {
	// Capabilities 托管驱动实际支持的能力：settings / update / resource / backup / logs / files / console。
	Capabilities() []string
	// State 返回托管服务器的进程状态（空串表示不确定）。
	State(ctx context.Context, s *gen.GameServer) string
	// Describe 补充托管服务器详情：进程、托管选项、主机名、进行中的任务。
	Describe(ctx context.Context, s *gen.GameServer, info *v1.GameServerInfo)
	// Spec 托管定义：进程角色、默认配置文件、日志模块、备份范围。
	Spec() GameHostingSpec
}

// GameHostingSpec 适配器给出的托管定义。
type GameHostingSpec struct {
	Roles       []*v1.GameAdapterRole
	ConfigFile  string
	LogModules  []string
	BackupScope []string
}

// GameAdapter 编进 Momoko 的游戏驱动：决定一款游戏能被托管到什么程度、GM 怎么接。
// 能力全部由已实现的驱动推出，不写死。
type GameAdapter struct {
	ID string
	// GMDriver GM 驱动：mgp 服务端原生实现 MGP。
	GMDriver string
	// Hosting 托管驱动（未实现时为 nil）。
	Hosting GameHostingDriver
}

// Modes 支持的接入方式：有 MGP 驱动即可仅 GM 接入，有托管驱动即可托管。
func (a *GameAdapter) Modes() []string {
	var modes []string
	if a.Hosting != nil {
		modes = append(modes, "hosted")
	}
	if a.GMDriver == "mgp" {
		modes = append(modes, "remote")
	}
	return modes
}

// SupportsMode 判断适配器是否支持某种接入方式。
func (a *GameAdapter) SupportsMode(mode string) bool {
	for _, m := range a.Modes() {
		if m == mode {
			return true
		}
	}
	return false
}

// Capabilities 已实现的能力：hosting / gm / passport（经 MGP 校验账号与自助指令）/ 托管驱动的能力。
func (a *GameAdapter) Capabilities() []string {
	var caps []string
	if a.Hosting != nil {
		caps = append(caps, "hosting")
		caps = append(caps, a.Hosting.Capabilities()...)
	}
	if a.GMDriver == "mgp" {
		caps = append(caps, "gm", "passport")
	}
	return caps
}

// ServerCapabilities 一台服务器当前可用的能力：托管服务器有托管驱动的全部能力；GM 与通行证需要接入 MGP。
func (a *GameAdapter) ServerCapabilities(s *gen.GameServer) []string {
	caps := []string{}
	if s.Mode == entgameserver.ModeHosted && a.Hosting != nil {
		caps = append(caps, "hosting")
		caps = append(caps, a.Hosting.Capabilities()...)
	}
	if a.GMDriver == "mgp" && MGPConfigured(s) {
		caps = append(caps, "gm", "passport")
	}
	return caps
}

// ToProto 转为接口结构。
func (a *GameAdapter) ToProto() *v1.GameAdapter {
	out := &v1.GameAdapter{
		Id:         a.ID,
		Available:  len(a.Modes()) > 0,
		Modes:      a.Modes(),
		GmDriver:   a.GMDriver,
	}
	for _, c := range a.Capabilities() {
		out.Capabilities = append(out.Capabilities, &v1.GameAdapterCapability{Key: c, Enabled: true})
	}
	if a.Hosting != nil {
		spec := a.Hosting.Spec()
		out.Roles = spec.Roles
		out.ConfigFile = spec.ConfigFile
		out.LogModules = spec.LogModules
		out.BackupScope = spec.BackupScope
	}
	return out
}

// gameAdapters 随 Momoko 版本内置的适配器（顺序即展示顺序）。
var gameAdapters = []*GameAdapter{
	{ID: "gakumas", GMDriver: "mgp"},
}

// GameAdapters 返回全部内置适配器。
func GameAdapters() []*GameAdapter { return gameAdapters }

// RegisterGameAdapter 注册（或替换）一个适配器。
func RegisterGameAdapter(a *GameAdapter) {
	for i, existing := range gameAdapters {
		if existing.ID == a.ID {
			gameAdapters[i] = a
			return
		}
	}
	gameAdapters = append(gameAdapters, a)
}

// GameAdapterByID 按 id 查找适配器。
func GameAdapterByID(id string) (*GameAdapter, bool) {
	for _, a := range gameAdapters {
		if a.ID == id {
			return a, true
		}
	}
	return nil, false
}

func gameServerMode(mode string) entgameserver.Mode {
	if mode == "hosted" {
		return entgameserver.ModeHosted
	}
	return entgameserver.ModeRemote
}

func gameVisibility(v string) entgameserver.PassportVisibility {
	switch v {
	case "invite":
		return entgameserver.PassportVisibilityInvite
	case "hidden":
		return entgameserver.PassportVisibilityHidden
	default:
		return entgameserver.PassportVisibilityPublic
	}
}

func itoa(n int) string { return strconv.Itoa(n) }

func strconvFormatSeconds(d time.Duration) string {
	return fmt.Sprintf("%.1f", d.Seconds())
}

func gmActorKind(kind string) entgmexecution.ActorKind {
	if kind == "player" {
		return entgmexecution.ActorKindPlayer
	}
	return entgmexecution.ActorKindAdmin
}

func gmChannel(channel string) entgmexecution.Channel {
	switch channel {
	case "passport":
		return entgmexecution.ChannelPassport
	case "game":
		return entgmexecution.ChannelGame
	default:
		return entgmexecution.ChannelAdmin
	}
}
