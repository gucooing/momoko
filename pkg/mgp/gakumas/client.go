// Package gakumas 是学園アイドルマスター（学マス）游戏服务端 MGP 接口的调用：每个函数对应一个 MGP 接口
// （游戏服务端 docs/mgp.md），筛选与分页由服务端完成，返回解码好的内容（api/gen/v1/gakumas 的类型）。
package gakumas

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	gk "momoko/api/gen/v1/gakumas"
	"momoko/pkg/mgp"
)

// ID 是学マス 的游戏 id。
const ID = "gakumas"

// ResourceJewelPaid 付费钻石的资源类型（游戏协议 ResourceType；邮件不能发放）。
const ResourceJewelPaid int32 = 1101

// Client 调用一台学マス 服务器的 MGP。
type Client struct {
	c *mgp.Client
}

func New(c *mgp.Client) *Client {
	return &Client{
		c: c,
	}
}

// Query 查询参数：空值省略，列表以逗号连接。
type Query url.Values

func (q Query) Str(key, v string) Query {
	if v != "" {
		url.Values(q).Set(key, v)
	}
	return q
}

func (q Query) Num(key string, v int64) Query {
	if v != 0 {
		url.Values(q).Set(key, strconv.FormatInt(v, 10))
	}
	return q
}

func (q Query) Flag(key string, v bool) Query {
	if v {
		url.Values(q).Set(key, "true")
	}
	return q
}

func (q Query) List(key string, v []string) Query {
	var keep []string
	for _, s := range v {
		if s = strings.TrimSpace(s); s != "" {
			keep = append(keep, s)
		}
	}
	if len(keep) > 0 {
		url.Values(q).Set(key, strings.Join(keep, ","))
	}
	return q
}

func (q Query) Nums(key string, v []int32) Query {
	keep := make([]string, 0, len(v))
	for _, n := range v {
		keep = append(keep, strconv.FormatInt(int64(n), 10))
	}
	return q.List(key, keep)
}

// Page 上一页的游标与每页条数（0 为服务端默认）。
func (q Query) Page(cursor string, limit int32) Query {
	return q.Str("cursor", cursor).Num("limit", int64(limit))
}

func playerPath(pid string) string { return "/players/" + url.PathEscape(pid) }

// MGP 的 JSON 字段名与 api/gen/v1/gakumas 消息的 JSON 名一致，数据直接解码进消息。
var decoder = protojson.UnmarshalOptions{DiscardUnknown: true}

func decode(raw []byte, out proto.Message) error {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	return decoder.Unmarshal(raw, out)
}

// get 调用一个查询并解码。
func get[T any, P interface {
	*T
	proto.Message
}](ctx context.Context, c *mgp.Client, path string, q Query) (*T, error) {
	raw, err := c.Get(ctx, path, url.Values(q))
	if err != nil {
		return nil, err
	}
	out := P(new(T))
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	return out, nil
}

