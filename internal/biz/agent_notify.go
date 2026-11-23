package biz

import (
	"context"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/schema"
	"momoko/pkg/agentnotify"
	"momoko/pkg/agentrun"
	"momoko/pkg/common"
	"momoko/pkg/response"
)

// ———————————————————————————— 通知设置 ————————————————————————————

// agentNotifyKinds 四种通知方式，按页面上的顺序。
var agentNotifyKinds = []string{"discord", "telegram", "feishu", "qq"}

// agentNotifyFields 每种方式要填的配置项；agentNotifySecrets 是其中加密保存、只返回脱敏值的。
var (
	agentNotifyFields = map[string][]string{
		"discord":  {"url"},
		"telegram": {"token", "chat"},
		"feishu":   {"url", "secret"},
		"qq":       {"appid", "secret", "to"},
	}
	agentNotifySecrets = map[string]map[string]bool{
		"discord":  {"url": true},
		"telegram": {"token": true},
		"feishu":   {"url": true, "secret": true},
		"qq":       {"secret": true},
	}
)

// 站内消息的种类：等待批准、等你回答问题、一轮结束、出错。文案由页面按种类和参数显示。
const (
	MessageKindAgentAsk      = "agent_ask"
	MessageKindAgentQuestion = "agent_question"
	MessageKindAgentDone     = "agent_done"
	MessageKindAgentError    = "agent_error"
)

type agentNotifyChannel struct {
	Enabled bool `json:"enabled"`
	// Fields 配置项；密钥类存的是密文。
	Fields map[string]string `json:"fields"`
}

// agentNotifyConfig 一个人的通知设置。每人一行存在 agent_notify 表里，没有那一行时三个时机都开、没有通知方式。
// 以前整个系统只有一份，存在 configs 表的 agent.notify 里（JSON 格式一样），升级时归给内置超级管理员（见 claimLegacyNotify）。
type agentNotifyConfig struct {
	Ask      bool                           `json:"ask"`
	Done     bool                           `json:"done"`
	Error    bool                           `json:"error"`
	Channels map[string]*agentNotifyChannel `json:"channels"`
}

// configured 这种方式设置过（必填项填完才能保存，所以有配置项就是设置过）。
func (ch *agentNotifyChannel) configured() bool {
	return ch != nil && len(ch.Fields) > 0
}

func defaultNotify() *agentNotifyConfig {
	return &agentNotifyConfig{Ask: true, Done: true, Error: true, Channels: map[string]*agentNotifyChannel{}}
}

func notifyOf(n *gen.AgentNotify) *agentNotifyConfig {
	cfg := &agentNotifyConfig{Ask: n.OnAsk, Done: n.OnDone, Error: n.OnError, Channels: map[string]*agentNotifyChannel{}}
	for kind, ch := range n.Channels {
		cfg.Channels[kind] = &agentNotifyChannel{Enabled: ch.Enabled, Fields: ch.Fields}
	}
	return cfg
}

