// Package azurpromilia 是蓝色星原游戏服务端 MGP 接口的调用（游戏服务端 server/README.md「MGP」）：
// 每个函数对应一个 MGP 接口，筛选与分页由服务端完成，返回解码好的内容（api/gen/v1/azurpromilia 的类型）。
package azurpromilia

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	az "momoko/api/gen/v1/azurpromilia"
	"momoko/pkg/mgp"
)

// ID 是蓝色星原的游戏 id。
const ID = "azurpromilia"

// Client 调用一台蓝色星原服务器的 MGP。
type Client struct {
	c *mgp.Client
}

func New(c *mgp.Client) *Client {
	return &Client{
		c: c,
	}
}

var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

func decode(raw []byte, out proto.Message) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return decoder.Unmarshal(raw, out)
}

func get[T any, P interface {
	*T
	proto.Message
}](ctx context.Context, c *mgp.Client, path string, q url.Values) (*T, error) {
	raw, err := c.Get(ctx, path, q)
	if err != nil {
		return nil, err
	}
	out := P(new(T))
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

func operate[T any, P interface {
	*T
	proto.Message
}](ctx context.Context, c *mgp.Client, path string, body any, opt mgp.OperationOptions) (*T, error) {
	r, err := c.Operate(ctx, http.MethodPost, path, body, opt)
	if err != nil {
		return nil, err
	}
	out := P(new(T))
	if err := decode(r.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}

// page 查询参数：上一页的游标与每页条数（0 为服务端默认）。
func page(q url.Values, cursor string, limit int32) url.Values {
	if q == nil {
		q = url.Values{}
	}
	if cursor != "" {
		q.Set("cursor", cursor)
	}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(int(limit)))
	}
	return q
}

func list(q url.Values, key string, v []string) {
	var keep []string
	for _, s := range v {
		if s = strings.TrimSpace(s); s != "" {
			keep = append(keep, s)
		}
	}
	if len(keep) > 0 {
		q.Set(key, strings.Join(keep, ","))
	}
}

func playerPath(pid string) string {
	return "/players/" + url.PathEscape(pid)
}

// ---------------- 服务器 ----------------

func (g *Client) Manifest(ctx context.Context) (*az.Manifest, error) {
	return get[az.Manifest](ctx, g.c, "/manifest", nil)
}

func (g *Client) Status(ctx context.Context) (*az.Status, error) {
	return get[az.Status](ctx, g.c, "/status", nil)
}

// DailyStats 最近 days 天（含今天）的统计；days 为 0 时取服务端默认。
func (g *Client) DailyStats(ctx context.Context, days int32) (*az.DailyStats, error) {
	q := url.Values{}
	if days > 0 {
		q.Set("days", strconv.Itoa(int(days)))
	}
	return get[az.DailyStats](ctx, g.c, "/stats/daily", q)
}

// ---------------- 玩家 ----------------

// PlayerFilter GET /players 的筛选：Q 匹配账号名或昵称的一部分、pid 或玩家 ID；Filter 为 all / online / banned。
type PlayerFilter struct {
	Q           string
	Filter      string
	PIDs        []string
	ExcludePIDs []string
}

// Players 玩家列表（items、nextCursor、total 与各筛选的人数 summary）。
func (g *Client) Players(ctx context.Context, f PlayerFilter, cursor string, limit int32) (*az.PlayerPage, error) {
	q := url.Values{}
	if f.Q != "" {
		q.Set("q", f.Q)
	}
	if f.Filter != "" {
		q.Set("filter", f.Filter)
	}
	list(q, "pids", f.PIDs)
	list(q, "excludePids", f.ExcludePIDs)
	return get[az.PlayerPage](ctx, g.c, "/players", page(q, cursor, limit))
}

// Player 玩家条目与概览。MGP 把条目的字段与 overview 放在同一层，分两次解码。
func (g *Client) Player(ctx context.Context, pid string) (*az.PlayerDetail, error) {
	raw, err := g.c.Get(ctx, playerPath(pid), nil)
	if err != nil {
		return nil, err
	}
	out := &az.PlayerDetail{
		Player:   &az.Player{},
		Overview: nil,
	}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	out.Player = &az.Player{}
	if err := decode(raw, out.Player); err != nil {
		return nil, err
	}
	return out, nil
}