// operate 执行一个操作并解码结果。
func operate[T any, P interface {
	*T
	proto.Message
}](ctx context.Context, c *mgp.Client, method, path string, body any, opt mgp.OperationOptions) (*T, error) {
	r, err := c.Operate(ctx, method, path, body, opt)
	if err != nil {
		return nil, err
	}
	out := P(new(T))
	if err := decode(r.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}

// ---------------- 服务器 ----------------

func (g *Client) Manifest(ctx context.Context) (*gk.Manifest, error) {
	return get[gk.Manifest](ctx, g.c, "/manifest", nil)
}

func (g *Client) Status(ctx context.Context) (*gk.Status, error) {
	return get[gk.Status](ctx, g.c, "/status", nil)
}

// Dashboard 工作台：days 为每日序列的天数（0 为默认 7）。
func (g *Client) Dashboard(ctx context.Context, days int32) (*gk.Dashboard, error) {
	return get[gk.Dashboard](ctx, g.c, "/dashboard", Query{}.Num("days", int64(days)))
}

func (g *Client) SetMaintenance(ctx context.Context, enabled bool, whitelist []string, opt mgp.OperationOptions) (*gk.Maintenance, error) {
	if whitelist == nil {
		whitelist = []string{}
	}
	return operate[gk.Maintenance](ctx, g.c, http.MethodPut, "/maintenance", map[string]any{
		"enabled":   enabled,
		"whitelist": whitelist,
	}, opt)
}

// ---------------- 玩家与账号 ----------------

// PlayerFilter GET /players 的筛选。
type PlayerFilter struct {
	Q           string
	Filter      string
	PIDs        []string
	ExcludePIDs []string
}

// Players 玩家列表（items、nextCursor、total 与各筛选的人数 summary）。
func (g *Client) Players(ctx context.Context, f PlayerFilter, cursor string, limit int32) (*gk.PlayerPage, error) {
	q := Query{}.Str("q", f.Q).Str("filter", f.Filter).List("pids", f.PIDs).List("excludePids", f.ExcludePIDs).Page(cursor, limit)
	return get[gk.PlayerPage](ctx, g.c, "/players", q)
}

// Player 玩家条目、概览与登录次数。MGP 把玩家条目的字段与 overview、logins 放在同一层，
// 分两次解码：先读 overview 与 logins，再把同一份内容读成玩家条目（解码会先清空目标，顺序不能反）。
func (g *Client) Player(ctx context.Context, pid string) (*gk.PlayerDetail, error) {
	raw, err := g.c.Get(ctx, playerPath(pid), nil)
	if err != nil {
		return nil, err
	}
	out := &gk.PlayerDetail{}
	if err := decode(raw, out); err != nil {
		return nil, err
	}
	player := &gk.Player{}
	if err := decode(raw, player); err != nil {
		return nil, err
	}
	out.Player = player
	return out, nil
}

// VerifyAccount 校验游戏账号密码，返回该账号的玩家条目；密码错误时错误码为 CredentialsInvalid。
func (g *Client) VerifyAccount(ctx context.Context, account, password string) (*gk.Player, error) {
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
	out := &gk.Player{}
	if err := decode(r.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateAccount 新建一个游戏账号（服务端生成用户名、没有密码，并像注册一样建好初始玩家），返回它的玩家条目。
func (g *Client) CreateAccount(ctx context.Context, opt mgp.OperationOptions) (*gk.Player, error) {
	return operate[gk.Player](ctx, g.c, http.MethodPost, "/accounts", map[string]any{}, opt)
}

// CredentialsInvalid 账号或密码错误的错误码。
const CredentialsInvalid = "CREDENTIALS_INVALID"

// PlayerNotFound 玩家不存在的错误码。
const PlayerNotFound = "PLAYER_NOT_FOUND"

// MemoryNotFound 记忆不存在的错误码。
const MemoryNotFound = "MEMORY_NOT_FOUND"

func (g *Client) Sessions(ctx context.Context, pid, cursor string, limit int32) (*gk.SessionPage, error) {
	return get[gk.SessionPage](ctx, g.c, playerPath(pid)+"/sessions", Query{}.Page(cursor, limit))
}

func (g *Client) Kick(ctx context.Context, pid string, opt mgp.OperationOptions) (*gk.KickResult, error) {
	return operate[gk.KickResult](ctx, g.c, http.MethodPost, playerPath(pid)+"/kick", nil, opt)
}

// Ban 封禁到 until（Unix 毫秒）或永久；reason 为游戏内显示的原因。
func (g *Client) Ban(ctx context.Context, pid string, until int64, permanent bool, reason string, opt mgp.OperationOptions) (*gk.BanResult, error) {
	return operate[gk.BanResult](ctx, g.c, http.MethodPut, playerPath(pid)+"/ban", map[string]any{
		"until":     until,
		"permanent": permanent,
		"reason":    reason,
	}, opt)
}

func (g *Client) Unban(ctx context.Context, pid string, opt mgp.OperationOptions) (*gk.BanResult, error) {
	return operate[gk.BanResult](ctx, g.c, http.MethodDelete, playerPath(pid)+"/ban", nil, opt)
}

func (g *Client) SetPassword(ctx context.Context, pid, password string, opt mgp.OperationOptions) (*gk.PasswordResult, error) {
	return operate[gk.PasswordResult](ctx, g.c, http.MethodPut, playerPath(pid)+"/password", map[string]any{
		"password": password,
	}, opt)
}

func (g *Client) Rename(ctx context.Context, pid, name string, opt mgp.OperationOptions) (*gk.RenameResult, error) {
	return operate[gk.RenameResult](ctx, g.c, http.MethodPut, playerPath(pid)+"/name", map[string]any{
		"name": name,
	}, opt)
}

// ---------------- 卡牌 ----------------

// IdolCardFilter 偶像卡列表的筛选：Q 模糊匹配卡 ID、卡名或角色名（游戏原文），其余同时生效。
type IdolCardFilter struct {
	Q           string
	CharacterID string
	Rarity      int32
	IDs         []string
}

// SupportCardFilter 支援卡列表的筛选：Q 模糊匹配卡 ID 或卡名（游戏原文），其余同时生效。
type SupportCardFilter struct {
	Q      string
	Rarity int32
	IDs    []string
}

// IdolCards 偶像卡（从新到旧，带持有的皮肤）。
func (g *Client) IdolCards(ctx context.Context, pid string, f IdolCardFilter, cursor string, limit int32) (*gk.IdolCardPage, error) {
	q := Query{}.Str("q", f.Q).Str("characterId", f.CharacterID).Num("rarity", int64(f.Rarity)).List("idolCardIds", f.IDs).Page(cursor, limit)
	return get[gk.IdolCardPage](ctx, g.c, playerPath(pid)+"/idol-cards", q)
}

// SupportCards 支援卡（从新到旧；lent 为出借的卡）。
func (g *Client) SupportCards(ctx context.Context, pid string, f SupportCardFilter, cursor string, limit int32) (*gk.SupportCardPage, error) {
	q := Query{}.Str("q", f.Q).Num("rarity", int64(f.Rarity)).List("supportCardIds", f.IDs).Page(cursor, limit)
	return get[gk.SupportCardPage](ctx, g.c, playerPath(pid)+"/support-cards", q)
}

// SetRentalSupportCard 设为援助（资料展示并出借）的支援卡。
func (g *Client) SetRentalSupportCard(ctx context.Context, pid, supportCardID string, opt mgp.OperationOptions) (*gk.RentalChange, error) {
	return operate[gk.RentalChange](ctx, g.c, http.MethodPut, playerPath(pid)+"/rental-support-card", map[string]any{
		"supportCardId": supportCardID,
	}, opt)
}

// GrantAllIdolCards 把缺少的偶像卡皮肤与碎片发到礼物箱。
func (g *Client) GrantAllIdolCards(ctx context.Context, pid, message string, opt mgp.OperationOptions) (*gk.IdolCardGrant, error) {
	return operate[gk.IdolCardGrant](ctx, g.c, http.MethodPost, playerPath(pid)+"/idol-cards/grant-all", map[string]any{
		"message": message,
	}, opt)
}

// GrantAllSupportCards 把缺少的支援卡张数发到礼物箱。
func (g *Client) GrantAllSupportCards(ctx context.Context, pid, message string, opt mgp.OperationOptions) (*gk.SupportCardGrant, error) {
	return operate[gk.SupportCardGrant](ctx, g.c, http.MethodPost, playerPath(pid)+"/support-cards/grant-all", map[string]any{
		"message": message,
	}, opt)
}

// ---------------- 道具与资源 ----------------

func (g *Client) Items(ctx context.Context, pid string, kind int32, itemIDs []string, cursor string, limit int32) (*gk.ItemPage, error) {
	q := Query{}.Num("type", int64(kind)).List("itemIds", itemIDs).Page(cursor, limit)
	return get[gk.ItemPage](ctx, g.c, playerPath(pid)+"/items", q)
}

func (g *Client) AdjustItem(ctx context.Context, pid, itemID string, quantity int64, opt mgp.OperationOptions) (*gk.ChangeList, error) {
	return operate[gk.ChangeList](ctx, g.c, http.MethodPost, playerPath(pid)+"/items/adjust", map[string]any{
		"itemId":   itemID,
		"quantity": quantity,
	}, opt)
}

func (g *Client) AdjustJewel(ctx context.Context, pid string, free, paid int64, opt mgp.OperationOptions) (*gk.ChangeList, error) {
	return operate[gk.ChangeList](ctx, g.c, http.MethodPost, playerPath(pid)+"/jewels/adjust", map[string]any{
		"free": free,
		"paid": paid,
	}, opt)
}

func (g *Client) RefillActionPoint(ctx context.Context, pid string, opt mgp.OperationOptions) (*gk.ChangeList, error) {
	return operate[gk.ChangeList](ctx, g.c, http.MethodPost, playerPath(pid)+"/action-point/refill", nil, opt)
}

func (g *Client) AddExp(ctx context.Context, pid string, exp int64, opt mgp.OperationOptions) (*gk.ChangeList, error) {
	return operate[gk.ChangeList](ctx, g.c, http.MethodPost, playerPath(pid)+"/exp/add", map[string]any{
		"exp": exp,
	}, opt)
}

// ---------------- 礼物与邮件 ----------------

// GiftFilter 礼物箱的筛选：State 为 unreceived / received / expired，Source 为 game / admin，
// Expiring 只保留 24 小时内到期的礼物；空值不筛选。
type GiftFilter struct {
	State       string
	ResourceIDs []string
	Source      string
	Expiring    bool
}

func (g *Client) Gifts(ctx context.Context, pid string, f GiftFilter, cursor string, limit int32) (*gk.GiftPage, error) {
	q := Query{}.Str("state", f.State).List("resourceIds", f.ResourceIDs).Str("source", f.Source).Flag("expiring", f.Expiring).Page(cursor, limit)
	return get[gk.GiftPage](ctx, g.c, playerPath(pid)+"/gifts", q)
}

// Reward 一项奖励：资源类型、资源 ID（货币为空串）与数量。
type Reward struct {
	ResourceType int32  `json:"resourceType"`
	ResourceID   string `json:"resourceId"`
	Quantity     int64  `json:"quantity"`
}

// GiftSend 发到礼物箱的内容与来源：SourceKind 为 gm（空）、mail 或 code；gm 时 SourceID 为空即用幂等键。
type GiftSend struct {
	Rewards    []Reward `json:"rewards"`
	Message    string   `json:"message,omitempty"`
	ExpireDays int32    `json:"expireDays,omitempty"`
	SourceKind string   `json:"sourceKind,omitempty"`
	SourceID   string   `json:"sourceId,omitempty"`
}

func (g *Client) SendGift(ctx context.Context, pid string, gift GiftSend, opt mgp.OperationOptions) (*gk.GiftSendResult, error) {
	return operate[gk.GiftSendResult](ctx, g.c, http.MethodPost, playerPath(pid)+"/gifts", gift, opt)
}

// GiftRevoke 撤回某来源未领取的礼物（SourceID 与 GiftIDs 至少一项）。
type GiftRevoke struct {
	SourceKind string   `json:"sourceKind"`
	SourceID   string   `json:"sourceId,omitempty"`
	GiftIDs    []string `json:"giftIds,omitempty"`
}

func (g *Client) RevokeGifts(ctx context.Context, pid string, revoke GiftRevoke, opt mgp.OperationOptions) (*gk.GiftRevokeResult, error) {
	return operate[gk.GiftRevokeResult](ctx, g.c, http.MethodPost, playerPath(pid)+"/gifts/revoke", revoke, opt)
}

// MailCondition 条件邮件的收件条件（Unix 毫秒，0 不限）。
type MailCondition struct {
	CreatedSince    int64 `json:"createdSince,omitempty"`
	CreatedBefore   int64 `json:"createdBefore,omitempty"`
	LastLoginSince  int64 `json:"lastLoginSince,omitempty"`
	LastLoginBefore int64 `json:"lastLoginBefore,omitempty"`
}

// MailContent 保存（mail.save）的邮件内容：Target 为 all、condition 或 players（Players 为 pid）；
// Message 是礼物箱文案（玩家可见），Note 是只给管理员看的备注；奖励不能是付费钻石。
// StartTime（Unix 毫秒）在草稿里是计划的定时时间（0 不定时），在定时邮件里是开始投递的时间；
// 领取期限 ExpireDays 从开始投递时算起。
type MailContent struct {
	Target        string         `json:"target"`
	Players       []string       `json:"players,omitempty"`
	Condition     *MailCondition `json:"condition,omitempty"`
	ExcludeBanned bool           `json:"excludeBanned"`
	Rewards       []Reward       `json:"rewards"`
	Message       string         `json:"message,omitempty"`
	Note          string         `json:"note,omitempty"`
	ExpireDays    int32          `json:"expireDays"`
	StartTime     int64          `json:"startTime,omitempty"`
}

func (g *Client) mailOperation(ctx context.Context, path string, body any, opt mgp.OperationOptions) (*gk.MailResult, error) {
	return operate[gk.MailResult](ctx, g.c, http.MethodPost, path, body, opt)
}

// SendMail 发送服务端保存的草稿：从 startTime（Unix 毫秒，0 或不晚于现在为立即）起投递
// （全服邮件在领取期限内也投递给之后注册的账号）。试运行时结果的 matched 为现在匹配的人数。
func (g *Client) SendMail(ctx context.Context, mailID string, startTime int64, opt mgp.OperationOptions) (*gk.MailResult, error) {
	return g.mailOperation(ctx, "/mails/"+url.PathEscape(mailID)+"/send", map[string]int64{"startTime": startTime}, opt)
}

// 邮件的错误码：已经发送（不是草稿）、服务端没有这封邮件、已经开始投递（不能修改）、不是草稿（不能删除）、
// 不是定时邮件（不能取消定时）、草稿还没写完（不能发送）。
const (
	MailAlreadySent    = "MAIL_ALREADY_SENT"
	MailNotFound       = "MAIL_NOT_FOUND"
	MailAlreadyStarted = "MAIL_ALREADY_STARTED"
	MailNotDraft       = "MAIL_NOT_DRAFT"
	MailNotScheduled   = "MAIL_NOT_SCHEDULED"
	MailIncomplete     = "MAIL_INCOMPLETE"
)

// 邮件状态（服务端按草稿标记、开始时间、投递与撤回时间得出）。
const (
	MailDraft     = "draft"
	MailScheduled = "scheduled"
	MailSending   = "sending"
	MailSent      = "sent"
	MailRevoking  = "revoking"
	MailRevoked   = "revoked"
)

// MailPlayer 指定玩家邮件的一名收件人。
type MailPlayer struct {
	PID       string `json:"pid"`
	Name      string `json:"name"`
	DisplayID string `json:"displayId"`
}

// MailView 服务端保存的一封邮件（时间为 Unix 毫秒）：DeliveredAt 为 0 表示开始时匹配的玩家还没投递完，
// 之后 Matched / Done / Failed 为匹配、送达与失败的人数；Recipients / Received 为持有与领取了这封邮件礼物的玩家数；
// RevokedAt 不为 0 表示已撤回，RevokeDoneAt 为撤回完成的时间，Revoked 为被撤回未领取礼物的玩家数。
// *By 为执行者（X-MGP-Operator）。
type MailView struct {
	MailID        string        `json:"mailId"`
	Status        string        `json:"status"`
	Target        string        `json:"target"`
	Players       []MailPlayer  `json:"players"`
	Condition     MailCondition `json:"condition"`
	ExcludeBanned bool          `json:"excludeBanned"`
	Message       string        `json:"message"`
	Note          string        `json:"note"`
	Rewards       []Reward      `json:"rewards"`
	ExpireDays    int32         `json:"expireDays"`
	StartTime     int64         `json:"startTime"`
	LimitTime     int64         `json:"limitTime"`
	DeliveredAt   int64         `json:"deliveredAt"`
	Matched       int32         `json:"matched"`
	Done          int32         `json:"done"`
	Failed        int32         `json:"failed"`
	RevokedAt     int64         `json:"revokedAt"`
	Revoked       int32         `json:"revoked"`
	RevokeDoneAt  int64         `json:"revokeDoneAt"`
	Recipients    int32         `json:"recipients"`
	Received      int32         `json:"received"`
	CreatedBy     string        `json:"createdBy"`
	UpdatedBy     string        `json:"updatedBy"`
	SentBy        string        `json:"sentBy"`
	RevokedBy     string        `json:"revokedBy"`
	CreatedAt     int64         `json:"createdAt"`
	UpdatedAt     int64         `json:"updatedAt"`
}

// MailFilter 邮件列表的筛选：Status 为 draft、scheduled、sent（投递中、已发送、撤回中、已撤回）或空（全部）；
// Q 匹配文案、备注、奖励或邮件 ID；SentSince 不为 0 时一并统计这之后投递完的邮件数。
type MailFilter struct {
	Status    string
	Q         string
	Cursor    string
	Limit     int32
	SentSince int64
}

// MailPage 邮件列表的一页：Counts 为 all / draft / scheduled / sent 各组的数量（不受筛选影响），
// SentSince 为 MailFilter.SentSince 之后投递完的邮件数。
type MailPage struct {
	Items      []MailView       `json:"items"`
	NextCursor string           `json:"nextCursor"`
	Total      int32            `json:"total"`
	Counts     map[string]int32 `json:"counts"`
	SentSince  int32            `json:"sentSince"`
}

// ListMails 服务端保存的邮件（含草稿），按最后修改时间从新到旧。
func (g *Client) ListMails(ctx context.Context, f MailFilter) (*MailPage, error) {
	q := Query{}.Str("status", f.Status).Str("q", f.Q).Page(f.Cursor, f.Limit).Num("sentSince", f.SentSince)
	out := &MailPage{}
	if err := g.c.GetInto(ctx, "/mails", url.Values(q), out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetMail 读取服务端保存的邮件；没有时返回错误码 MailNotFound。
func (g *Client) GetMail(ctx context.Context, mailID string) (*MailView, error) {
	out := &MailView{}
	if err := g.c.GetInto(ctx, "/mails/"+url.PathEscape(mailID), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// operateMail 执行一个返回邮件的操作。
func (g *Client) operateMail(ctx context.Context, method, path string, body any, opt mgp.OperationOptions) (*MailView, error) {
	r, err := g.c.Operate(ctx, method, path, body, opt)
	if err != nil {
		return nil, err
	}
	out := &MailView{}
	if err := json.Unmarshal(r.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveMail 保存草稿（没有这封邮件时新建），或修改还没开始投递的定时邮件（内容须完整、开始时间晚于现在）；
// 已经开始投递时返回错误码 MailAlreadyStarted。
func (g *Client) SaveMail(ctx context.Context, mailID string, content MailContent, opt mgp.OperationOptions) (*MailView, error) {
	return g.operateMail(ctx, http.MethodPut, "/mails/"+url.PathEscape(mailID), content, opt)
}

// DeleteMail 删除草稿；不是草稿时返回错误码 MailNotDraft。
func (g *Client) DeleteMail(ctx context.Context, mailID string, opt mgp.OperationOptions) (*MailView, error) {
	return g.operateMail(ctx, http.MethodDelete, "/mails/"+url.PathEscape(mailID), nil, opt)
}

// CancelMail 取消定时：还没到开始时间的邮件回到草稿；已经开始投递时返回错误码 MailAlreadyStarted。
func (g *Client) CancelMail(ctx context.Context, mailID string, opt mgp.OperationOptions) (*gk.MailResult, error) {
	return g.mailOperation(ctx, "/mails/"+url.PathEscape(mailID)+"/cancel", nil, opt)
}

// RevokeMail 撤回邮件：之后不再投递给任何人，并撤回它所有尚未领取的礼物。
func (g *Client) RevokeMail(ctx context.Context, mailID string, opt mgp.OperationOptions) (*gk.MailResult, error) {
	return g.mailOperation(ctx, "/mails/"+url.PathEscape(mailID)+"/revoke", nil, opt)
}

// ---------------- 抽卡 ----------------

// GashaDraws 抽卡记录：gashaIDs 只保留这些卡池，cardIDs 只保留抽到其中任一张卡的抽卡。
func (g *Client) GashaDraws(ctx context.Context, pid string, gashaIDs, cardIDs []string, cursor string, limit int32) (*gk.GashaDrawPage, error) {
	q := Query{}.List("gashaIds", gashaIDs).List("cardIds", cardIDs).Page(cursor, limit)
	return get[gk.GashaDrawPage](ctx, g.c, playerPath(pid)+"/gasha-draws", q)
}

func (g *Client) GashaStats(ctx context.Context) (*gk.GashaStatList, error) {
	return get[gk.GashaStatList](ctx, g.c, "/gashas/stats", nil)
}

// ---------------- 记忆与培育 ----------------

// MemoryFilter 记忆列表的筛选：Produced 只保留培育结果；Q 模糊匹配记忆的偶像卡 ID、卡名或角色名（游戏原文）；
// Rarity 为偶像卡稀有度，Grades 为评价（ResultGrade）；其余同时生效。
type MemoryFilter struct {
	Produced    bool
	Q           string
	CharacterID string
	Rarity      int32
	Grades      []int32
	IdolCardIDs []string
}

// Memories 记忆（从新到旧；lent 为出借的记忆；summary 为按稀有度、按评价的条数）。
func (g *Client) Memories(ctx context.Context, pid string, f MemoryFilter, cursor string, limit int32) (*gk.MemoryPage, error) {
	q := Query{}.Flag("produced", f.Produced).Str("q", f.Q).Str("characterId", f.CharacterID).Num("rarity", int64(f.Rarity)).Nums("grades", f.Grades).List("idolCardIds", f.IdolCardIDs).Page(cursor, limit)
	return get[gk.MemoryPage](ctx, g.c, playerPath(pid)+"/memories", q)
}

// Memory 一条记忆的详情与培育记录；记忆不存在时错误码为 MemoryNotFound。
func (g *Client) Memory(ctx context.Context, pid, memoryID string) (*gk.MemoryDetail, error) {
	return get[gk.MemoryDetail](ctx, g.c, playerPath(pid)+"/memories/"+url.PathEscape(memoryID), nil)
}

// MemoryPhoto 记忆照片（相册对象 album/<publicUserId>/<id>.jpg）；不存在时错误码为 MEMORY_PHOTO_NOT_FOUND。
func (g *Client) MemoryPhoto(ctx context.Context, path string) (*gk.MemoryPhoto, error) {
	return get[gk.MemoryPhoto](ctx, g.c, "/memory-photo", Query{}.Str("path", path))
}

// SetRentalMemory 设为援助（资料展示并出借）的记忆。
func (g *Client) SetRentalMemory(ctx context.Context, pid, memoryID string, opt mgp.OperationOptions) (*gk.RentalChange, error) {
	return operate[gk.RentalChange](ctx, g.c, http.MethodPut, playerPath(pid)+"/rental-memory", map[string]any{
		"memoryId": memoryID,
	}, opt)
}

// ProduceFilter 全服培育结果的筛选（时间为 Unix 毫秒）。
type ProduceFilter struct {
	ProduceGroupID string
	ProduceID      string
	CharacterID    string
	Since          int64
	Until          int64
	Q              string
}

func (g *Client) Produces(ctx context.Context, f ProduceFilter, cursor string, limit int32) (*gk.ProducePage, error) {
	q := Query{}.Str("produceGroupId", f.ProduceGroupID).Str("produceId", f.ProduceID).Str("characterId", f.CharacterID).
		Num("since", f.Since).Num("until", f.Until).Str("q", f.Q).Page(cursor, limit)
	return get[gk.ProducePage](ctx, g.c, "/produces", q)
}

// ---------------- 竞赛、研修与塔 ----------------

// MeishiImage 玩家资料名片的图片；默认名片或没有图片时错误码为 MEISHI_IMAGE_NOT_FOUND。
func (g *Client) MeishiImage(ctx context.Context, pid string) (*gk.MeishiImage, error) {
	return get[gk.MeishiImage](ctx, g.c, playerPath(pid)+"/meishi-image", nil)
}

func (g *Client) PlayerPvp(ctx context.Context, pid string) (*gk.PlayerPvp, error) {
	return get[gk.PlayerPvp](ctx, g.c, playerPath(pid)+"/pvp", nil)
}

func (g *Client) SetPvpRate(ctx context.Context, pid string, rate int32, opt mgp.OperationOptions) (*gk.ChangeList, error) {
	return operate[gk.ChangeList](ctx, g.c, http.MethodPut, playerPath(pid)+"/pvp/rate", map[string]any{
		"rate": rate,
	}, opt)
}

func (g *Client) ResetPvpDaily(ctx context.Context, pid string, opt mgp.OperationOptions) (*gk.ChangeList, error) {
	return operate[gk.ChangeList](ctx, g.c, http.MethodPost, playerPath(pid)+"/pvp/reset-daily", nil, opt)
}

// PvpRanking 当前赛季的排名；q 搜索玩家。
func (g *Client) PvpRanking(ctx context.Context, q, cursor string, limit int32) (*gk.PvpRankingPage, error) {
	return get[gk.PvpRankingPage](ctx, g.c, "/pvp/ranking", Query{}.Str("q", q).Page(cursor, limit))
}

func (g *Client) PlayerSeminar(ctx context.Context, pid string) (*gk.PlayerSeminar, error) {
	return get[gk.PlayerSeminar](ctx, g.c, playerPath(pid)+"/seminar", nil)
}

func (g *Client) ClearSeminar(ctx context.Context, pid string, opt mgp.OperationOptions) (*gk.SeminarClearResult, error) {
	return operate[gk.SeminarClearResult](ctx, g.c, http.MethodPost, playerPath(pid)+"/seminar/clear", nil, opt)
}

func (g *Client) ResetSeminar(ctx context.Context, pid string, opt mgp.OperationOptions) (*gk.SeminarResetResult, error) {
	return operate[gk.SeminarResetResult](ctx, g.c, http.MethodPost, playerPath(pid)+"/seminar/reset", nil, opt)
}

func (g *Client) SeminarStats(ctx context.Context) (*gk.SeminarStatList, error) {
	return get[gk.SeminarStatList](ctx, g.c, "/seminar/stats", nil)
}

// TowerRanking 塔在当前排行季的排名；q 搜索玩家。
func (g *Client) TowerRanking(ctx context.Context, towerID, q, cursor string, limit int32) (*gk.TowerRankingPage, error) {
	return get[gk.TowerRankingPage](ctx, g.c, "/towers/"+url.PathEscape(towerID)+"/ranking", Query{}.Str("q", q).Page(cursor, limit))
}

// ---------------- 存档 ----------------

// ExportArchive 完整存档（JSON 文本，不含礼物箱）。
func (g *Client) ExportArchive(ctx context.Context, pid string) (*gk.ArchiveExport, error) {
	var doc struct {
		Archive json.RawMessage `json:"archive"`
	}
	if err := g.c.GetInto(ctx, playerPath(pid)+"/archive", nil, &doc); err != nil {
		return nil, err
	}
	return &gk.ArchiveExport{
		Archive: string(doc.Archive),
	}, nil
}

func (g *Client) Backups(ctx context.Context, pid string) (*gk.BackupList, error) {
	return get[gk.BackupList](ctx, g.c, playerPath(pid)+"/backups", nil)
}

func (g *Client) Backup(ctx context.Context, pid, note string, opt mgp.OperationOptions) (*gk.Backup, error) {
	return operate[gk.Backup](ctx, g.c, http.MethodPost, playerPath(pid)+"/backups", map[string]any{
		"note": note,
	}, opt)
}

func (g *Client) Restore(ctx context.Context, pid string, backupID int64, opt mgp.OperationOptions) (*gk.RestoreResult, error) {
	return operate[gk.RestoreResult](ctx, g.c, http.MethodPost, playerPath(pid)+"/backups/"+strconv.FormatInt(backupID, 10)+"/restore", nil, opt)
}

// ---------------- 数据文件 ----------------

// ConfigFile 数据文件的 MGP 路径。
type ConfigFile string

const (
	ConfigGasha      ConfigFile = "/config/gasha"
	ConfigNotice     ConfigFile = "/config/notice"
	ConfigPvpRate    ConfigFile = "/config/pvp-rate"
	ConfigHome       ConfigFile = "/config/home"
	ConfigResearch   ConfigFile = "/config/research"
	ConfigStoryEvent ConfigFile = "/config/story-event"
	ConfigLoginBonus ConfigFile = "/config/login-bonus"
)

// ConfigDocument 数据文件与其 revision。
type ConfigDocument struct {
	File     string          `json:"file"`
	Revision string          `json:"revision"`
	Content  json.RawMessage `json:"content"`
}

func (g *Client) Config(ctx context.Context, file ConfigFile) (*ConfigDocument, error) {
	out := &ConfigDocument{}
	if err := g.c.GetInto(ctx, string(file), nil, out); err != nil {
		return nil, err
	}
	return out, nil
}

// gashaDef 卡池数据文件里一个卡池用到的字段。
type gashaDef struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          int32  `json:"type"`
	BannerAssetID string `json:"bannerAssetId"`
	StartTime     int64  `json:"startTime"`
	EndTime       int64  `json:"endTime"`
	Order         int32  `json:"order"`
}

// OpenGashas 现在开放的卡池：读卡池数据文件，按 order、ID 排列（与游戏服务端的卡池顺序相同），
// 保留开始时间已到、还没结束（endTime 为 0 不结束）的卡池。
func (g *Client) OpenGashas(ctx context.Context, now time.Time) ([]*gk.GashaEntry, error) {
	doc, err := g.Config(ctx, ConfigGasha)
	if err != nil {
		return nil, err
	}
	var cfg struct {
		Gashas []gashaDef `json:"gashas"`
	}
	if err := json.Unmarshal(doc.Content, &cfg); err != nil {
		return nil, fmt.Errorf("gasha config: %w", err)
	}
	slices.SortStableFunc(cfg.Gashas, func(a, b gashaDef) int {
		return cmp.Or(cmp.Compare(a.Order, b.Order), strings.Compare(a.ID, b.ID))
	})
	ms := now.UnixMilli()
	out := []*gk.GashaEntry{}
	for _, v := range cfg.Gashas {
		if v.StartTime <= ms && (v.EndTime == 0 || ms < v.EndTime) {
			out = append(out, &gk.GashaEntry{
				Id:            v.ID,
				Name:          v.Name,
				Type:          v.Type,
				BannerAssetId: v.BannerAssetID,
				StartTime:     float64(v.StartTime),
				EndTime:       float64(v.EndTime),
			})
		}
	}
	return out, nil
}

// ConfigPublish 发布或校验数据文件的结果。
type ConfigPublish struct {
	File        string `json:"file"`
	Valid       bool   `json:"valid"`
	Error       string `json:"error"`
	Revision    string `json:"revision"`
	NewRevision string `json:"newRevision"`
	Published   bool   `json:"published"`
}

// PublishConfig 发布数据文件（卡池与公告）；试运行只校验。revision 为编辑所基于的版本。
func (g *Client) PublishConfig(ctx context.Context, file ConfigFile, content json.RawMessage, revision, comment string, opt mgp.OperationOptions) (*ConfigPublish, error) {
	body := map[string]any{
		"content": content,
	}
	if revision != "" {
		body["revision"] = revision
	}
	if comment != "" {
		body["comment"] = comment
	}
	r, err := g.c.Operate(ctx, http.MethodPut, string(file), body, opt)
	if err != nil {
		return nil, err
	}
	out := &ConfigPublish{}
	if err := json.Unmarshal(r.Data, out); err != nil {
		return nil, err
	}
	return out, nil
}
