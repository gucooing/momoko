package biz

import (
	"context"
	"crypto/rand"
	"encoding/json"
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
	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/data/ent/gen"
	entbatch "momoko/internal/data/ent/gen/giftcodebatch"
	entschema "momoko/internal/data/ent/schema"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
	"momoko/pkg/response"
)

var (
	ErrGameMailNotFound   = response.BadRequest("邮件不存在")
	ErrGameMailState      = response.BadRequest("邮件当前的状态不能这样操作，请刷新后再试")
	ErrGameMailID         = response.BadRequest("邮件 id 无效")
	ErrGameMailIncomplete = response.BadRequest("邮件还没写完：需要标题、至少一项奖励和收件人")
	ErrGiftBatchNotFound  = response.BadRequest("兑换码批次不存在")
	ErrGiftCodeTaken      = response.BadRequest("这台服务器上已经有这个兑换码")
	ErrGiftBatchInvalid   = response.BadRequest("兑换码批次参数无效")
)

const (
	// mailRevokeTimeout 一次后台撤回最长的执行时间。
	mailRevokeTimeout = 10 * time.Minute
	// mailScheduleLead 定时投递的时间至少要晚于现在这么久，否则按立即投递。
	mailScheduleLead = 30 * time.Second

	rewardsMax = 20

	giftNameMax       = 64
	giftMessageMax    = 64
	giftQuantityMax   = 10000
	giftLimitMax      = 1_000_000
	giftPerPlayerMax  = 1000
	giftEligibleMax   = 3650
	giftRecentRedeems = 20
)

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

// GameOpsRepo 兑换码的持久化（gift_code_batch、gift_code、gift_code_redeem）。
type GameOpsRepo interface {
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

// GameOpsUsecase 运营（管理端）：邮件补偿与兑换码批次的管理与兑换记录。
// 邮件（含草稿）全部保存在游戏服务端，Momoko 不保存：列表、详情、保存与删除草稿直接调 MGP 邮件接口；
// 发送（mail.send）、取消定时（mail.cancel）与撤回（mail.revoke）经 GmUsecase.Execute 执行并写进 GM 记录
// （source = mail:<id>）。玩家兑换在通行证一侧（PassportUsecase.Redeem）。
type GameOpsUsecase struct {
	gm   *GmUsecase
	repo GameOpsRepo
	log  *log.Helper

	// locks 串行化同一台服务器的批次创建与同一批次的修改。
	locks sync.Map
}

func NewGameOpsUsecase(gm *GmUsecase, repo GameOpsRepo, logger log.Logger) *GameOpsUsecase {
	return &GameOpsUsecase{
		gm:   gm,
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "biz/game_ops")),
	}
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
		return response.BadRequest("至少需要一项奖励")
	}
	if len(rewards) > rewardsMax {
		return response.BadRequest("奖励最多 20 项")
	}
	for _, r := range rewards {
		if r.Quantity <= 0 {
			return response.BadRequest("奖励数量必须大于 0")
		}
		if slices.Contains(forbidden, r.ResourceType) {
			return response.BadRequest("付费钻石不能通过礼物发放")
		}
	}
	return nil
}

// ---------------- 邮件补偿 ----------------

