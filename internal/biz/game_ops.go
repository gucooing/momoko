package biz

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entmail "momoko/internal/data/ent/gen/gamemail"
	entbatch "momoko/internal/data/ent/gen/giftcodebatch"
	entredeem "momoko/internal/data/ent/gen/giftcoderedeem"
	entschema "momoko/internal/data/ent/schema"
	"momoko/pkg/mgp"
	"momoko/pkg/response"
)

var (
	ErrGameMailNotFound    = response.BadRequest(404, "邮件不存在")
	ErrGameMailState       = response.BadRequest(409, "邮件当前的状态不能这样操作，请刷新后再试")
	ErrGameMailIncomplete  = response.BadRequest(400, "邮件还没写完：需要标题、至少一项奖励和收件人")
	ErrGiftBatchNotFound   = response.BadRequest(404, "兑换码批次不存在")
	ErrGiftCodeTaken       = response.BadRequest(409, "这台服务器上已经有这个兑换码")
	ErrGiftBatchInvalid    = response.BadRequest(400, "兑换码批次参数无效")
	ErrGiftRedeemNoAccount = response.BadRequest(400, "没有绑定这台服务器的游戏账号")
	ErrGiftRedeemEmpty     = response.BadRequest(400, "请输入兑换码")
)

const (
	mailCommand       = "mail.broadcast"
	mailRevokeCommand = "mail.revoke"
	giftCommand       = "gift.send"
	giftsReport       = "gifts"

	// mailStatsTTL 已发送邮件的领取统计缓存多久后重新向 MGP 报表查询。
	mailStatsTTL = 60 * time.Second
	// mailDeliverTimeout 一次后台投递 / 撤回最长的执行时间。
	mailDeliverTimeout = 5 * time.Minute
	mailLoopEvery      = 15 * time.Second

	mailTitleMax   = 64
	mailBodyMax    = 2000
	mailPlayersMax = 1000
	rewardsMax     = 20

	giftNameMax       = 64
	giftMessageMax    = 64
	giftQuantityMax   = 10000
	giftLimitMax      = 1_000_000
	giftPerPlayerMax  = 1000
	giftEligibleMax   = 3650
	giftRecentRedeems = 20

	// 兑换频率限制：每个通行证用户每分钟最多尝试的次数（被限流的尝试不计）。
	redeemRateLimit  = 10
	redeemRateWindow = time.Minute
)

// 兑换结果（gift_code_redeem.result）。
const (
	RedeemSuccess         = "success"
	RedeemAlreadyRedeemed = "already_redeemed"
	RedeemExpired         = "expired"
	RedeemNotStarted      = "not_started"
	RedeemExhausted       = "exhausted"
	RedeemDisabled        = "disabled"
	RedeemNotEligible     = "not_eligible"
	RedeemBanned          = "banned"
	RedeemRateLimited     = "rate_limited"
	RedeemNotFound        = "not_found"
	RedeemCodeUsed        = "code_used"
	RedeemChannelClosed   = "channel_closed"
	RedeemFailed          = "failed"
)

// 兑换入口。游戏内（game）入口需要游戏服务端把玩家输入的码转给 Momoko，MGP v1 没有这条通道，
// 所以 game 只能作为批次设置保存，目前只有通行证（passport）会真正调用兑换。
const (
	GiftChannelPassport = "passport"
	GiftChannelGame     = "game"
	GiftChannelAdmin    = "admin"
)

// GameMailFilter 邮件列表筛选。
type GameMailFilter struct {
	ServerID   string
	Statuses   []string
	Q          string
	Page, Size int
}

// GiftBatchFilter 兑换码批次筛选。
type GiftBatchFilter struct {
	ServerID string
	// State active 进行中（含未开始）/ ended 已结束（过期、用完、停用）；空为全部
	State      string
	Q          string
	Now        time.Time
	Page, Size int
}

// GiftRedeemFilter 兑换记录筛选。
type GiftRedeemFilter struct {
	ServerID string
	BatchID  string
	// Result success / failed（除 success 外）；空为全部
	Result     string
	Page, Size int
}

// GiftRedeemStats 兑换统计（统计卡片）。
type GiftRedeemStats struct {
	TodaySuccess  int
	TodayByBatch  map[string]int
	TotalSuccess  int
	FirstSuccess  *time.Time
	TodayFailures map[string]int
}

// GameOpsRepo 邮件补偿与兑换码的持久化（game_mail、gift_code_batch、gift_code、gift_code_redeem）。
type GameOpsRepo interface {
	CreateMail(ctx context.Context, m *gen.GameMail) (*gen.GameMail, error)
	// SaveMail 写回邮件的全部可修改字段。
	SaveMail(ctx context.Context, m *gen.GameMail) (*gen.GameMail, error)
	GetMail(ctx context.Context, id string) (*gen.GameMail, error)
	DeleteMail(ctx context.Context, id string) error
	ListMails(ctx context.Context, f GameMailFilter) ([]*gen.GameMail, int, error)
	// CountMails 各分组的数量：all / draft / scheduled / sent（含 sending、failed、revoking、revoked）。
	CountMails(ctx context.Context, serverID string) (map[string]int32, error)
	// CountMailsSentSince 在 since 之后发送成功（含之后撤回）的邮件数。
	CountMailsSentSince(ctx context.Context, serverID string, since time.Time) (int, error)
	// SwitchMailStatus 仅当邮件处于 from 中的某个状态时改为 to；返回是否改成功（并发安全的状态转换）。
	SwitchMailStatus(ctx context.Context, id string, from []string, to string) (bool, error)
	// DueMails 到点的定时邮件。
	DueMails(ctx context.Context, now time.Time, limit int) ([]*gen.GameMail, error)
	// MailsInStatus 处于某些状态的全部邮件（启动时恢复中断的投递）。
	MailsInStatus(ctx context.Context, statuses []string) ([]*gen.GameMail, error)
	// SaveMailMatched 只写回试运行匹配的人数与时间。
	SaveMailMatched(ctx context.Context, id string, matched int, at time.Time) (*gen.GameMail, error)
	// SaveMailStats 写回领取统计缓存。
	SaveMailStats(ctx context.Context, id string, recipients, claimed int, at time.Time) error

	// CreateBatch 创建批次与一次性码（同一事务）。
	CreateBatch(ctx context.Context, b *gen.GiftCodeBatch, codes []string) (*gen.GiftCodeBatch, error)
	SaveBatch(ctx context.Context, b *gen.GiftCodeBatch) (*gen.GiftCodeBatch, error)
	GetBatch(ctx context.Context, id string) (*gen.GiftCodeBatch, error)
	ListBatches(ctx context.Context, f GiftBatchFilter) ([]*gen.GiftCodeBatch, int, error)
	// CountBatches all / active / ended。
	CountBatches(ctx context.Context, serverID string, now time.Time) (map[string]int32, error)
	BatchNames(ctx context.Context, ids []string) (map[string]string, error)
	// CodesTaken 返回 codes 里在这台服务器上已经存在的码（通用码与一次性码）。
	CodesTaken(ctx context.Context, serverID string, codes []string) (map[string]struct{}, error)
	// FindCode 按码查找批次；一次性码同时返回码本身。
	FindCode(ctx context.Context, serverID, code string) (*gen.GiftCodeBatch, *gen.GiftCode, error)
	GetCode(ctx context.Context, id string) (*gen.GiftCode, error)
	// ReserveUse 在未停用且未用完时把已用次数加一；返回是否成功。
	ReserveUse(ctx context.Context, batchID string) (bool, error)
	ReleaseUse(ctx context.Context, batchID string) error
	// ClaimCode 在一次性码未被使用时标记为 pid 使用；返回是否成功。
	ClaimCode(ctx context.Context, codeID, pid, name string, at time.Time) (bool, error)
	ReleaseCode(ctx context.Context, codeID string) error
	ListCodes(ctx context.Context, batchID, state string, page, size int) ([]*gen.GiftCode, int, error)

	CreateRedeem(ctx context.Context, r *gen.GiftCodeRedeem) (*gen.GiftCodeRedeem, error)
	ListRedeems(ctx context.Context, f GiftRedeemFilter) ([]*gen.GiftCodeRedeem, int, error)
	CountRedeems(ctx context.Context, batchID string) (success, failure int, err error)
	// CountPlayerRedeems 某玩家在某批次成功兑换的次数。
	CountPlayerRedeems(ctx context.Context, serverID, batchID, pid string) (int, error)
	// CountAttempts 某通行证用户在 since 之后的兑换尝试次数（不含被限流的）。
	CountAttempts(ctx context.Context, passportUserID string, since time.Time) (int, error)
	RedeemStats(ctx context.Context, serverID string, dayStart time.Time) (*GiftRedeemStats, error)
}