func (g *Client) Logins(ctx context.Context, pid, cursor string, limit int32) (*az.LoginPage, error) {
	return get[az.LoginPage](ctx, g.c, playerPath(pid)+"/logins", page(nil, cursor, limit))
}

// Items 背包道具（按道具 ID，同一道具合计）；ids 不为空时只取这些道具。
func (g *Client) Items(ctx context.Context, pid string, ids []int32, cursor string, limit int32) (*az.ItemPage, error) {
	q := url.Values{}
	keep := make([]string, 0, len(ids))
	for _, id := range ids {
		keep = append(keep, strconv.Itoa(int(id)))
	}
	list(q, "ids", keep)
	return get[az.ItemPage](ctx, g.c, playerPath(pid)+"/items", page(q, cursor, limit))
}

func (g *Client) Currencies(ctx context.Context, pid string) (*az.ItemPage, error) {
	return get[az.ItemPage](ctx, g.c, playerPath(pid)+"/currencies", nil)
}

func (g *Client) Heroes(ctx context.Context, pid, cursor string, limit int32) (*az.HeroPage, error) {
	return get[az.HeroPage](ctx, g.c, playerPath(pid)+"/heroes", page(nil, cursor, limit))
}

func (g *Client) Pets(ctx context.Context, pid, cursor string, limit int32) (*az.PetPage, error) {
	return get[az.PetPage](ctx, g.c, playerPath(pid)+"/pets", page(nil, cursor, limit))
}

func (g *Client) Mails(ctx context.Context, pid, cursor string, limit int32) (*az.MailPage, error) {
	return get[az.MailPage](ctx, g.c, playerPath(pid)+"/mails", page(nil, cursor, limit))
}

// VerifyAccount 校验游戏账号密码，返回该账号的玩家条目；密码错误时错误码为 mgp.CodeCredentialsInvalid。
func (g *Client) VerifyAccount(ctx context.Context, account, password string) (*az.Player, error) {
	r, err := g.c.Do(ctx, mgp.Call{
		Method: http.MethodPost,
		Path:   "/accounts/verify",
		Body: map[string]string{
			"account":  account,
			"password": password,
		},
	})
	if err != nil {
		return nil, err
	}
	out := &az.Player{}
	if err := decode(r.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateAccount 新建一个没有密码的游戏账号，返回它的玩家条目。
func (g *Client) CreateAccount(ctx context.Context, opt mgp.OperationOptions) (*az.Player, error) {
	return operate[az.Player](ctx, g.c, "/accounts", map[string]any{}, opt)
}

// ---------------- 对玩家的操作 ----------------

func rewards(rs []*az.Reward) []map[string]int32 {
	out := make([]map[string]int32, 0, len(rs))
	for _, r := range rs {
		out = append(out, map[string]int32{
			"type": r.GetType(),
			"id":   r.GetId(),
			"num":  r.GetNum(),
		})
	}
	return out
}

// AdjustItems 发放 grant、扣除 take（扣除只能是货币与普通道具，不足时失败）。
func (g *Client) AdjustItems(ctx context.Context, pid string, grant, take []*az.Reward, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/items/adjust", map[string]any{
		"grant": rewards(grant),
		"take":  rewards(take),
	}, opt)
}

func mailBody(m *az.MailContent) map[string]any {
	return map[string]any{
		"sender":  m.GetSender(),
		"title":   m.GetTitle(),
		"content": m.GetContent(),
		"rewards": rewards(m.GetRewards()),
		"effect":  uint32(m.GetEffect()),
	}
}

// Mail 给玩家投递一封邮件。
func (g *Client) Mail(ctx context.Context, pid string, m *az.MailContent, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/mail", mailBody(m), opt)
}

// Kick 把在线玩家踢下线；不在线时错误码为 NOT_ONLINE。
func (g *Client) Kick(ctx context.Context, pid string, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/kick", map[string]any{}, opt)
}

// Ban 封禁到 until（Unix 毫秒，0 为永久），reason 为游戏内显示的原因。
func (g *Client) Ban(ctx context.Context, pid string, until int64, reason string, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/ban", map[string]any{
		"until":  until,
		"reason": reason,
	}, opt)
}

