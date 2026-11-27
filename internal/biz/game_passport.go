package biz

import (
	"context"
	"strconv"

	az "momoko/api/gen/v1/azurpromilia"
	gk "momoko/api/gen/v1/gakumas"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/azurpromilia"
	"momoko/pkg/mgp/gakumas"
)

// GamePlayer 通行证绑定记录的游戏玩家快照：pid、游戏内显示的 ID、登录账号、昵称、等级与头像素材（没有时为空）。
type GamePlayer struct {
	Pid       string
	DisplayID string
	Account   string
	Name      string
	Level     int
	Avatar    string
}

// GamePassportDriver 适配器的通行证驱动：经 MGP 校验游戏账号密码、新建游戏账号、读取与搜索玩家。
// 账号或密码错误时错误码为 mgp.CodeCredentialsInvalid，玩家不存在时为 mgp.CodePlayerNotFound。
type GamePassportDriver interface {
	VerifyAccount(ctx context.Context, c *mgp.Client, account, password string) (*GamePlayer, error)
	CreateAccount(ctx context.Context, c *mgp.Client, opt mgp.OperationOptions) (*GamePlayer, error)
	Player(ctx context.Context, c *mgp.Client, pid string) (*GamePlayer, error)
	SearchPlayers(ctx context.Context, c *mgp.Client, q string, limit int) ([]*GamePlayer, error)
}

// passportDriverOf 服务器所属游戏的通行证驱动；没有时游戏账号不能经通行证绑定或新建。
func passportDriverOf(game string) (GamePassportDriver, bool) {
	a, ok := GameAdapterByID(game)
	if !ok || a.Passport == nil {
		return nil, false
	}
	return a.Passport, true
}

type gakumasPassport struct{}

func gakumasPlayer(p *gk.Player) *GamePlayer {
	return &GamePlayer{
		Pid:       p.GetPid(),
		DisplayID: p.GetDisplayId(),
		Account:   p.GetAccount(),
		Name:      p.GetName(),
		Level:     int(p.GetLevel()),
		Avatar:    p.GetAvatarIdolCardId(),
	}
}

func (gakumasPassport) VerifyAccount(ctx context.Context, c *mgp.Client, account, password string) (*GamePlayer, error) {
	p, err := gakumas.New(c).VerifyAccount(ctx, account, password)
	if err != nil {
		return nil, err
	}
	return gakumasPlayer(p), nil
}

func (gakumasPassport) CreateAccount(ctx context.Context, c *mgp.Client, opt mgp.OperationOptions) (*GamePlayer, error) {
	p, err := gakumas.New(c).CreateAccount(ctx, opt)
	if err != nil {
		return nil, err
	}
	return gakumasPlayer(p), nil
}

func (gakumasPassport) Player(ctx context.Context, c *mgp.Client, pid string) (*GamePlayer, error) {
	detail, err := gakumas.New(c).Player(ctx, pid)
	if err != nil {
		return nil, err
	}
	return gakumasPlayer(detail.GetPlayer()), nil
}

func (gakumasPassport) SearchPlayers(ctx context.Context, c *mgp.Client, q string, limit int) ([]*GamePlayer, error) {
	page, err := gakumas.New(c).Players(ctx, gakumas.PlayerFilter{Q: q}, "", int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]*GamePlayer, 0, len(page.GetItems()))
	for _, p := range page.GetItems() {
		out = append(out, gakumasPlayer(p))
	}
	return out, nil
}

type azurpromiliaPassport struct{}

// azurpromiliaPlayer 蓝色星原的玩家条目；Avatar 为「名片头像部件:性别」（如 10102:2，前端按它取头像），创角前为空。
func azurpromiliaPlayer(p *az.Player) *GamePlayer {
	avatar := ""
	if p.GetSex() != 0 {
		avatar = strconv.Itoa(int(p.GetHead())) + ":" + strconv.Itoa(int(p.GetSex()))
	}
	return &GamePlayer{
		Pid:       p.GetPid(),
		DisplayID: p.GetDisplayId(),
		Account:   p.GetAccount(),
		Name:      p.GetName(),
		Level:     int(p.GetLevel()),
		Avatar:    avatar,
	}
}

func (azurpromiliaPassport) VerifyAccount(ctx context.Context, c *mgp.Client, account, password string) (*GamePlayer, error) {
	p, err := azurpromilia.New(c).VerifyAccount(ctx, account, password)
	if err != nil {
		return nil, err
	}
	return azurpromiliaPlayer(p), nil
}

func (azurpromiliaPassport) CreateAccount(ctx context.Context, c *mgp.Client, opt mgp.OperationOptions) (*GamePlayer, error) {
	p, err := azurpromilia.New(c).CreateAccount(ctx, opt)
	if err != nil {
		return nil, err
	}
	return azurpromiliaPlayer(p), nil
}

func (azurpromiliaPassport) Player(ctx context.Context, c *mgp.Client, pid string) (*GamePlayer, error) {
	d, err := azurpromilia.New(c).Player(ctx, pid)
	if err != nil {
		return nil, err
	}
	return azurpromiliaPlayer(d.GetPlayer()), nil
}

func (azurpromiliaPassport) SearchPlayers(ctx context.Context, c *mgp.Client, q string, limit int) ([]*GamePlayer, error) {
	page, err := azurpromilia.New(c).Players(ctx, azurpromilia.PlayerFilter{Q: q}, "", int32(limit))
	if err != nil {
		return nil, err
	}
	out := make([]*GamePlayer, 0, len(page.GetItems()))
	for _, p := range page.GetItems() {
		out = append(out, azurpromiliaPlayer(p))
	}
	return out, nil
}