// GameOpsUsecase GM 运营里的邮件补偿与兑换码：
// 邮件经 MGP mail.broadcast 投递、mail.revoke 撤回，定时邮件由后台循环到点发送；
// 兑换码在 Momoko 校验，兑换成功用 gift.send（sourceKind=code）发放，供通行证「自助服务」调用。
// 发送、撤回与兑换发放都经 GmUsecase.Execute，写进 GM 记录（source = mail:<id> / code:<batchId>）。
type GameOpsUsecase struct {
	gm   *GmUsecase
	repo GameOpsRepo
	log  *log.Helper

	// locks 串行化同一批次、同一通行证用户的兑换，避免并发超发或绕过频率限制。
	locks sync.Map
}

func NewGameOpsUsecase(gm *GmUsecase, repo GameOpsRepo, logger log.Logger) *GameOpsUsecase {
	uc := &GameOpsUsecase{
		gm:   gm,
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/game_ops")),
	}
	go uc.mailLoop()
	return uc
}

func (uc *GameOpsUsecase) lock(key string) func() {
	l, _ := uc.locks.LoadOrStore(key, &sync.Mutex{})
	mu := l.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// ---------------- 通用转换 ----------------

func rewardsOf(in []*v1.GameReward) []entschema.GameReward {
	out := make([]entschema.GameReward, 0, len(in))
	for _, r := range in {
		if r == nil {
			continue
		}
		out = append(out, entschema.GameReward{ResourceType: r.ResourceType, ResourceID: strings.TrimSpace(r.ResourceId), Quantity: r.Quantity})
	}
	return out
}

func rewardsProto(in []entschema.GameReward) []*v1.GameReward {
	out := make([]*v1.GameReward, 0, len(in))
	for _, r := range in {
		out = append(out, &v1.GameReward{ResourceType: r.ResourceType, ResourceId: r.ResourceID, Quantity: r.Quantity})
	}
	return out
}

func tsOf(t *time.Time) *timestamppb.Timestamp {
	if t == nil || t.IsZero() {
		return nil
	}
	return timestamppb.New(*t)
}

func timeOf(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil || !ts.IsValid() || ts.AsTime().Unix() <= 0 {
		return nil
	}
	t := ts.AsTime()
	return &t
}

// validText 检查文本长度与控制字符（换行允许时 multiline 为真）。
func validText(s string, max int, multiline bool) bool {
	if !utf8.ValidString(s) || utf8.RuneCountInString(s) > max {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) && !(multiline && (r == '\n' || r == '\r' || r == '\t')) {
			return false
		}
	}
	return true
}

// validateRewards 校验奖励：1–20 项、数量为正；forbidden 为不允许的资源类型。
func validateRewards(rewards []entschema.GameReward, allowEmpty bool, forbidden ...int32) error {
	if len(rewards) == 0 && !allowEmpty {
		return response.BadRequest(400, "至少需要一项奖励")
	}
	if len(rewards) > rewardsMax {
		return response.BadRequest(400, "奖励最多 20 项")
	}
	for _, r := range rewards {
		if r.Quantity <= 0 {
			return response.BadRequest(400, "奖励数量必须大于 0")
		}
		if slices.Contains(forbidden, r.ResourceType) {
			return response.BadRequest(400, "付费钻石不能通过邮件发放")
		}
	}
	return nil
}

// errText 把执行错误转成存储与展示用的「错误码: 信息」。
func errText(err error) string {
	var re response.Error
	if errors.As(err, &re) {
		return re.Message
	}
	return err.Error()
}

// ---------------- 邮件补偿 ----------------