func (g *Client) Unban(ctx context.Context, pid string, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/unban", map[string]any{}, opt)
}

func (g *Client) ResetPassword(ctx context.Context, pid, password string, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/password", map[string]any{
		"password": password,
	}, opt)
}

func (g *Client) Rename(ctx context.Context, pid, name string, opt mgp.OperationOptions) (*az.OperationData, error) {
	return operate[az.OperationData](ctx, g.c, playerPath(pid)+"/rename", map[string]any{
		"name": name,
	}, opt)
}

// ---------------- 邮件补偿 ----------------

func (g *Client) GmMails(ctx context.Context, q, cursor string, limit int32) (*az.GmMailPage, error) {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	return get[az.GmMailPage](ctx, g.c, "/mails", page(v, cursor, limit))
}

func (g *Client) GmMail(ctx context.Context, id string) (*az.GmMail, error) {
	return get[az.GmMail](ctx, g.c, "/mails/"+url.PathEscape(id), nil)
}

// SendMail 发送邮件补偿：target 为 all（全服，可定时与限定领取截止）或 players。
func (g *Client) SendMail(ctx context.Context, target string, pids []string, m *az.MailContent, startAt, endAt int64, opt mgp.OperationOptions) (*az.SendMailResult, error) {
	body := mailBody(m)
	body["target"] = target
	if pids == nil {
		pids = []string{}
	}
	body["pids"] = pids
	body["startAt"] = startAt
	body["endAt"] = endAt
	return operate[az.SendMailResult](ctx, g.c, "/mails", body, opt)
}

// RevokeMail 撤回全服邮件。
func (g *Client) RevokeMail(ctx context.Context, id string, opt mgp.OperationOptions) (*az.GmMail, error) {
	return operate[az.GmMail](ctx, g.c, "/mails/"+url.PathEscape(id)+"/revoke", map[string]any{}, opt)
}

// ---------------- 兑换码 ----------------

func (g *Client) CodeBatches(ctx context.Context, q, cursor string, limit int32) (*az.CodeBatchPage, error) {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	return get[az.CodeBatchPage](ctx, g.c, "/codes", page(v, cursor, limit))
}

func (g *Client) CodeBatch(ctx context.Context, id string) (*az.CodeBatch, error) {
	return get[az.CodeBatch](ctx, g.c, "/codes/"+url.PathEscape(id), nil)
}

// CodesInput 新建的兑换码批次：Codes 为指定的码（通用码），否则生成 Count 个码。
type CodesInput struct {
	Name     string
	Codes    []string
	Count    int32
	CodeUses int32
	MinLevel int32
	StartAt  int64
	EndAt    int64
	Mail     *az.MailContent
}

func (g *Client) CreateCodes(ctx context.Context, in CodesInput, opt mgp.OperationOptions) (*az.CreateCodesResult, error) {
	body := mailBody(in.Mail)
	body["name"] = in.Name
	if len(in.Codes) > 0 {
		body["codes"] = in.Codes
	} else {
		body["count"] = in.Count
	}
	body["codeUses"] = in.CodeUses
	body["minLevel"] = in.MinLevel
	body["startAt"] = in.StartAt
	body["endAt"] = in.EndAt
	return operate[az.CreateCodesResult](ctx, g.c, "/codes", body, opt)
}

// SetCodeBatchDisabled 停用或恢复整批兑换码。
func (g *Client) SetCodeBatchDisabled(ctx context.Context, id string, disabled bool, opt mgp.OperationOptions) (*az.CodeBatch, error) {
	action := "/enable"
	if disabled {
		action = "/disable"
	}
	return operate[az.CodeBatch](ctx, g.c, "/codes/"+url.PathEscape(id)+action, map[string]any{}, opt)
}

func (g *Client) BatchCodes(ctx context.Context, id, cursor string, limit int32) (*az.CodePage, error) {
	return get[az.CodePage](ctx, g.c, "/codes/"+url.PathEscape(id)+"/codes", page(nil, cursor, limit))
}

func (g *Client) BatchRedemptions(ctx context.Context, id, cursor string, limit int32) (*az.RedemptionPage, error) {
	return get[az.RedemptionPage](ctx, g.c, "/codes/"+url.PathEscape(id)+"/redemptions", page(nil, cursor, limit))
}