// mailClient 邮件所在的学マス 服务器与它的 MGP 客户端。
func (uc *GameOpsUsecase) mailClient(ctx context.Context, serverID string) (*gen.GameServer, *gakumas.Client, error) {
	s, client, err := uc.gm.Servers().ClientOf(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	if s.Game != gakumas.ID {
		return nil, nil, ErrGmGameUnsupported
	}
	return s, gakumas.New(client), nil
}

// mailError 把服务端的邮件错误转为接口错误。
func mailError(err error) error {
	return mailCodeError(mgp.CodeOf(err), GameMGPError(err))
}

// msOf Unix 毫秒转为时间戳（0 为空）。
func msOf(ms int64) *timestamppb.Timestamp {
	if ms <= 0 {
		return nil
	}
	return timestamppb.New(time.UnixMilli(ms))
}

// GameMailProto 服务端的邮件转为接口结构。
func GameMailProto(m *gakumas.MailView) *v1.GameMail {
	out := &v1.GameMail{
		Id:            m.MailID,
		Status:        m.Status,
		Target:        m.Target,
		Players:       make([]*v1.GameTarget, 0, len(m.Players)),
		ExcludeBanned: m.ExcludeBanned,
		Message:       m.Message,
		Note:          m.Note,
		Rewards:       make([]*v1.GameReward, 0, len(m.Rewards)),
		ExpireDays:    m.ExpireDays,
		StartTime:     msOf(m.StartTime),
		LimitTime:     msOf(m.LimitTime),
		DeliveredAt:   msOf(m.DeliveredAt),
		Matched:       m.Matched,
		Done:          m.Done,
		Failed:        m.Failed,
		RevokedAt:     msOf(m.RevokedAt),
		RevokeDoneAt:  msOf(m.RevokeDoneAt),
		Revoked:       m.Revoked,
		Recipients:    m.Recipients,
		Received:      m.Received,
		CreatedBy:     m.CreatedBy,
		UpdatedBy:     m.UpdatedBy,
		SentBy:        m.SentBy,
		RevokedBy:     m.RevokedBy,
		CreateTime:    msOf(m.CreatedAt),
		UpdateTime:    msOf(m.UpdatedAt),
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
	for _, r := range m.Rewards {
		out.Rewards = append(out.Rewards, &v1.GameReward{ResourceType: r.ResourceType, ResourceId: r.ResourceID, Quantity: r.Quantity})
	}
	return out
}

// mailOptions 不写 GM 记录的邮件操作（保存、删除草稿）的执行选项。
func mailOptions(actor GmActor) mgp.OperationOptions {
	return mgp.OperationOptions{Operator: actor.Name, IdempotencyKey: "mail-" + uuid.NewString()}
}

// mailOperation 以执行者对邮件所在的学マス 服务器执行邮件操作（写 GM 记录，source = mail:<id>）。
func (uc *GameOpsUsecase) mailOperation(ctx context.Context, actor GmActor, serverID, opID, mailID string, params any, dryRun bool, summary map[string]string) (*gen.GmExecution, *gk.MailResult, error) {
	s, client, err := uc.mailClient(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	op, _ := gakumasOperation(opID)
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, nil, ErrGmParams
	}
	call, err := gakumasCall(client, opID, raw)
	if err != nil {
		return nil, nil, err
	}
	e, outcomes, err := uc.gm.Execute(ctx, actor, s, op, GmExecutionInput{
		Params:  raw,
		DryRun:  dryRun,
		Summary: summary,
		Source:  "mail:" + mailID,
	}, nil, call)
	if err != nil {
		return nil, nil, err
	}
	if e.Status == "failed" {
		return e, nil, mailExecutionError(e)
	}
	for _, o := range outcomes {
		if r, ok := o.Result.(*gk.MailResult); ok {
			return e, r, nil
		}
	}
	return e, &gk.MailResult{}, nil
}

// mailExecutionError 把一条失败的邮件操作 GM 记录转成接口错误。
func mailExecutionError(e *gen.GmExecution) error {
	msg := e.ErrorMessage
	if e.ErrorCode != "" {
		msg = e.ErrorCode + ": " + msg
	}
	return mailCodeError(e.ErrorCode, response.BadRequest(msg))
}

// mailCodeError 服务端的邮件错误码对应的接口错误；其他错误码返回 other。
func mailCodeError(code string, other error) error {
	switch code {
	case gakumas.MailNotFound:
		return ErrGameMailNotFound
	case gakumas.MailAlreadyStarted, gakumas.MailAlreadySent, gakumas.MailNotDraft, gakumas.MailNotScheduled:
		return ErrGameMailState
	case gakumas.MailIncomplete:
		return ErrGameMailIncomplete
	}
	return other
}

// mailSummary GM 记录里可展示的摘要。
func mailSummary(m *gakumas.MailView) map[string]string {
	out := map[string]string{
		"mailId":  m.MailID,
		"title":   m.Message,
		"target":  m.Target,
		"rewards": itoa(len(m.Rewards)),
	}
	if m.Target == "players" {
		out["players"] = itoa(len(m.Players))
	}
	return out
}

// ListMails 邮件列表：筛选、搜索、分页与各组数量都由服务端查询；本月按服务器时区算。
func (uc *GameOpsUsecase) ListMails(ctx context.Context, req *v1.ListGameMailsRequest) (*v1.ListGameMailsResponse, error) {
	s, g, err := uc.mailClient(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	size := req.PageSize
	if size <= 0 || size > 100 {
		size = 20
	}
	page := max(req.Page, 1)
	now := time.Now().In(uc.gm.Servers().Location(s))
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	list, err := g.ListMails(ctx, gakumas.MailFilter{
		Status:    req.Status,
		Q:         strings.TrimSpace(req.Q),
		Cursor:    itoa(int((page - 1) * size)),
		Limit:     size,
		SentSince: monthStart.UnixMilli(),
	})
	if err != nil {
		return nil, mailError(err)
	}
	out := &v1.ListGameMailsResponse{
		Items:         make([]*v1.GameMail, 0, len(list.Items)),
		Total:         list.Total,
		Counts:        list.Counts,
		SentThisMonth: list.SentSince,
	}
	for i := range list.Items {
		out.Items = append(out.Items, GameMailProto(&list.Items[i]))
	}
	return out, nil
}

// GetMail 邮件详情。
func (uc *GameOpsUsecase) GetMail(ctx context.Context, serverID, id string) (*v1.GameMail, error) {
	_, g, err := uc.mailClient(ctx, serverID)
	if err != nil {
		return nil, err
	}
	m, err := g.GetMail(ctx, id)
	if err != nil {
		return nil, mailError(err)
	}
	return GameMailProto(m), nil
}

// SaveMail 保存草稿（没有这封邮件时新建），或修改还没开始投递的定时邮件；内容由服务端校验。
func (uc *GameOpsUsecase) SaveMail(ctx context.Context, actor GmActor, req *v1.SaveGameMailRequest) (*v1.GameMail, error) {
	if _, err := uuid.Parse(req.Id); err != nil {
		return nil, ErrGameMailID
	}
	_, g, err := uc.mailClient(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	content := gakumas.MailContent{
		Target:        req.Target,
		Players:       req.Players,
		ExcludeBanned: req.ExcludeBanned,
		Rewards:       make([]gakumas.Reward, 0, len(req.Rewards)),
		Message:       strings.TrimSpace(req.Message),
		Note:          strings.TrimSpace(req.Note),
		ExpireDays:    req.ExpireDays,
	}
	if c := req.Condition; c != nil && req.Target == "condition" {
		content.Condition = &gakumas.MailCondition{
			CreatedSince:    c.CreatedSince,
			CreatedBefore:   c.CreatedBefore,
			LastLoginSince:  c.LastLoginSince,
			LastLoginBefore: c.LastLoginBefore,
		}
	}
	for _, r := range req.Rewards {
		if r != nil {
			content.Rewards = append(content.Rewards, gakumas.Reward{ResourceType: r.ResourceType, ResourceID: strings.TrimSpace(r.ResourceId), Quantity: r.Quantity})
		}
	}
	if at := timeOf(req.StartTime); at != nil {
		content.StartTime = at.UnixMilli()
	}
	m, err := g.SaveMail(ctx, req.Id, content, mailOptions(actor))
	if err != nil {
		return nil, mailError(err)
	}
	return GameMailProto(m), nil
}

// DeleteMail 删除草稿。
func (uc *GameOpsUsecase) DeleteMail(ctx context.Context, actor GmActor, serverID, id string) error {
	_, g, err := uc.mailClient(ctx, serverID)
	if err != nil {
		return err
	}
	if _, err := g.DeleteMail(ctx, id, mailOptions(actor)); err != nil {
		return mailError(err)
	}
	return nil
}

// DryRunMail 试运行发送（写进 GM 记录）：返回现在匹配的人数与服务端注册总数。
func (uc *GameOpsUsecase) DryRunMail(ctx context.Context, actor GmActor, serverID, id string) (*v1.DryRunGameMailResponse, error) {
	s, g, err := uc.mailClient(ctx, serverID)
	if err != nil {
		return nil, err
	}
	m, err := g.GetMail(ctx, id)
	if err != nil {
		return nil, mailError(err)
	}
	e, r, err := uc.mailOperation(ctx, actor, serverID, gkMailSend, id, gkMailSendParams{MailID: id}, true, mailSummary(m))
	if err != nil {
		return nil, err
	}
	out := &v1.DryRunGameMailResponse{Matched: r.Matched, ExecutionId: e.ID}
	if st, err := uc.gm.Servers().Status(ctx, s); err == nil {
		out.Registered = int32(st.Registered)
	}
	return out, nil
}

// SendMail 发送草稿：start_time 晚于现在至少 30 秒时从那时起定时投递，否则立即投递。
func (uc *GameOpsUsecase) SendMail(ctx context.Context, actor GmActor, req *v1.SendGameMailRequest) (*v1.GameMail, error) {
	_, g, err := uc.mailClient(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	m, err := g.GetMail(ctx, req.Id)
	if err != nil {
		return nil, mailError(err)
	}
	var start int64
	if at := timeOf(req.StartTime); at != nil && at.After(time.Now().Add(mailScheduleLead)) {
		start = at.UnixMilli()
	}
	if _, _, err := uc.mailOperation(ctx, actor, req.ServerId, gkMailSend, req.Id, gkMailSendParams{MailID: req.Id, StartTime: start}, false, mailSummary(m)); err != nil {
		return nil, err
	}
	return uc.GetMail(ctx, req.ServerId, req.Id)
}

// CancelMail 取消定时：还没开始投递的定时邮件回到草稿。
func (uc *GameOpsUsecase) CancelMail(ctx context.Context, actor GmActor, serverID, id string) (*v1.GameMail, error) {
	_, g, err := uc.mailClient(ctx, serverID)
	if err != nil {
		return nil, err
	}
	m, err := g.GetMail(ctx, id)
	if err != nil {
		return nil, mailError(err)
	}
	if _, _, err := uc.mailOperation(ctx, actor, serverID, gkMailCancel, id, gkMailID{MailID: id}, false, mailSummary(m)); err != nil {
		return nil, err
	}
	return uc.GetMail(ctx, serverID, id)
}

// RevokeMail 撤回已投递（或投递中）的邮件：服务端先标记撤回、不再投递，再撤回所有尚未领取的礼物。
// 撤回可能要处理很多礼物箱，在后台执行；返回的邮件为撤回中，之后以服务端的状态为准。
func (uc *GameOpsUsecase) RevokeMail(ctx context.Context, actor GmActor, req *v1.RevokeGameMailRequest) (*v1.GameMail, error) {
	_, g, err := uc.mailClient(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	m, err := g.GetMail(ctx, req.Id)
	if err != nil {
		return nil, mailError(err)
	}
	if m.Status != gakumas.MailSent && m.Status != gakumas.MailSending {
		return nil, ErrGameMailState
	}
	go func(ctx context.Context) {
		ctx, cancel := context.WithTimeout(ctx, mailRevokeTimeout)
		defer cancel()
		if _, _, err := uc.mailOperation(ctx, actor, req.ServerId, gkMailRevoke, req.Id, gkMailID{MailID: req.Id}, false, mailSummary(m)); err != nil {
			uc.log.Errorf("revoke mail %s: %v", req.Id, err)
		}
	}(context.WithoutCancel(ctx))
	out := GameMailProto(m)
	out.Status = gakumas.MailRevoking
	return out, nil
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
			return nil, response.BadRequest("兑换入口无效")
		}
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return nil, response.BadRequest("至少选择一个兑换入口")
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
		return nil, response.BadRequest("生成兑换码失败，请重试")
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
		return nil, response.BadRequest("批次名称为 1–64 字")
	}
	message := strings.TrimSpace(req.Message)
	if !validText(message, giftMessageMax, false) {
		return nil, response.BadRequest("礼物箱文案最多 64 字")
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
		return nil, response.BadRequest("次数上限超出范围")
	}
	startAt, endAt := timeOf(req.StartAt), timeOf(req.EndAt)
	if startAt != nil && endAt != nil && !endAt.After(*startAt) {
		return nil, response.BadRequest("结束时间必须晚于开始时间")
	}
	if endAt != nil && !endAt.After(time.Now()) {
		return nil, response.BadRequest("结束时间必须晚于现在")
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
			return nil, response.BadRequest("注册天数为 1–3650 天")
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
				return nil, response.BadRequest("生成兑换码失败，请重试")
			}
		} else {
			if !universalCodeRe.MatchString(code) {
				return nil, response.BadRequest("兑换码为 4–16 位字母和数字")
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
			return nil, response.BadRequest("前缀为 1–12 位字母和数字")
		}
		if req.Quantity < 1 || req.Quantity > giftQuantityMax {
			return nil, response.BadRequest("生成数量为 1–10000")
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
		return nil, response.BadRequest("批次名称为 1–64 字")
	}
	message := strings.TrimSpace(req.Message)
	if !validText(message, giftMessageMax, false) {
		return nil, response.BadRequest("礼物箱文案最多 64 字")
	}
	channels, err := validChannels(req.Channels)
	if err != nil {
		return nil, err
	}
	if req.PerPlayerLimit < 1 || req.PerPlayerLimit > giftPerPlayerMax {
		return nil, response.BadRequest("次数上限超出范围")
	}
	endAt := timeOf(req.EndAt)
	if endAt != nil && b.StartAt != nil && !endAt.After(*b.StartAt) {
		return nil, response.BadRequest("结束时间必须晚于开始时间")
	}
	if b.Kind == entbatch.KindUniversal {
		if req.TotalLimit < 0 || req.TotalLimit > giftLimitMax || (req.TotalLimit > 0 && int(req.TotalLimit) < b.UsedCount) {
			return nil, response.BadRequest("总次数上限不能小于已用次数")
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