// mailOf 读取邮件并确认属于这台服务器。
func (uc *GameOpsUsecase) mailOf(ctx context.Context, serverID, id string) (*gen.GameMail, error) {
	m, err := uc.repo.GetMail(ctx, id)
	if gen.IsNotFound(err) || (err == nil && m.ServerID != serverID) {
		return nil, ErrGameMailNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return m, nil
}

// GameMailProto 转为接口结构。
func GameMailProto(m *gen.GameMail) *v1.GameMail {
	out := &v1.GameMail{
		Id:                m.ID,
		ServerId:          m.ServerID,
		Title:             m.Title,
		Body:              m.Body,
		Rewards:           rewardsProto(m.Rewards),
		Target:            string(m.Target),
		ExcludeBanned:     m.ExcludeBanned,
		ExpireDays:        int32(m.ExpireDays),
		ScheduledAt:       tsOf(m.ScheduledAt),
		Status:            string(m.Status),
		RecipientCount:    int32(m.RecipientCount),
		ClaimedCount:      int32(m.ClaimedCount),
		StatsAt:           tsOf(m.StatsAt),
		Error:             m.Error,
		ExecutionId:       m.ExecutionID,
		CreatedByName:     m.CreatedByName,
		UpdatedByName:     m.UpdatedByName,
		SentByName:        m.SentByName,
		SentAt:            tsOf(m.SentAt),
		RevokedByName:     m.RevokedByName,
		RevokedAt:         tsOf(m.RevokedAt),
		CreateTime:        timestamppb.New(m.CreateTime),
		UpdateTime:        timestamppb.New(m.UpdateTime),
		MatchedCount:      int32(m.MatchedCount),
		MatchedAt:         tsOf(m.MatchedAt),
		FailedCount:       int32(m.FailedCount),
		RevokedCount:      int32(m.RevokedCount),
		RevokeExecutionId: m.RevokeExecutionID,
		Condition: &v1.GameMailCondition{
			CreatedSince:    m.Condition.CreatedSince,
			CreatedBefore:   m.Condition.CreatedBefore,
			LastLoginSince:  m.Condition.LastLoginSince,
			LastLoginBefore: m.Condition.LastLoginBefore,
		},
	}
	for _, p := range m.Players {
		out.Players = append(out.Players, &v1.GameTarget{Pid: p.PID, Name: p.Name, DisplayId: p.DisplayID})
	}
	return out
}

// paidJewelType 返回 manifest 里付费钻石的资源类型（邮件不能发放）；不知道时返回 false。
func (uc *GameOpsUsecase) paidJewelType(ctx context.Context, s *gen.GameServer) (int32, bool) {
	m, err := uc.gm.Servers().Manifest(ctx, s)
	if err != nil {
		return 0, false
	}
	v, ok := m.ResourceType["jewelPaid"]
	return v, ok
}

// applyMailInput 把编辑内容写进邮件（草稿可以不完整，发送时再检查）。
func (uc *GameOpsUsecase) applyMailInput(ctx context.Context, s *gen.GameServer, m *gen.GameMail, req *v1.SaveGameMailRequest) error {
	title := strings.TrimSpace(req.Title)
	if !validText(title, mailTitleMax, false) {
		return response.BadRequest(400, "标题最多 64 字，且不能包含控制字符")
	}
	body := strings.TrimSpace(req.Body)
	if !validText(body, mailBodyMax, true) {
		return response.BadRequest(400, "正文最多 2000 字")
	}
	rewards := rewardsOf(req.Rewards)
	var forbidden []int32
	if paid, ok := uc.paidJewelType(ctx, s); ok {
		forbidden = append(forbidden, paid)
	}
	if err := validateRewards(rewards, true, forbidden...); err != nil {
		return err
	}
	target := req.Target
	if target == "" {
		target = "all"
	}
	if target != "all" && target != "condition" && target != "players" {
		return response.BadRequest(400, "收件人范围无效")
	}
	players := make([]entschema.GmTarget, 0, len(req.Players))
	seen := map[string]struct{}{}
	for _, p := range req.Players {
		if p == nil || strings.TrimSpace(p.Pid) == "" {
			continue
		}
		pid := strings.TrimSpace(p.Pid)
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		players = append(players, entschema.GmTarget{PID: pid, Name: p.Name, DisplayID: p.DisplayId})
	}
	if len(players) > mailPlayersMax {
		return response.BadRequest(400, "指定玩家最多 1000 名")
	}
	var cond entschema.GameMailCondition
	if c := req.Condition; c != nil {
		cond = entschema.GameMailCondition{CreatedSince: c.CreatedSince, CreatedBefore: c.CreatedBefore, LastLoginSince: c.LastLoginSince, LastLoginBefore: c.LastLoginBefore}
	}
	if cond.CreatedSince < 0 || cond.CreatedBefore < 0 || cond.LastLoginSince < 0 || cond.LastLoginBefore < 0 ||
		(cond.CreatedSince > 0 && cond.CreatedBefore > 0 && cond.CreatedSince >= cond.CreatedBefore) ||
		(cond.LastLoginSince > 0 && cond.LastLoginBefore > 0 && cond.LastLoginSince >= cond.LastLoginBefore) {
		return response.BadRequest(400, "收件条件的时间范围无效")
	}
	if req.ExpireDays < 0 || req.ExpireDays > 365 {
		return response.BadRequest(400, "领取期限为 0–365 天")
	}
	if target != "players" {
		players = players[:0]
	}
	if target != "condition" {
		cond = entschema.GameMailCondition{}
	}
	recipientsChanged := string(m.Target) != target || m.ExcludeBanned != req.ExcludeBanned || m.Condition != cond || !slices.EqualFunc(m.Players, players, func(a, b entschema.GmTarget) bool { return a.PID == b.PID })
	m.Title, m.Body, m.Rewards = title, body, rewards
	m.Target, m.Players, m.Condition = entmail.Target(target), players, cond
	m.ExcludeBanned, m.ExpireDays = req.ExcludeBanned, int(req.ExpireDays)
	if recipientsChanged {
		m.MatchedCount, m.MatchedAt = 0, nil
	}
	return nil
}

// mailReady 检查邮件能不能试运行或发送。
func mailReady(m *gen.GameMail) error {
	if m.Title == "" || len(m.Rewards) == 0 || (m.Target == entmail.TargetPlayers && len(m.Players) == 0) {
		return ErrGameMailIncomplete
	}
	return nil
}

// mailParams 生成 mail.broadcast 的参数（邮件 id 即 MGP mailId）。
func mailParams(m *gen.GameMail) json.RawMessage {
	pids := make([]string, 0, len(m.Players))
	for _, p := range m.Players {
		pids = append(pids, p.PID)
	}
	rewards := m.Rewards
	if rewards == nil {
		rewards = []entschema.GameReward{}
	}
	params := map[string]any{
		"mailId":        m.ID,
		"target":        string(m.Target),
		"excludeBanned": m.ExcludeBanned,
		"rewards":       rewards,
		"message":       m.Title,
		"expireDays":    m.ExpireDays,
	}
	switch m.Target {
	case entmail.TargetPlayers:
		params["players"] = pids
	case entmail.TargetCondition:
		params["condition"] = m.Condition
	}
	raw, _ := json.Marshal(params)
	return raw
}

// mailSummary GM 记录里可展示的摘要（不含参数原文）。
func mailSummary(m *gen.GameMail) map[string]string {
	out := map[string]string{
		"mailId":  m.ID,
		"title":   m.Title,
		"target":  string(m.Target),
		"rewards": itoa(len(m.Rewards)),
	}
	if m.Target == entmail.TargetPlayers {
		out["players"] = itoa(len(m.Players))
	}
	return out
}

// mailBroadcastSummary mail.broadcast / mail.revoke 的 summary。
type mailBroadcastSummary struct {
	MailID    string `json:"mailId"`
	Matched   int    `json:"matched"`
	Delivered int    `json:"delivered"`
	Gifts     int    `json:"gifts"`
	Failed    int    `json:"failed"`
}

func parseMailSummary(raw json.RawMessage) mailBroadcastSummary {
	var r mgp.CommandResult
	var out mailBroadcastSummary
	if json.Unmarshal(raw, &r) == nil && len(r.Summary) > 0 {
		_ = json.Unmarshal(r.Summary, &out)
	}
	return out
}

// giftStats MGP gifts 报表。
type giftStats struct {
	Gifts           int `json:"gifts"`
	ReceivedGifts   int `json:"receivedGifts"`
	Players         int `json:"players"`
	ReceivedPlayers int `json:"receivedPlayers"`
}

// hasReport 判断服务端是否提供某个报表。
func hasReport(m *mgp.Manifest, id string) bool {
	if m == nil {
		return false
	}
	for _, r := range m.Reports {
		if r.ID == id {
			return true
		}
	}
	return false
}

// fetchMailStats 查询一封邮件的礼物领取统计。
func fetchMailStats(ctx context.Context, client *mgp.Client, mailID string) (*giftStats, error) {
	var st giftStats
	q := url.Values{"sourceKind": {"mail"}, "sourceId": {mailID}}
	if err := client.GetInto(ctx, "/reports/"+giftsReport, q, &st); err != nil {
		return nil, err
	}
	return &st, nil
}

// refreshMailStats 已发送（含已撤回）的邮件领取统计超过 60 秒就重新查询（并发、整体 5 秒超时；失败时保留缓存）。
func (uc *GameOpsUsecase) refreshMailStats(ctx context.Context, serverID string, mails []*gen.GameMail) {
	now := time.Now()
	var stale []*gen.GameMail
	for _, m := range mails {
		if (m.Status == entmail.StatusSent || m.Status == entmail.StatusRevoked) && (m.StatsAt == nil || now.Sub(*m.StatsAt) > mailStatsTTL) {
			stale = append(stale, m)
		}
	}
	if len(stale) == 0 {
		return
	}
	s, client, err := uc.gm.Servers().ClientOf(ctx, serverID)
	if err != nil {
		return
	}
	if mf, err := uc.gm.Servers().Manifest(ctx, s); err != nil || !hasReport(mf, giftsReport) {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for _, m := range stale {
		wg.Add(1)
		sem <- struct{}{}
		go func(m *gen.GameMail) {
			defer wg.Done()
			defer func() { <-sem }()
			st, err := fetchMailStats(ctx, client, m.ID)
			if err != nil {
				return
			}
			at := time.Now()
			if m.RecipientCount == 0 && m.Status == entmail.StatusSent {
				m.RecipientCount = st.Players
			}
			m.ClaimedCount, m.StatsAt = st.ReceivedPlayers, &at
			if err := uc.repo.SaveMailStats(context.WithoutCancel(ctx), m.ID, m.RecipientCount, m.ClaimedCount, at); err != nil {
				uc.log.Warnf("save mail stats %s: %v", m.ID, err)
			}
		}(m)
	}
	wg.Wait()
}

// ListMails 邮件列表（含各分组数量与本月已发数）；当前页已发送的邮件刷新领取统计。
func (uc *GameOpsUsecase) ListMails(ctx context.Context, req *v1.ListGameMailsRequest) (*v1.ListGameMailsResponse, error) {
	s, err := uc.gm.Servers().Get(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	f := GameMailFilter{ServerID: s.ID, Q: strings.TrimSpace(req.Q), Page: int(req.Page), Size: int(req.PageSize)}
	switch req.Status {
	case "draft":
		f.Statuses = []string{"draft"}
	case "scheduled":
		f.Statuses = []string{"scheduled"}
	case "sent":
		f.Statuses = []string{"sending", "sent", "failed", "revoking", "revoked"}
	case "":
	default:
		f.Statuses = []string{req.Status}
	}
	items, total, err := uc.repo.ListMails(ctx, f)
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.refreshMailStats(ctx, s.ID, items)
	counts, err := uc.repo.CountMails(ctx, s.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	now := time.Now().In(uc.gm.Servers().Location(s))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	sent, err := uc.repo.CountMailsSentSince(ctx, s.ID, monthStart)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGameMailsResponse{Total: int32(total), Counts: counts, SentThisMonth: int32(sent)}
	for _, m := range items {
		out.Items = append(out.Items, GameMailProto(m))
	}
	return out, nil
}

// GetMail 邮件详情（已发送的刷新领取统计）。
func (uc *GameOpsUsecase) GetMail(ctx context.Context, serverID, id string) (*v1.GameMail, error) {
	m, err := uc.mailOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	uc.refreshMailStats(ctx, serverID, []*gen.GameMail{m})
	return GameMailProto(m), nil
}

// CreateMail 新建草稿。
func (uc *GameOpsUsecase) CreateMail(ctx context.Context, actor GmActor, req *v1.SaveGameMailRequest) (*v1.GameMail, error) {
	s, err := uc.gm.Servers().Get(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	m := &gen.GameMail{
		ID:            uuid.NewString(),
		ServerID:      s.ID,
		Status:        entmail.StatusDraft,
		CreatedBy:     actor.UserID,
		CreatedByName: actor.Name,
		UpdatedByName: actor.Name,
	}
	if err := uc.applyMailInput(ctx, s, m, req); err != nil {
		return nil, err
	}
	m.ScheduledAt = timeOf(req.ScheduledAt)
	saved, err := uc.repo.CreateMail(ctx, m)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return GameMailProto(saved), nil
}

// UpdateMail 修改草稿或定时邮件。
func (uc *GameOpsUsecase) UpdateMail(ctx context.Context, actor GmActor, req *v1.SaveGameMailRequest) (*v1.GameMail, error) {
	s, err := uc.gm.Servers().Get(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	m, err := uc.mailOf(ctx, s.ID, req.Id)
	if err != nil {
		return nil, err
	}
	if m.Status != entmail.StatusDraft && m.Status != entmail.StatusScheduled {
		return nil, ErrGameMailState
	}
	if err := uc.applyMailInput(ctx, s, m, req); err != nil {
		return nil, err
	}
	if at := timeOf(req.ScheduledAt); at != nil {
		if m.Status == entmail.StatusScheduled && !at.After(time.Now()) {
			return nil, response.BadRequest(400, "定时发送的时间必须晚于现在")
		}
		m.ScheduledAt = at
	} else if m.Status == entmail.StatusDraft {
		m.ScheduledAt = nil
	}
	if m.Status == entmail.StatusScheduled {
		if err := mailReady(m); err != nil {
			return nil, err
		}
	}
	m.UpdatedByName = actor.Name
	saved, err := uc.repo.SaveMail(ctx, m)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return GameMailProto(saved), nil
}

// DeleteMail 删除草稿。
func (uc *GameOpsUsecase) DeleteMail(ctx context.Context, serverID, id string) error {
	m, err := uc.mailOf(ctx, serverID, id)
	if err != nil {
		return err
	}
	if m.Status != entmail.StatusDraft {
		return ErrGameMailState
	}
	if err := uc.repo.DeleteMail(ctx, m.ID); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// DuplicateMail 复制一封邮件为新草稿（内容与收件人相同，统计与原因清空）。
func (uc *GameOpsUsecase) DuplicateMail(ctx context.Context, actor GmActor, serverID, id string) (*v1.GameMail, error) {
	src, err := uc.mailOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	m := &gen.GameMail{
		ID:            uuid.NewString(),
		ServerID:      src.ServerID,
		Title:         src.Title,
		Body:          src.Body,
		Rewards:       src.Rewards,
		Target:        src.Target,
		Players:       src.Players,
		Condition:     src.Condition,
		ExcludeBanned: src.ExcludeBanned,
		ExpireDays:    src.ExpireDays,
		Status:        entmail.StatusDraft,
		CreatedBy:     actor.UserID,
		CreatedByName: actor.Name,
		UpdatedByName: actor.Name,
	}
	saved, err := uc.repo.CreateMail(ctx, m)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return GameMailProto(saved), nil
}

// executeError 把一条失败的 GM 记录转成接口错误。
func executeError(e *gen.GmExecution) error {
	msg := e.ErrorMessage
	if e.ErrorCode != "" {
		msg = e.ErrorCode + ": " + msg
	}
	return response.BadRequest(400, msg)
}

// DryRunMail 试运行 mail.broadcast（记进 GM 记录）：返回按当前收件人匹配的人数并记在邮件上。
func (uc *GameOpsUsecase) DryRunMail(ctx context.Context, actor GmActor, serverID, id string) (*v1.DryRunGameMailResponse, error) {
	m, err := uc.mailOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	if err := mailReady(m); err != nil {
		return nil, err
	}
	e, raw, err := uc.gm.Execute(ctx, actor, serverID, mailCommand, GmCommandInput{
		Params:  mailParams(m),
		DryRun:  true,
		Summary: mailSummary(m),
		Source:  "mail:" + m.ID,
	})
	if err != nil {
		return nil, err
	}
	if e.Status == "failed" {
		return nil, executeError(e)
	}
	sum := parseMailSummary(raw)
	now := time.Now()
	// 只写回试运行结果，不覆盖期间的其他修改。
	latest, err := uc.repo.SaveMailMatched(ctx, m.ID, sum.Matched, now)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.DryRunGameMailResponse{Matched: int32(sum.Matched), Mail: GameMailProto(latest), ExecutionId: e.ID}
	var r mgp.CommandResult
	if json.Unmarshal(raw, &r) == nil && len(r.Summary) > 0 {
		out.Summary = string(r.Summary)
	}
	if s, err := uc.gm.Servers().Get(ctx, serverID); err == nil {
		if st, err := uc.gm.Servers().Status(ctx, s); err == nil {
			out.Registered = int32(st.Registered)
		}
	}
	return out, nil
}

// SendMail 立即发送（后台投递，状态 sending → sent / failed），或按 scheduled_at 定时发送。
func (uc *GameOpsUsecase) SendMail(ctx context.Context, actor GmActor, req *v1.SendGameMailRequest) (*v1.GameMail, error) {
	m, err := uc.mailOf(ctx, req.ServerId, req.Id)
	if err != nil {
		return nil, err
	}
	if m.Status != entmail.StatusDraft && m.Status != entmail.StatusScheduled && m.Status != entmail.StatusFailed {
		return nil, ErrGameMailState
	}
	if err := mailReady(m); err != nil {
		return nil, err
	}
	s, err := uc.gm.Servers().Get(ctx, m.ServerID)
	if err != nil {
		return nil, err
	}
	if mf, err := uc.gm.Servers().Manifest(ctx, s); err != nil {
		return nil, err
	} else if _, ok := mf.Command(mailCommand); !ok {
		return nil, ErrGmCommandNotFound
	}
	if paid, ok := uc.paidJewelType(ctx, s); ok {
		if err := validateRewards(m.Rewards, false, paid); err != nil {
			return nil, err
		}
	}
	m.SentBy, m.SentByName, m.Error = actor.UserID, actor.Name, ""
	if at := timeOf(req.ScheduledAt); at != nil && at.After(time.Now().Add(30*time.Second)) {
		if ok, err := uc.repo.SwitchMailStatus(ctx, m.ID, []string{"draft", "scheduled", "failed"}, "scheduled"); err != nil {
			return nil, ErrSystem(err)
		} else if !ok {
			return nil, ErrGameMailState
		}
		m.Status, m.ScheduledAt = entmail.StatusScheduled, at
		saved, err := uc.repo.SaveMail(ctx, m)
		if err != nil {
			return nil, ErrSystem(err)
		}
		return GameMailProto(saved), nil
	}
	ok, err := uc.repo.SwitchMailStatus(ctx, m.ID, []string{"draft", "scheduled", "failed"}, "sending")
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !ok {
		return nil, ErrGameMailState
	}
	m.Status = entmail.StatusSending
	saved, err := uc.repo.SaveMail(ctx, m)
	if err != nil {
		return nil, ErrSystem(err)
	}
	go uc.deliver(context.WithoutCancel(ctx), actor, saved.ID)
	return GameMailProto(saved), nil
}

// deliver 后台执行 mail.broadcast 并写回结果（同一封邮件只会有一个 deliver：状态已经是 sending）。
func (uc *GameOpsUsecase) deliver(ctx context.Context, actor GmActor, id string) {
	ctx, cancel := context.WithTimeout(ctx, mailDeliverTimeout)
	defer cancel()
	m, err := uc.repo.GetMail(ctx, id)
	if err != nil {
		uc.log.Errorf("deliver mail %s: %v", id, err)
		return
	}
	e, raw, err := uc.gm.Execute(ctx, actor, m.ServerID, mailCommand, GmCommandInput{
		Params:  mailParams(m),
		Summary: mailSummary(m),
		Source:  "mail:" + m.ID,
	})
	now := time.Now()
	switch {
	case err != nil:
		m.Status, m.Error = entmail.StatusFailed, errText(err)
	case e.Status == "failed" && e.ErrorCode == "MAIL_ALREADY_SENT":
		// 之前的投递（例如超时或中断的那次）其实已经送达：以服务端的礼物统计为准。
		m.Status, m.Error, m.ExecutionID = entmail.StatusSent, "", e.ID
		if m.SentAt == nil {
			m.SentAt = &now
		}
		if _, client, err := uc.gm.Servers().ClientOf(ctx, m.ServerID); err == nil {
			if st, err := fetchMailStats(ctx, client, m.ID); err == nil {
				m.RecipientCount, m.ClaimedCount, m.StatsAt = st.Players, st.ReceivedPlayers, &now
			}
		}
	case e.Status == "failed":
		m.Status, m.ExecutionID = entmail.StatusFailed, e.ID
		m.Error = e.ErrorMessage
		if e.ErrorCode != "" {
			m.Error = e.ErrorCode + ": " + e.ErrorMessage
		}
	default:
		sum := parseMailSummary(raw)
		m.Status, m.Error, m.ExecutionID = entmail.StatusSent, "", e.ID
		m.RecipientCount, m.FailedCount, m.ClaimedCount = sum.Delivered, sum.Failed, 0
		m.SentAt, m.StatsAt = &now, &now
	}
	if _, err := uc.repo.SaveMail(context.WithoutCancel(ctx), m); err != nil {
		uc.log.Errorf("save delivered mail %s: %v", m.ID, err)
	}
}

// CancelMail 取消定时，回到草稿。
func (uc *GameOpsUsecase) CancelMail(ctx context.Context, actor GmActor, serverID, id string) (*v1.GameMail, error) {
	m, err := uc.mailOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	ok, err := uc.repo.SwitchMailStatus(ctx, m.ID, []string{"scheduled"}, "draft")
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !ok {
		return nil, ErrGameMailState
	}
	m.Status, m.UpdatedByName, m.SentBy, m.SentByName = entmail.StatusDraft, actor.Name, "", ""
	saved, err := uc.repo.SaveMail(ctx, m)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return GameMailProto(saved), nil
}

// RevokeMail 撤回已发送邮件里尚未领取的礼物（后台执行 mail.revoke，状态 revoking → revoked；失败回到 sent 并记下错误）。
func (uc *GameOpsUsecase) RevokeMail(ctx context.Context, actor GmActor, req *v1.RevokeGameMailRequest) (*v1.GameMail, error) {
	m, err := uc.mailOf(ctx, req.ServerId, req.Id)
	if err != nil {
		return nil, err
	}
	s, err := uc.gm.Servers().Get(ctx, m.ServerID)
	if err != nil {
		return nil, err
	}
	if mf, err := uc.gm.Servers().Manifest(ctx, s); err != nil {
		return nil, err
	} else if _, ok := mf.Command(mailRevokeCommand); !ok {
		return nil, ErrGmCommandNotFound
	}
	ok, err := uc.repo.SwitchMailStatus(ctx, m.ID, []string{"sent"}, "revoking")
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !ok {
		return nil, ErrGameMailState
	}
	m.Status, m.Error, m.RevokedByName = entmail.StatusRevoking, "", actor.Name
	saved, err := uc.repo.SaveMail(ctx, m)
	if err != nil {
		return nil, ErrSystem(err)
	}
	go uc.revoke(context.WithoutCancel(ctx), actor, saved.ID)
	return GameMailProto(saved), nil
}

func (uc *GameOpsUsecase) revoke(ctx context.Context, actor GmActor, id string) {
	ctx, cancel := context.WithTimeout(ctx, mailDeliverTimeout)
	defer cancel()
	m, err := uc.repo.GetMail(ctx, id)
	if err != nil {
		uc.log.Errorf("revoke mail %s: %v", id, err)
		return
	}
	params, _ := json.Marshal(map[string]string{"mailId": m.ID})
	e, raw, err := uc.gm.Execute(ctx, actor, m.ServerID, mailRevokeCommand, GmCommandInput{
		Params:  params,
		Summary: map[string]string{"mailId": m.ID, "title": m.Title},
		Source:  "mail:" + m.ID,
	})
	now := time.Now()
	switch {
	case err != nil:
		m.Status, m.Error, m.RevokedByName = entmail.StatusSent, errText(err), ""
	case e.Status == "failed":
		m.Status, m.RevokeExecutionID, m.RevokedByName = entmail.StatusSent, e.ID, ""
		m.Error = e.ErrorMessage
		if e.ErrorCode != "" {
			m.Error = e.ErrorCode + ": " + e.ErrorMessage
		}
	default:
		sum := parseMailSummary(raw)
		m.Status, m.Error, m.RevokeExecutionID = entmail.StatusRevoked, "", e.ID
		m.RevokedCount, m.RevokedAt, m.StatsAt = sum.Delivered, &now, nil
	}
	if _, err := uc.repo.SaveMail(context.WithoutCancel(ctx), m); err != nil {
		uc.log.Errorf("save revoked mail %s: %v", m.ID, err)
	}
}

// mailLoop 启动时把中断的投递 / 撤回恢复成可重试的状态，之后每 15 秒发送到点的定时邮件。
func (uc *GameOpsUsecase) mailLoop() {
	time.Sleep(5 * time.Second)
	uc.recoverMails()
	ticker := time.NewTicker(mailLoopEvery)
	defer ticker.Stop()
	for {
		uc.sendDue()
		<-ticker.C
	}
}

// recoverMails 进程重启前没有完成的投递标为失败（重新发送时 MGP 会用 MAIL_ALREADY_SENT 防止重复投递），
// 没有完成的撤回回到已发送。
func (uc *GameOpsUsecase) recoverMails() {
	ctx := context.Background()
	list, err := uc.repo.MailsInStatus(ctx, []string{"sending", "revoking"})
	if err != nil {
		uc.log.Warnf("recover mails: %v", err)
		return
	}
	for _, m := range list {
		if m.Status == entmail.StatusSending {
			m.Status, m.Error = entmail.StatusFailed, "INTERRUPTED"
		} else {
			m.Status, m.Error, m.RevokedByName = entmail.StatusSent, "INTERRUPTED", ""
		}
		if _, err := uc.repo.SaveMail(ctx, m); err != nil {
			uc.log.Warnf("recover mail %s: %v", m.ID, err)
		}
	}
}

// sendDue 发送到点的定时邮件（以设置定时的管理员名义执行）。
func (uc *GameOpsUsecase) sendDue() {
	ctx := context.Background()
	due, err := uc.repo.DueMails(ctx, time.Now(), 20)
	if err != nil {
		uc.log.Warnf("due mails: %v", err)
		return
	}
	for _, m := range due {
		ok, err := uc.repo.SwitchMailStatus(ctx, m.ID, []string{"scheduled"}, "sending")
		if err != nil || !ok {
			continue
		}
		actor := GmActor{UserID: m.SentBy, Name: m.SentByName, Kind: "admin", Channel: "admin"}
		go uc.deliver(ctx, actor, m.ID)
	}
}

// ---------------- 兑换码 ----------------

// codeAlphabet 生成兑换码用的字符（去掉容易看错的 0 O 1 I）；32 个字符，随机字节取模无偏。
const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

var (
	universalCodeRe = regexp.MustCompile(`^[A-Z0-9]{4,16}$`)
	codePrefixRe    = regexp.MustCompile(`^[A-Z0-9]{1,12}$`)
)

func randomCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b), nil
}

// uniqueCode 一次性码：PREFIX-XXXX-XXXX（没有前缀时 XXXX-XXXX）。
func uniqueCode(prefix string) (string, error) {
	a, err := randomCode(4)
	if err != nil {
		return "", err
	}
	b, err := randomCode(4)
	if err != nil {
		return "", err
	}
	if prefix == "" {
		return a + "-" + b, nil
	}
	return prefix + "-" + a + "-" + b, nil
}

// NormalizeGiftCode 规范化玩家输入的兑换码：去掉空白、转大写。
func NormalizeGiftCode(code string) string {
	return strings.ToUpper(strings.Join(strings.Fields(code), ""))
}

// GiftBatchStatus 批次状态：disabled / expired / exhausted / not_started / active。
func GiftBatchStatus(b *gen.GiftCodeBatch, now time.Time) string {
	switch {
	case b.DisabledAt != nil:
		return "disabled"
	case b.EndAt != nil && !now.Before(*b.EndAt):
		return "expired"
	case b.TotalLimit > 0 && b.UsedCount >= b.TotalLimit:
		return "exhausted"
	case b.StartAt != nil && now.Before(*b.StartAt):
		return "not_started"
	default:
		return "active"
	}
}

// GiftCodeBatchProto 转为接口结构。
func GiftCodeBatchProto(b *gen.GiftCodeBatch, now time.Time, today int) *v1.GiftCodeBatch {
	return &v1.GiftCodeBatch{
		Id:                   b.ID,
		ServerId:             b.ServerID,
		Name:                 b.Name,
		Kind:                 string(b.Kind),
		Code:                 b.Code,
		Prefix:               b.Prefix,
		Quantity:             int32(b.Quantity),
		Rewards:              rewardsProto(b.Rewards),
		Message:              b.Message,
		TotalLimit:           int32(b.TotalLimit),
		PerPlayerLimit:       int32(b.PerPlayerLimit),
		StartAt:              tsOf(b.StartAt),
		EndAt:                tsOf(b.EndAt),
		Eligibility:          string(b.Eligibility),
		RegisteredWithinDays: int32(b.RegisteredWithinDays),
		Channels:             b.Channels,
		UsedCount:            int32(b.UsedCount),
		Status:               GiftBatchStatus(b, now),
		DisabledAt:           tsOf(b.DisabledAt),
		DisabledByName:       b.DisabledByName,
		CreatedByName:        b.CreatedByName,
		CreateTime:           timestamppb.New(b.CreateTime),
		TodayCount:           int32(today),
	}
}

// GiftCodeRedeemProto 转为接口结构。
func GiftCodeRedeemProto(r *gen.GiftCodeRedeem, batchName string) *v1.GiftCodeRedeem {
	return &v1.GiftCodeRedeem{
		Id:             r.ID,
		BatchId:        r.BatchID,
		BatchName:      batchName,
		Code:           r.Code,
		PlayerId:       r.PlayerID,
		Nickname:       r.Nickname,
		PassportUserId: r.PassportUserID,
		Channel:        string(r.Channel),
		Platform:       r.Platform,
		Result:         string(r.Result),
		ErrorMessage:   r.ErrorMessage,
		ExecutionId:    r.ExecutionID,
		CreateTime:     timestamppb.New(r.CreateTime),
	}
}

func (uc *GameOpsUsecase) batchOf(ctx context.Context, serverID, id string) (*gen.GiftCodeBatch, error) {
	b, err := uc.repo.GetBatch(ctx, id)
	if gen.IsNotFound(err) || (err == nil && b.ServerID != serverID) {
		return nil, ErrGiftBatchNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return b, nil
}

// validChannels 校验兑换入口（至少一个，去重）。
func validChannels(in []string) ([]string, error) {
	var out []string
	for _, c := range in {
		c = strings.TrimSpace(c)
		if c != GiftChannelPassport && c != GiftChannelGame {
			return nil, response.BadRequest(400, "兑换入口无效")
		}
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, response.BadRequest(400, "至少选择一个兑换入口")
	}
	return out, nil
}

// ListBatches 批次列表、各分组数量与统计卡片。
func (uc *GameOpsUsecase) ListBatches(ctx context.Context, req *v1.ListGiftCodeBatchesRequest) (*v1.ListGiftCodeBatchesResponse, error) {
	s, err := uc.gm.Servers().Get(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	items, total, err := uc.repo.ListBatches(ctx, GiftBatchFilter{
		ServerID: s.ID, State: req.Status, Q: strings.TrimSpace(req.Q), Now: now, Page: int(req.Page), Size: int(req.PageSize),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	counts, err := uc.repo.CountBatches(ctx, s.ID, now)
	if err != nil {
		return nil, ErrSystem(err)
	}
	st, err := uc.repo.RedeemStats(ctx, s.ID, uc.gm.Servers().DayStart(s, now))
	if err != nil {
		return nil, ErrSystem(err)
	}
	stats := &v1.GiftCodeStats{
		ActiveBatches:       counts["active"],
		EndedBatches:        counts["ended"],
		TodayRedeems:        int32(st.TodaySuccess),
		TotalRedeems:        int32(st.TotalSuccess),
		FirstRedeemAt:       tsOf(st.FirstSuccess),
		TodayFailureReasons: map[string]int32{},
	}
	topID := ""
	for id, n := range st.TodayByBatch {
		if n > int(stats.TodayTopCount) || (n == int(stats.TodayTopCount) && id < topID) {
			topID, stats.TodayTopCount = id, int32(n)
		}
	}
	if topID != "" {
		if names, err := uc.repo.BatchNames(ctx, []string{topID}); err == nil {
			stats.TodayTopBatch = names[topID]
		}
	}
	for k, n := range st.TodayFailures {
		stats.TodayFailures += int32(n)
		stats.TodayFailureReasons[k] = int32(n)
	}
	out := &v1.ListGiftCodeBatchesResponse{Total: int32(total), Counts: counts, Stats: stats}
	for _, b := range items {
		out.Items = append(out.Items, GiftCodeBatchProto(b, now, st.TodayByBatch[b.ID]))
	}
	return out, nil
}

// GetBatch 批次详情、最近 20 条兑换与成功 / 失败次数。
func (uc *GameOpsUsecase) GetBatch(ctx context.Context, serverID, id string) (*v1.GetGiftCodeBatchResponse, error) {
	b, err := uc.batchOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	s, err := uc.gm.Servers().Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	recent, _, err := uc.repo.ListRedeems(ctx, GiftRedeemFilter{ServerID: serverID, BatchID: b.ID, Page: 1, Size: giftRecentRedeems})
	if err != nil {
		return nil, ErrSystem(err)
	}
	success, failure, err := uc.repo.CountRedeems(ctx, b.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	today := 0
	if st, err := uc.repo.RedeemStats(ctx, serverID, uc.gm.Servers().DayStart(s, now)); err == nil {
		today = st.TodayByBatch[b.ID]
	}
	out := &v1.GetGiftCodeBatchResponse{Batch: GiftCodeBatchProto(b, now, today), SuccessCount: int32(success), FailureCount: int32(failure)}
	for _, r := range recent {
		out.Recent = append(out.Recent, GiftCodeRedeemProto(r, b.Name))
	}
	return out, nil
}

// generateCodes 生成 n 个这台服务器上没有用过的一次性码（加密随机数）。
func (uc *GameOpsUsecase) generateCodes(ctx context.Context, serverID, prefix string, n int) ([]string, error) {
	set := make(map[string]struct{}, n)
	for attempt := 0; attempt < 8 && len(set) < n; attempt++ {
		var batch []string
		for len(set)+len(batch) < n {
			c, err := uniqueCode(prefix)
			if err != nil {
				return nil, err
			}
			if _, ok := set[c]; ok || slices.Contains(batch, c) {
				continue
			}
			batch = append(batch, c)
		}
		taken, err := uc.repo.CodesTaken(ctx, serverID, batch)
		if err != nil {
			return nil, err
		}
		for _, c := range batch {
			if _, ok := taken[c]; !ok {
				set[c] = struct{}{}
			}
		}
	}
	if len(set) < n {
		return nil, response.BadRequest(500, "生成兑换码失败，请重试")
	}
	out := make([]string, 0, n)
	for c := range set {
		out = append(out, c)
	}
	slices.Sort(out)
	return out, nil
}

// CreateBatch 新建批次：通用码（输入或自动生成）或一次性码（服务端用加密随机数生成，每台服务器内唯一）。
func (uc *GameOpsUsecase) CreateBatch(ctx context.Context, actor GmActor, req *v1.CreateGiftCodeBatchRequest) (*v1.GiftCodeBatch, error) {
	s, err := uc.gm.Servers().Get(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || !validText(name, giftNameMax, false) {
		return nil, response.BadRequest(400, "批次名称为 1–64 字")
	}
	message := strings.TrimSpace(req.Message)
	if !validText(message, giftMessageMax, false) {
		return nil, response.BadRequest(400, "礼物箱文案最多 64 字")
	}
	rewards := rewardsOf(req.Rewards)
	if err := validateRewards(rewards, false); err != nil {
		return nil, err
	}
	channels, err := validChannels(req.Channels)
	if err != nil {
		return nil, err
	}
	if req.PerPlayerLimit < 1 || req.PerPlayerLimit > giftPerPlayerMax || req.TotalLimit < 0 || req.TotalLimit > giftLimitMax {
		return nil, response.BadRequest(400, "次数上限超出范围")
	}
	startAt, endAt := timeOf(req.StartAt), timeOf(req.EndAt)
	if startAt != nil && endAt != nil && !endAt.After(*startAt) {
		return nil, response.BadRequest(400, "结束时间必须晚于开始时间")
	}
	if endAt != nil && !endAt.After(time.Now()) {
		return nil, response.BadRequest(400, "结束时间必须晚于现在")
	}
	eligibility := req.Eligibility
	if eligibility == "" {
		eligibility = "all"
	}
	switch eligibility {
	case "all":
		req.RegisteredWithinDays = 0
	case "condition":
		if req.RegisteredWithinDays < 1 || req.RegisteredWithinDays > giftEligibleMax {
			return nil, response.BadRequest(400, "注册天数为 1–3650 天")
		}
	default:
		return nil, ErrGiftBatchInvalid
	}
	b := &gen.GiftCodeBatch{
		ID:                   uuid.NewString(),
		ServerID:             s.ID,
		Name:                 name,
		Rewards:              rewards,
		Message:              message,
		TotalLimit:           int(req.TotalLimit),
		PerPlayerLimit:       int(req.PerPlayerLimit),
		StartAt:              startAt,
		EndAt:                endAt,
		Eligibility:          entbatch.Eligibility(eligibility),
		RegisteredWithinDays: int(req.RegisteredWithinDays),
		Channels:             channels,
		CreatedBy:            actor.UserID,
		CreatedByName:        actor.Name,
	}
	// 同一台服务器的批次创建串行执行，保证码在服务器内唯一（查重与写入之间不被插队）。
	unlock := uc.lock("create:" + s.ID)
	defer unlock()
	var codes []string
	switch req.Kind {
	case "universal":
		code := NormalizeGiftCode(req.Code)
		if code == "" {
			for i := 0; i < 8 && code == ""; i++ {
				c, err := randomCode(8)
				if err != nil {
					return nil, ErrSystem(err)
				}
				taken, err := uc.repo.CodesTaken(ctx, s.ID, []string{c})
				if err != nil {
					return nil, ErrSystem(err)
				}
				if _, ok := taken[c]; !ok {
					code = c
				}
			}
			if code == "" {
				return nil, response.BadRequest(500, "生成兑换码失败，请重试")
			}
		} else {
			if !universalCodeRe.MatchString(code) {
				return nil, response.BadRequest(400, "兑换码为 4–16 位字母和数字")
			}
			taken, err := uc.repo.CodesTaken(ctx, s.ID, []string{code})
			if err != nil {
				return nil, ErrSystem(err)
			}
			if _, ok := taken[code]; ok {
				return nil, ErrGiftCodeTaken
			}
		}
		b.Kind, b.Code = entbatch.KindUniversal, code
	case "unique":
		prefix := NormalizeGiftCode(req.Prefix)
		if prefix != "" && !codePrefixRe.MatchString(prefix) {
			return nil, response.BadRequest(400, "前缀为 1–12 位字母和数字")
		}
		if req.Quantity < 1 || req.Quantity > giftQuantityMax {
			return nil, response.BadRequest(400, "生成数量为 1–10000")
		}
		codes, err = uc.generateCodes(ctx, s.ID, prefix, int(req.Quantity))
		if err != nil {
			return nil, err
		}
		b.Kind, b.Prefix, b.Quantity, b.TotalLimit = entbatch.KindUnique, prefix, int(req.Quantity), int(req.Quantity)
	default:
		return nil, ErrGiftBatchInvalid
	}
	saved, err := uc.repo.CreateBatch(ctx, b, codes)
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, ErrGiftCodeTaken
		}
		return nil, ErrSystem(err)
	}
	return GiftCodeBatchProto(saved, time.Now(), 0), nil
}

// UpdateBatch 编辑批次：名称、结束时间、次数、兑换入口与礼物箱文案（码、类型与奖励不能改）。
func (uc *GameOpsUsecase) UpdateBatch(ctx context.Context, req *v1.UpdateGiftCodeBatchRequest) (*v1.GiftCodeBatch, error) {
	unlock := uc.lock("batch:" + req.Id)
	defer unlock()
	b, err := uc.batchOf(ctx, req.ServerId, req.Id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" || !validText(name, giftNameMax, false) {
		return nil, response.BadRequest(400, "批次名称为 1–64 字")
	}
	message := strings.TrimSpace(req.Message)
	if !validText(message, giftMessageMax, false) {
		return nil, response.BadRequest(400, "礼物箱文案最多 64 字")
	}
	channels, err := validChannels(req.Channels)
	if err != nil {
		return nil, err
	}
	if req.PerPlayerLimit < 1 || req.PerPlayerLimit > giftPerPlayerMax {
		return nil, response.BadRequest(400, "次数上限超出范围")
	}
	endAt := timeOf(req.EndAt)
	if endAt != nil && b.StartAt != nil && !endAt.After(*b.StartAt) {
		return nil, response.BadRequest(400, "结束时间必须晚于开始时间")
	}
	if b.Kind == entbatch.KindUniversal {
		if req.TotalLimit < 0 || req.TotalLimit > giftLimitMax || (req.TotalLimit > 0 && int(req.TotalLimit) < b.UsedCount) {
			return nil, response.BadRequest(400, "总次数上限不能小于已用次数")
		}
		b.TotalLimit = int(req.TotalLimit)
	}
	b.Name, b.Message, b.Channels, b.PerPlayerLimit, b.EndAt = name, message, channels, int(req.PerPlayerLimit), endAt
	saved, err := uc.repo.SaveBatch(ctx, b)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return GiftCodeBatchProto(saved, time.Now(), 0), nil
}

// DisableBatch 停用批次（之后的兑换返回 disabled）。
func (uc *GameOpsUsecase) DisableBatch(ctx context.Context, actor GmActor, serverID, id string) (*v1.GiftCodeBatch, error) {
	unlock := uc.lock("batch:" + id)
	defer unlock()
	b, err := uc.batchOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	if b.DisabledAt == nil {
		now := time.Now()
		b.DisabledAt, b.DisabledByName = &now, actor.Name
		if b, err = uc.repo.SaveBatch(ctx, b); err != nil {
			return nil, ErrSystem(err)
		}
	}
	return GiftCodeBatchProto(b, time.Now(), 0), nil
}

// ListCodes 一次性码列表（导出用）。
func (uc *GameOpsUsecase) ListCodes(ctx context.Context, req *v1.ListGiftCodesRequest) (*v1.ListGiftCodesResponse, error) {
	b, err := uc.batchOf(ctx, req.ServerId, req.BatchId)
	if err != nil {
		return nil, err
	}
	items, total, err := uc.repo.ListCodes(ctx, b.ID, req.State, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGiftCodesResponse{Total: int32(total)}
	for _, c := range items {
		out.Items = append(out.Items, &v1.GiftCodeItem{Code: c.Code, UsedByPlayer: c.UsedByPlayer, UsedByName: c.UsedByName, UsedAt: tsOf(c.UsedAt)})
	}
	return out, nil
}

// ListRedeems 兑换记录（可按批次与结果筛选，导出用）。
func (uc *GameOpsUsecase) ListRedeems(ctx context.Context, req *v1.ListGiftCodeRedeemsRequest) (*v1.ListGiftCodeRedeemsResponse, error) {
	if _, err := uc.gm.Servers().Get(ctx, req.ServerId); err != nil {
		return nil, err
	}
	if req.BatchId != "" {
		if _, err := uc.batchOf(ctx, req.ServerId, req.BatchId); err != nil {
			return nil, err
		}
	}
	items, total, err := uc.repo.ListRedeems(ctx, GiftRedeemFilter{
		ServerID: req.ServerId, BatchID: req.BatchId, Result: req.Result, Page: int(req.Page), Size: int(req.PageSize),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(items))
	for _, r := range items {
		if r.BatchID != "" && !slices.Contains(ids, r.BatchID) {
			ids = append(ids, r.BatchID)
		}
	}
	names, err := uc.repo.BatchNames(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGiftCodeRedeemsResponse{Total: int32(total)}
	for _, r := range items {
		out.Items = append(out.Items, GiftCodeRedeemProto(r, names[r.BatchID]))
	}
	return out, nil
}

// GiftRedeemInput 玩家兑换一个兑换码（通行证「自助服务」调用；目标是该通行证在这台服务器绑定的游戏账号）。
type GiftRedeemInput struct {
	ServerID       string
	PassportUserID string
	PassportName   string
	PID            string
	Nickname       string
	Code           string
	// Channel passport / game
	Channel  string
	Platform string
	IP       string
}

// GiftRedeemOutcome 兑换结果。Result 为 gift_code_redeem.result 的取值（success / already_redeemed / expired …）。
// 兑换没有成功（码不存在、过期、次数用完、发放失败……）时返回 Result 与 nil 错误；只有参数或系统错误才返回 error。
type GiftRedeemOutcome struct {
	Result      string
	Rewards     []entschema.GameReward
	ExecutionID string
	// BatchName 批次名称（码不存在时为空；只给管理端看，玩家端不要展示）
	BatchName string
	// ErrorMessage 发放失败时的错误详情（Result = failed）
	ErrorMessage string
}

// RedeemGiftCode 校验兑换码并发放奖励（每次都写一条 gift_code_redeem）。
// 同一通行证用户、同一批次的兑换串行执行；每个通行证用户每分钟最多尝试 10 次。
// 校验顺序：频率 → 码 → 停用 / 未开始 / 过期 / 用完 → 兑换入口 → 一次性码是否已用 → 每人次数 → 封禁 → 适用玩家；
// 通过后先占用次数（与一次性码），再以玩家身份执行 gift.send（sourceKind=code、sourceId=批次 id），失败时释放占用。
func (uc *GameOpsUsecase) RedeemGiftCode(ctx context.Context, in GiftRedeemInput) (*GiftRedeemOutcome, error) {
	ctx = context.WithoutCancel(ctx)
	if strings.TrimSpace(in.PID) == "" || in.PassportUserID == "" {
		return nil, ErrGiftRedeemNoAccount
	}
	code := NormalizeGiftCode(in.Code)
	if code == "" {
		return nil, ErrGiftRedeemEmpty
	}
	if len(code) > 64 {
		code = code[:64]
	}
	channel := in.Channel
	if channel == "" {
		channel = GiftChannelPassport
	}
	if channel != GiftChannelPassport && channel != GiftChannelGame && channel != GiftChannelAdmin {
		return nil, response.BadRequest(400, "兑换入口无效")
	}
	s, client, err := uc.gm.Servers().ClientOf(ctx, in.ServerID)
	if err != nil {
		return nil, err
	}
	rec := &gen.GiftCodeRedeem{
		ServerID:       s.ID,
		Code:           code,
		PlayerID:       in.PID,
		Nickname:       in.Nickname,
		PassportUserID: in.PassportUserID,
		Channel:        entredeem.Channel(channel),
		Platform:       in.Platform,
	}
	out := &GiftRedeemOutcome{}
	finish := func(result, message string) (*GiftRedeemOutcome, error) {
		rec.Result, rec.ErrorMessage = entredeem.Result(result), message
		if _, err := uc.repo.CreateRedeem(ctx, rec); err != nil {
			return nil, ErrSystem(err)
		}
		out.Result, out.ErrorMessage, out.ExecutionID = result, message, rec.ExecutionID
		return out, nil
	}

	unlockUser := uc.lock("user:" + in.PassportUserID)
	defer unlockUser()
	now := time.Now()
	attempts, err := uc.repo.CountAttempts(ctx, in.PassportUserID, now.Add(-redeemRateWindow))
	if err != nil {
		return nil, ErrSystem(err)
	}
	if attempts >= redeemRateLimit {
		return finish(RedeemRateLimited, "")
	}

	found, gc, err := uc.repo.FindCode(ctx, s.ID, code)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if found == nil {
		return finish(RedeemNotFound, "")
	}
	rec.BatchID = found.ID
	unlockBatch := uc.lock("batch:" + found.ID)
	defer unlockBatch()
	// 拿到锁之后重新读取，保证次数与状态是最新的。
	b, err := uc.repo.GetBatch(ctx, found.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out.BatchName = b.Name
	now = time.Now()
	switch GiftBatchStatus(b, now) {
	case "disabled":
		return finish(RedeemDisabled, "")
	case "not_started":
		return finish(RedeemNotStarted, "")
	case "expired":
		return finish(RedeemExpired, "")
	case "exhausted":
		return finish(RedeemExhausted, "")
	}
	if channel != GiftChannelAdmin && !slices.Contains(b.Channels, channel) {
		return finish(RedeemChannelClosed, "")
	}
	if gc != nil {
		if gc, err = uc.repo.GetCode(ctx, gc.ID); err != nil {
			return nil, ErrSystem(err)
		}
		if gc.UsedByPlayer != "" {
			if gc.UsedByPlayer == in.PID {
				return finish(RedeemAlreadyRedeemed, "")
			}
			return finish(RedeemCodeUsed, "")
		}
	}
	used, err := uc.repo.CountPlayerRedeems(ctx, s.ID, b.ID, in.PID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if used >= max(b.PerPlayerLimit, 1) {
		return finish(RedeemAlreadyRedeemed, "")
	}
	var player struct {
		mgp.Player
		CreatedAt int64 `json:"createdAt"`
	}
	if err := client.GetInto(ctx, "/players/"+url.PathEscape(in.PID), nil, &player); err != nil {
		return finish(RedeemFailed, errText(GameMGPError(err)))
	}
	if player.Name != "" {
		rec.Nickname = player.Name
	}
	if player.BannedUntil > now.UnixMilli() {
		return finish(RedeemBanned, "")
	}
	if b.Eligibility == entbatch.EligibilityCondition && b.RegisteredWithinDays > 0 {
		since := now.AddDate(0, 0, -b.RegisteredWithinDays).UnixMilli()
		if player.CreatedAt <= 0 || player.CreatedAt < since {
			return finish(RedeemNotEligible, "")
		}
	}

	// 占用次数与一次性码，再发放；发放失败时释放。
	ok, err := uc.repo.ReserveUse(ctx, b.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !ok {
		return finish(RedeemExhausted, "")
	}
	release := func() {
		if err := uc.repo.ReleaseUse(ctx, b.ID); err != nil {
			uc.log.Errorf("release gift batch %s: %v", b.ID, err)
		}
		if gc != nil {
			if err := uc.repo.ReleaseCode(ctx, gc.ID); err != nil {
				uc.log.Errorf("release gift code %s: %v", gc.ID, err)
			}
		}
	}
	if gc != nil {
		ok, err := uc.repo.ClaimCode(ctx, gc.ID, in.PID, rec.Nickname, now)
		if err != nil || !ok {
			if err := uc.repo.ReleaseUse(ctx, b.ID); err != nil {
				uc.log.Errorf("release gift batch %s: %v", b.ID, err)
			}
			if err != nil {
				return nil, ErrSystem(err)
			}
			return finish(RedeemCodeUsed, "")
		}
	}
	params, _ := json.Marshal(map[string]any{
		"rewards":    b.Rewards,
		"message":    b.Message,
		"sourceKind": "code",
		"sourceId":   b.ID,
	})
	actorName := in.PassportName
	if actorName == "" {
		actorName = rec.Nickname
	}
	actor := GmActor{UserID: in.PassportUserID, Name: actorName, Kind: "player", Channel: channel, IP: in.IP, Platform: in.Platform}
	e, _, err := uc.gm.Execute(ctx, actor, s.ID, giftCommand, GmCommandInput{
		Targets: []string{in.PID},
		Params:  params,
		Summary: map[string]string{"batchId": b.ID, "batch": b.Name, "code": code, "rewards": itoa(len(b.Rewards))},
		Source:  "code:" + b.ID,
	})
	if err != nil {
		release()
		return finish(RedeemFailed, errText(err))
	}
	rec.ExecutionID = e.ID
	if e.Status != "success" {
		release()
		msg := e.ErrorMessage
		if e.ErrorCode != "" {
			msg = e.ErrorCode + ": " + msg
		}
		return finish(RedeemFailed, msg)
	}
	out.Rewards = b.Rewards
	return finish(RedeemSuccess, "")
}