// loadNotify userID 的通知设置；没设置过是默认的。
func (uc *AgentUsecase) loadNotify(ctx context.Context, userID string) (*agentNotifyConfig, error) {
	n, err := uc.repo.GetAgentNotify(ctx, userID)
	if gen.IsNotFound(err) {
		return defaultNotify(), nil
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return notifyOf(n), nil
}

func (uc *AgentUsecase) saveNotifyRow(ctx context.Context, userID string, cfg *agentNotifyConfig) error {
	n := &gen.AgentNotify{ID: userID, OnAsk: cfg.Ask, OnDone: cfg.Done, OnError: cfg.Error, Channels: map[string]schema.AgentNotifyChannel{}}
	for kind, ch := range cfg.Channels {
		if ch != nil {
			n.Channels[kind] = schema.AgentNotifyChannel{Enabled: ch.Enabled, Fields: ch.Fields}
		}
	}
	return uc.repo.SaveAgentNotify(ctx, n)
}

// saveNotify 保存 userID 的通知设置。他原来的 Telegram 要是正在轮询，用的是旧配置，停掉，下次要发批准时按新配置起。
func (uc *AgentUsecase) saveNotify(ctx context.Context, userID string, cfg *agentNotifyConfig) error {
	if old, err := uc.loadNotify(ctx, userID); err == nil {
		if ch := old.Channels["telegram"]; ch.configured() {
			fields := uc.notifyFields("telegram", ch)
			uc.telegram.stopKey(telegramKey(fields["token"], fields["chat"]))
		}
	}
	if err := uc.saveNotifyRow(ctx, userID, cfg); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// notifyTarget 要改谁的通知设置：没给或者是自己，就是自己的；别人的要管理员（Agent 管理页）。
func (uc *AgentUsecase) notifyTarget(ctx context.Context, userID string) (string, error) {
	self, err := agentUserID(ctx)
	if err != nil {
		return "", err
	}
	if userID == "" || userID == self {
		return self, nil
	}
	if err := uc.requireAdmin(ctx); err != nil {
		return "", err
	}
	return userID, nil
}

// claimLegacyNotify 以前通知设置只有一份、谁的会话都按它发：配过通知方式的话归给内置超级管理员（他还没有自己的设置时），
// 然后把那一份清回默认，下次启动就不再归了。
func (uc *AgentUsecase) claimLegacyNotify(ctx context.Context) {
	raw, _ := uc.config.Get(ctx, common.ConfigAgentNotify)
	legacy := agentParse[agentNotifyConfig](raw)
	if len(legacy.Channels) == 0 {
		return
	}
	_, err := uc.repo.GetAgentNotify(ctx, BuiltinAdminUserID)
	if gen.IsNotFound(err) {
		err = uc.saveNotifyRow(ctx, BuiltinAdminUserID, &legacy)
	}
	if err != nil {
		uc.log.Errorf("把以前共用的通知设置归给内置超级管理员失败：%v", err)
		return
	}
	if err := uc.config.BatchUpdate(ctx, map[common.ConfigKey]string{common.ConfigAgentNotify: agentJSON(defaultNotify())}); err != nil {
		uc.log.Warnf("清掉以前共用的通知设置失败：%v", err)
	}
}

// notifyFields 一种方式的明文配置，发送时用。
func (uc *AgentUsecase) notifyFields(kind string, ch *agentNotifyChannel) agentnotify.Fields {
	out := agentnotify.Fields{}
	if ch == nil {
		return out
	}
	for key, value := range ch.Fields {
		if agentNotifySecrets[kind][key] {
			value = uc.decrypt(value)
		}
		out[key] = value
	}
	return out
}

func (uc *AgentUsecase) notifyChannelPB(kind string, ch *agentNotifyChannel) *v1.AgentNotifyChannel {
	out := &v1.AgentNotifyChannel{Kind: kind, Fields: map[string]string{}}
	if ch == nil || len(ch.Fields) == 0 {
		return out
	}
	out.Configured, out.Enabled = true, ch.Enabled
	for key, value := range ch.Fields {
		if agentNotifySecrets[kind][key] {
			// 脱敏只留后四位：给页面认出是哪一个，不给原文。
			value = agentTail(uc.decrypt(value))
		}
		out.Fields[key] = value
	}
	return out
}

// GetNotify 自己的通知设置。
func (uc *AgentUsecase) GetNotify(ctx context.Context) (*v1.GetAgentNotifyResponse, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	cfg, err := uc.loadNotify(ctx, userID)
	if err != nil {
		return nil, err
	}
	resp := &v1.GetAgentNotifyResponse{Events: &v1.AgentNotifyEvents{Ask: cfg.Ask, Done: cfg.Done, Error: cfg.Error}}
	for _, kind := range agentNotifyKinds {
		resp.Channels = append(resp.Channels, uc.notifyChannelPB(kind, cfg.Channels[kind]))
	}
	return resp, nil
}

func (uc *AgentUsecase) UpdateNotifyEvents(ctx context.Context, ev *v1.AgentNotifyEvents) error {
	userID, err := agentUserID(ctx)
	if err != nil {
		return err
	}
	cfg, err := uc.loadNotify(ctx, userID)
	if err != nil {
		return err
	}
	cfg.Ask, cfg.Done, cfg.Error = ev.GetAsk(), ev.GetDone(), ev.GetError()
	return uc.saveNotify(ctx, userID, cfg)
}

// mergeNotifyFields 表单里的配置和已保存的合起来（明文）：密钥类留空表示保留原值。
func (uc *AgentUsecase) mergeNotifyFields(kind string, stored *agentNotifyChannel, incoming map[string]string) (agentnotify.Fields, error) {
	keys, ok := agentNotifyFields[kind]
	if !ok {
		return nil, response.BadRequest("不认识的通知方式")
	}
	old := uc.notifyFields(kind, stored)
	out := agentnotify.Fields{}
	for _, key := range keys {
		value := strings.TrimSpace(incoming[key])
		if value == "" && agentNotifySecrets[kind][key] {
			value = old[key]
		}
		out[key] = value
	}
	return out, nil
}

// SaveNotifyChannel 保存自己的一种通知方式。必填项填完才能保存，保存后自动启用。
func (uc *AgentUsecase) SaveNotifyChannel(ctx context.Context, kind string, fields map[string]string) (*v1.AgentNotifyChannel, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	cfg, err := uc.loadNotify(ctx, userID)
	if err != nil {
		return nil, err
	}
	plain, err := uc.mergeNotifyFields(kind, cfg.Channels[kind], fields)
	if err != nil {
		return nil, err
	}
	if _, err := agentnotify.New(kind, plain); err != nil {
		return nil, agentErr(err)
	}
	ch := &agentNotifyChannel{Enabled: true, Fields: map[string]string{}}
	for key, value := range plain {
		if agentNotifySecrets[kind][key] {
			value = uc.encrypt(value)
		}
		ch.Fields[key] = value
	}
	cfg.Channels[kind] = ch
	if err := uc.saveNotify(ctx, userID, cfg); err != nil {
		return nil, err
	}
	return uc.notifyChannelPB(kind, ch), nil
}

// SetNotifyChannelEnabled 启用或停用一种通知方式：自己的；userID 是别人时要管理员（Agent 管理页）。
func (uc *AgentUsecase) SetNotifyChannelEnabled(ctx context.Context, userID, kind string, enabled bool) error {
	userID, err := uc.notifyTarget(ctx, userID)
	if err != nil {
		return err
	}
	cfg, err := uc.loadNotify(ctx, userID)
	if err != nil {
		return err
	}
	ch := cfg.Channels[kind]
	if !ch.configured() {
		return response.BadRequest("这种通知方式还没配置")
	}
	ch.Enabled = enabled
	return uc.saveNotify(ctx, userID, cfg)
}

// DeleteNotifyChannel 移除一种通知方式的配置：自己的；userID 是别人时要管理员（Agent 管理页）。
func (uc *AgentUsecase) DeleteNotifyChannel(ctx context.Context, userID, kind string) error {
	userID, err := uc.notifyTarget(ctx, userID)
	if err != nil {
		return err
	}
	cfg, err := uc.loadNotify(ctx, userID)
	if err != nil {
		return err
	}
	if !cfg.Channels[kind].configured() {
		return response.BadRequest("这种通知方式还没配置")
	}
	delete(cfg.Channels, kind)
	return uc.saveNotify(ctx, userID, cfg)
}

// ListNotify Agent 管理页：所有人的通知设置，只列设置过通知方式的人；密钥只给脱敏值。
func (uc *AgentUsecase) ListNotify(ctx context.Context) (*v1.ListAgentNotifyResponse, error) {
	if err := uc.requireAdmin(ctx); err != nil {
		return nil, err
	}
	rows, err := uc.repo.ListAgentNotify(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp := &v1.ListAgentNotifyResponse{}
	ids := make([]string, 0, len(rows))
	for _, n := range rows {
		cfg := notifyOf(n)
		item := &v1.AgentNotifyOwner{UserId: n.ID, Events: &v1.AgentNotifyEvents{Ask: cfg.Ask, Done: cfg.Done, Error: cfg.Error}}
		for _, kind := range agentNotifyKinds {
			if ch := cfg.Channels[kind]; ch.configured() {
				item.Channels = append(item.Channels, uc.notifyChannelPB(kind, ch))
			}
		}
		if len(item.Channels) > 0 {
			resp.Items = append(resp.Items, item)
			ids = append(ids, n.ID)
		}
	}
	if resp.Users, err = uc.agentUsersPB(ctx, ids); err != nil {
		return nil, err
	}
	return resp, nil
}

// notifyCounts 通知标签上的数量：自己的是四种方式、启用了几种；all 时是所有人设置过的通知方式、其中启用着的。
func (uc *AgentUsecase) notifyCounts(ctx context.Context, userID string, all bool) (total, enabled int, err error) {
	if !all {
		cfg, err := uc.loadNotify(ctx, userID)
		if err != nil {
			return 0, 0, err
		}
		for _, kind := range agentNotifyKinds {
			if ch := cfg.Channels[kind]; ch.configured() && ch.Enabled {
				enabled++
			}
		}
		return len(agentNotifyKinds), enabled, nil
	}
	rows, err := uc.repo.ListAgentNotify(ctx)
	if err != nil {
		return 0, 0, ErrSystem(err)
	}
	for _, n := range rows {
		for _, ch := range n.Channels {
			if len(ch.Fields) > 0 {
				total++
				if ch.Enabled {
					enabled++
				}
			}
		}
	}
	return total, enabled, nil
}

// TestNotifyChannel 发一条测试：用表单里的配置，留空的密钥用自己已保存的。
func (uc *AgentUsecase) TestNotifyChannel(ctx context.Context, kind string, fields map[string]string) error {
	userID, err := agentUserID(ctx)
	if err != nil {
		return err
	}
	cfg, err := uc.loadNotify(ctx, userID)
	if err != nil {
		return err
	}
	plain, err := uc.mergeNotifyFields(kind, cfg.Channels[kind], fields)
	if err != nil {
		return err
	}
	sender, err := agentnotify.New(kind, plain)
	if err != nil {
		return agentErr(err)
	}
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
	defer cancel()
	return agentErr(sender.Send(sendCtx, agentnotify.Message{
		Title: "Momoko 通知测试",
		Lines: []string{"这是一条测试消息，收到就说明配置没问题。"},
	}))
}

// ———————————————————————————— 发通知 ————————————————————————————

// notify 一个会话到了该通知人的时机（event：ask / question / done / error）：站内消息照常发，
// 外部渠道按会话主人自己的时机开关和各自的启用状态发（提问和批准共用「等待批准」那个开关）。
// detail 是要批准的命令、问的问题、结束说明或出错原因。
func (uc *AgentUsecase) notify(sessionID, event, detail string, approval *agentApproval) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	s, err := uc.repo.GetAgentSession(ctx, sessionID)
	if err != nil {
		return
	}
	kind := map[string]string{
		"ask": MessageKindAgentAsk, "question": MessageKindAgentQuestion, "done": MessageKindAgentDone, "error": MessageKindAgentError,
	}[event]
	_ = uc.messages.CreateMessages(ctx, []*UserMessage{{
		UserID:   s.UserID,
		Category: MessageCategoryNotice,
		Kind:     kind,
		Params:   map[string]string{"sessionId": s.ID, "title": s.Title, "detail": detail},
	}})

	cfg, err := uc.loadNotify(ctx, s.UserID)
	if err != nil {
		return
	}
	if on := map[string]bool{"ask": cfg.Ask, "question": cfg.Ask, "done": cfg.Done, "error": cfg.Error}[event]; !on {
		return
	}
	var msg *agentnotify.Message
	for _, channel := range agentNotifyKinds {
		ch := cfg.Channels[channel]
		if !ch.configured() || !ch.Enabled {
			continue
		}
		fields := uc.notifyFields(channel, ch)
		sender, err := agentnotify.New(channel, fields)
		if err != nil {
			continue
		}
		if msg == nil {
			msg = uc.notifyMessage(ctx, s, event, detail, approval)
		}
		if err := sender.Send(ctx, *msg); err != nil {
			uc.log.Warnf("agent notify via %s: %v", channel, err)
			continue
		}
		// 批准按钮点了之后要靠轮询收回来：有等着回答的批准时才轮询。
		if channel == "telegram" && approval != nil {
			uc.telegram.ensure(uc, fields["token"], fields["chat"])
		}
	}
}

// notifyMessage 发到外部渠道的那条消息：标题、Agent 和工作区、命令或说明、打开会话的链接、批准用的动作。
// 外部渠道没有 Momoko 的页面，文案只能在这里拼好。
func (uc *AgentUsecase) notifyMessage(ctx context.Context, s *gen.AgentSession, event, detail string, approval *agentApproval) *agentnotify.Message {
	title := s.Title
	if title == "" {
		title = "新会话"
	}
	msg := &agentnotify.Message{}
	switch event {
	case "ask":
		msg.Title = "「" + title + "」在等你批准"
	case "question":
		msg.Title = "「" + title + "」有问题要问你"
	case "done":
		msg.Title = "「" + title + "」这一轮结束了"
	default:
		msg.Title = "「" + title + "」出错了"
	}
	names := uc.loadNames(ctx)
	if name, _ := names.runtime(s.RuntimeID); name != "" {
		msg.Lines = append(msg.Lines, "Agent："+name)
	}
	if w := names.workspaces[s.WorkspaceID]; w != nil {
		msg.Lines = append(msg.Lines, "工作区："+w.Name)
	}
	if approval != nil && approval.perm.Title != "" && approval.perm.Title != detail {
		msg.Lines = append(msg.Lines, approval.perm.Title)
	}
	if detail != "" {
		msg.Lines = append(msg.Lines, detail)
	}
	// 链接用系统设置里的对外地址；没设置时用发起这一轮的那个页面的地址。
	site, _ := uc.config.Get(ctx, common.ConfigSiteExternalURL)
	if site == "" {
		site = s.Origin
	}
	site = strings.TrimRight(site, "/")
	if site != "" {
		msg.Link = site + "/agent/code?session=" + url.QueryEscape(s.ID)
	}
	if approval != nil {
		for i, opt := range approval.perm.Options {
			action := agentnotify.Action{
				Label: opt.Name,
				Data:  "ap:" + approval.token + ":" + strconv.Itoa(i),
				Deny:  strings.HasPrefix(opt.Kind, "reject"),
			}
			if site != "" {
				action.URL = site + "/public/agent/approval/" + approval.token + "?option=" + url.QueryEscape(opt.ID)
			}
			msg.Actions = append(msg.Actions, action)
		}
	}
	return msg
}

// ———————————————————————————— 批准令牌 ————————————————————————————

// agentApproval 一张等人回答的批准卡片对外的凭证：通知里的链接和 Telegram 按钮凭令牌回答，不需要登录。
type agentApproval struct {
	token     string
	sessionID string
	perm      agentrun.Permission
	created   time.Time
	resolved  bool
	// outcome 回答时选中的选项名；这一轮被停止时为空。
	outcome string
}

// agentApprovals 批准令牌，按令牌和会话两边建了索引。回答过的留一天（公开批准页还要显示结果），之后清掉。
type agentApprovals struct {
	mu        sync.Mutex
	byToken   map[string]*agentApproval
	bySession map[string]map[string]*agentApproval
}

func newAgentApprovals() agentApprovals {
	return agentApprovals{byToken: map[string]*agentApproval{}, bySession: map[string]map[string]*agentApproval{}}
}

func (a *agentApprovals) add(sessionID string, p *agentrun.Permission) *agentApproval {
	ap := &agentApproval{token: strings.ReplaceAll(uuid.NewString(), "-", ""), sessionID: sessionID, perm: *p, created: time.Now()}
	a.mu.Lock()
	defer a.mu.Unlock()
	// 添加时顺手清掉一天前的。
	for token, old := range a.byToken {
		if time.Since(old.created) > 24*time.Hour {
			delete(a.byToken, token)
			delete(a.bySession[old.sessionID], token)
		}
	}
	a.byToken[ap.token] = ap
	if a.bySession[sessionID] == nil {
		a.bySession[sessionID] = map[string]*agentApproval{}
	}
	a.bySession[sessionID][ap.token] = ap
	return ap
}

func (a *agentApprovals) find(token string) (agentApproval, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	ap := a.byToken[token]
	if ap == nil {
		return agentApproval{}, false
	}
	return *ap, true
}

// settle 一张卡片有了结果：optionID 为空表示这一轮被停止。
func (a *agentApprovals) settle(sessionID, requestID, optionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ap := range a.bySession[sessionID] {
		if ap.perm.ID != requestID {
			continue
		}
		ap.resolved = true
		for _, opt := range ap.perm.Options {
			if optionID != "" && opt.ID == optionID {
				ap.outcome = opt.Name
			}
		}
	}
}

// dropSession 会话的进程没了：它所有还在等的卡片都不会再有结果。
func (a *agentApprovals) dropSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ap := range a.bySession[sessionID] {
		ap.resolved = true
	}
}

// pending 还有没有等着回答的卡片。
func (a *agentApprovals) pending() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, ap := range a.byToken {
		if !ap.resolved {
			return true
		}
	}
	return false
}

func (uc *AgentUsecase) settleApproval(sessionID, requestID, optionID string) {
	uc.approvals.settle(sessionID, requestID, optionID)
	uc.telegram.idle(uc)
}

// resolvePermission 回答一张批准卡片。页面、公开批准页、Telegram 按钮都走这里。
func (uc *AgentUsecase) resolvePermission(sessionID, requestID, optionID string) error {
	l := uc.live(sessionID)
	if l == nil {
		return response.BadRequest("这个批准已经处理过了，或者这一轮已经结束")
	}
	if err := l.session.Resolve(requestID, optionID); err != nil {
		return agentErr(err)
	}
	uc.settleApproval(sessionID, requestID, optionID)
	return nil
}

// PublicApproval 公开批准页：凭令牌看要批准的是什么。令牌不存在按过期处理。
func (uc *AgentUsecase) PublicApproval(ctx context.Context, token string) (*v1.PublicAgentApprovalResponse, error) {
	a, ok := uc.approvals.find(token)
	if !ok {
		return &v1.PublicAgentApprovalResponse{State: "expired"}, nil
	}
	resp := &v1.PublicAgentApprovalResponse{Title: a.perm.Title, Detail: a.perm.Command, State: "pending", Outcome: a.outcome}
	if a.resolved {
		resp.State = "resolved"
	}
	for _, opt := range a.perm.Options {
		resp.Options = append(resp.Options, &v1.AgentChoice{Id: opt.ID, Name: opt.Name, Kind: opt.Kind})
	}
	if s, err := uc.repo.GetAgentSession(ctx, a.sessionID); err == nil {
		names := uc.loadNames(ctx)
		resp.SessionTitle = s.Title
		resp.RuntimeName, _ = names.runtime(s.RuntimeID)
		if w := names.workspaces[s.WorkspaceID]; w != nil {
			resp.WorkspaceName = w.Name
		}
	}
	return resp, nil
}

func (uc *AgentUsecase) PublicResolveApproval(_ context.Context, token, optionID string) error {
	a, ok := uc.approvals.find(token)
	if !ok || a.resolved {
		return response.BadRequest("这个批准已经处理过了，或者这一轮已经结束")
	}
	return uc.resolvePermission(a.sessionID, a.perm.ID, optionID)
}

// ———————————————————————————— Telegram 按钮 ————————————————————————————

// agentTelegram Telegram 消息里的批准按钮点了之后要靠长轮询收回来。只在发出过带按钮的批准、
// 而且还有没回答的批准时轮询；都回答完了就停。每个人的通知设置各是各的，一份配置（机器人 + 聊天）一个轮询。
type agentTelegram struct {
	mu      sync.Mutex
	polling map[string]context.CancelFunc
}

func telegramKey(token, chat string) string {
	return token + "\x00" + chat
}

// ensure 按这份配置轮询；已经在按同一份配置轮询就不动。
func (t *agentTelegram) ensure(uc *AgentUsecase, token, chat string) {
	key := telegramKey(token, chat)
	t.mu.Lock()
	defer t.mu.Unlock()
	if _, ok := t.polling[key]; ok {
		return
	}
	if t.polling == nil {
		t.polling = map[string]context.CancelFunc{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.polling[key] = cancel
	go agentnotify.PollTelegram(ctx, token, chat, uc.onTelegram)
}

// stopKey 停掉按这份配置的轮询（有人改了通知设置，旧配置不再用了）。
func (t *agentTelegram) stopKey(key string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if cancel, ok := t.polling[key]; ok {
		cancel()
		delete(t.polling, key)
	}
}

// idle 没有等着回答的批准了就停止轮询。
func (t *agentTelegram) idle(uc *AgentUsecase) {
	if !uc.approvals.pending() {
		t.stop()
	}
}

func (t *agentTelegram) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	for key, cancel := range t.polling {
		cancel()
		delete(t.polling, key)
	}
}

// onTelegram 有人点了 Telegram 消息里的批准按钮。返回给他看的提示，以及这条消息的按钮要不要收起来。
func (uc *AgentUsecase) onTelegram(cb agentnotify.TelegramCallback) (string, bool) {
	parts := strings.Split(cb.Data, ":")
	if len(parts) != 3 || parts[0] != "ap" {
		return "", false
	}
	a, ok := uc.approvals.find(parts[1])
	index, err := strconv.Atoi(parts[2])
	if !ok || a.resolved || err != nil || index < 0 || index >= len(a.perm.Options) {
		return "这个批准已经处理过了", true
	}
	opt := a.perm.Options[index]
	if err := uc.resolvePermission(a.sessionID, a.perm.ID, opt.ID); err != nil {
		return err.Error(), true
	}
	return "已选择：" + opt.Name + "（" + cb.From + "）", true
}
