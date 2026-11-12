package biz

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp"
	"momoko/pkg/passporttoken"
)

// gameLink 当前用户在这台服务器上的绑定（游戏空间的一切数据都只针对它）。
func (uc *PassportUsecase) gameLink(ctx context.Context, userID, serverID string) (*gen.GameAccountLink, error) {
	l, err := uc.repo.LinkOf(ctx, userID, serverID)
	if gen.IsNotFound(err) || (err == nil && (l == nil || l.PlayerID == "")) {
		return nil, ErrPassportLinkNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return l, nil
}

// playerManifest manifest 中玩家端用到的部分（不含指令）。
func playerManifest(m *mgp.Manifest) json.RawMessage {
	return jsonRaw(map[string]any{
		"protocol":      m.Protocol,
		"game":          m.Game,
		"capabilities":  m.Capabilities,
		"views":         m.Views,
		"modes":         m.Modes,
		"reports":       m.Reports,
		"catalogs":      m.Catalogs,
		"resourceTypes": m.ResourceType,
	})
}

// Game 游戏空间上下文；touch 时记录 last_seen_at（进入游戏空间），否则只是首页卡片的摘要。
func (uc *PassportUsecase) Game(ctx context.Context, userID, serverID string, touch bool) (*v1.PassportGameResponse, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	s, err := uc.Servers().Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	out := &v1.PassportGameResponse{Server: uc.Servers().Brief(ctx, s), Timezone: uc.Servers().Location(s).String()}
	if touch {
		now := time.Now()
		l.LastSeenAt = &now
		if err := uc.repo.TouchLink(ctx, l.ID, now); err != nil {
			uc.log.Warnf("touch link: %v", err)
		}
		if m, err := uc.Servers().Manifest(ctx, s); err == nil {
			out.Manifest = JSONValue(playerManifest(m))
		}
	}
	client, err := uc.Servers().Client(s)
	if err != nil {
		out.PlayerError = "PASSPORT_GAME_UNAVAILABLE"
	} else if raw, err := client.Get(ctx, playerPath(l.PlayerID), nil); err != nil {
		out.PlayerError = passportFirst(mgp.CodeOf(err), "PASSPORT_GAME_UNAVAILABLE")
	} else {
		out.Player = JSONValue(raw)
		uc.refreshSnapshot(ctx, l, raw)
	}
	ss := uc.selfServiceCounts(ctx)
	servers := map[string]*gen.GameServer{s.ID: s}
	out.Link = uc.linkProto(ctx, l, servers, ss)
	if touch {
		if links, err := uc.repo.ListLinks(ctx, userID); err == nil {
			sortLinks(links)
			all := uc.serverMap(ctx)
			for _, other := range links {
				if other.ID != l.ID {
					out.Others = append(out.Others, uc.linkProto(ctx, other, all, ss))
				}
			}
		}
	}
	return out, nil
}

// clientFor 绑定所在服务器的 MGP 客户端与 manifest。
func (uc *PassportUsecase) clientFor(ctx context.Context, serverID string) (*gen.GameServer, *mgp.Client, *mgp.Manifest, error) {
	s, client, err := uc.Servers().ClientOf(ctx, serverID)
	if err != nil {
		return nil, nil, nil, err
	}
	m, err := uc.Servers().Manifest(ctx, s)
	if err != nil {
		return nil, nil, nil, err
	}
	return s, client, m, nil
}

func hasEntry(list []mgp.ManifestEntry, id string) bool {
	for _, e := range list {
		if e.ID == id {
			return true
		}
	}
	return false
}

// clampQuery 解析前端透传的查询串，并限制 limit。
func clampQuery(raw string, def, maxLimit int, allowed map[string]bool) url.Values {
	in := MGPQuery(raw)
	out := url.Values{}
	for k, v := range in {
		if len(v) == 0 || (allowed != nil && !allowed[k]) {
			continue
		}
		out.Set(k, v[0])
	}
	n, err := strconv.Atoi(out.Get("limit"))
	if err != nil || n <= 0 {
		n = def
	}
	if n > maxLimit {
		n = maxLimit
	}
	out.Set("limit", strconv.Itoa(n))
	return out
}

// View 自己的分页数据（pid 强制为绑定的游戏账号）。
func (uc *PassportUsecase) View(ctx context.Context, userID, serverID, view, query string) (json.RawMessage, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	_, client, m, err := uc.clientFor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if !hasEntry(m.Views, view) {
		return nil, ErrPassportViewNotFound
	}
	q := clampQuery(query, 20, 200, nil)
	raw, err := client.Get(ctx, playerPath(l.PlayerID)+"/views/"+url.PathEscape(view), q)
	if err != nil {
		return nil, GameMGPError(err)
	}
	return raw, nil
}

var catalogQueryKeys = map[string]bool{"q": true, "characterId": true, "rarity": true, "cursor": true, "limit": true}

// Catalog 选择器数据（名称与素材 ID，每次最多 50 条）。
func (uc *PassportUsecase) Catalog(ctx context.Context, userID, serverID, kind, query string) (json.RawMessage, error) {
	if _, err := uc.gameLink(ctx, userID, serverID); err != nil {
		return nil, err
	}
	_, client, m, err := uc.clientFor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if !hasEntry(m.Catalogs, kind) {
		return nil, ErrPassportViewNotFound
	}
	raw, err := client.Get(ctx, "/catalogs/"+url.PathEscape(kind), clampQuery(query, 20, 50, catalogQueryKeys))
	if err != nil {
		return nil, GameMGPError(err)
	}
	return raw, nil
}

// Ongoing 正在进行的卡池（来自 gasha 报表，只保留 id、名称、横幅与起止时间，不含抽卡统计）。
func (uc *PassportUsecase) Ongoing(ctx context.Context, userID, serverID string) (json.RawMessage, error) {
	if _, err := uc.gameLink(ctx, userID, serverID); err != nil {
		return nil, err
	}
	_, client, m, err := uc.clientFor(ctx, serverID)
	if err != nil {
		return nil, err
	}
	type banner struct {
		GashaID       string `json:"gashaId"`
		Name          string `json:"name"`
		Type          int    `json:"type"`
		BannerAssetID string `json:"bannerAssetId"`
		StartTime     int64  `json:"startTime"`
		EndTime       int64  `json:"endTime"`
		Open          bool   `json:"open"`
	}
	out := []banner{}
	if !hasEntry(m.Reports, "gasha") {
		return jsonRaw(out), nil
	}
	var list []banner
	if err := client.GetInto(ctx, "/reports/gasha", nil, &list); err != nil {
		return nil, GameMGPError(err)
	}
	now := time.Now().UnixMilli()
	for _, b := range list {
		if !b.Open || (b.EndTime != 0 && b.EndTime < now) || (b.StartTime != 0 && b.StartTime > now) {
			continue
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if (out[i].EndTime == 0) != (out[j].EndTime == 0) {
			return out[j].EndTime == 0
		}
		return out[i].EndTime < out[j].EndTime
	})
	return jsonRaw(out), nil
}

func riskNeedsTotp(risk string) bool { return risk == mgp.RiskHigh || risk == mgp.RiskDanger }

// SelfService 已开放的自助指令与我的使用情况。
func (uc *PassportUsecase) SelfService(ctx context.Context, userID, serverID string) (*v1.ListPassportSelfServiceResponse, error) {
	if _, err := uc.gameLink(ctx, userID, serverID); err != nil {
		return nil, err
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	totpOn := u.TotpSecret != "" && u.TotpEnabledAt != nil
	list, err := uc.gm.SelfServiceFor(ctx, serverID, userID)
	if err != nil {
		return nil, err
	}
	out := &v1.ListPassportSelfServiceResponse{}
	for _, it := range list {
		p := it.Policy
		item := &v1.PassportSelfService{
			Command:       p.Command,
			Title:         p.Title,
			Group:         p.Group,
			Risk:          p.Risk,
			Target:        p.Target,
			Params:        p.Params,
			DailyLimit:    p.DailyLimit,
			CooldownSec:   p.CooldownSec,
			NeedApproval:  p.NeedApproval,
			PlayerNote:    p.PlayerNote,
			UsedToday:     int32(it.UsedToday),
			Remaining:     int32(it.Remaining),
			NextAvailable: tsOrNil(it.NextAvailable),
			HasPending:    it.HasPending,
			NeedTotp:      totpOn && riskNeedsTotp(p.Risk),
		}
		if it.Last != nil {
			item.LastUsedAt = timestamppb.New(it.Last.CreateTime)
			item.LastStatus = string(it.Last.Status)
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// changesOf 读出指令结果里某个玩家的变化与 detail。
func changesOf(result string, pid string) ([]*v1.PassportChange, *structpb.Value) {
	if result == "" {
		return nil, nil
	}
	var r mgp.CommandResult
	if json.Unmarshal([]byte(result), &r) != nil {
		return nil, nil
	}
	for _, t := range r.Results {
		if t.PID != pid && len(r.Results) > 1 {
			continue
		}
		var out []*v1.PassportChange
		for _, c := range t.Changes {
			out = append(out, &v1.PassportChange{Kind: c.Kind, ResourceId: c.ResourceID, Before: JSONValue(c.Before), After: JSONValue(c.After)})
		}
		return out, JSONValue(t.Detail)
	}
	return nil, nil
}

// ExecuteSelfService 执行一条自助指令（目标强制为我绑定的游戏账号）。
func (uc *PassportUsecase) ExecuteSelfService(ctx context.Context, c *passporttoken.Claims, req *v1.ExecutePassportSelfServiceRequest) (*v1.ExecutePassportSelfServiceResponse, error) {
	l, err := uc.gameLink(ctx, c.UserID, req.ServerId)
	if err != nil {
		return nil, err
	}
	u, err := uc.user(ctx, c.UserID)
	if err != nil {
		return nil, err
	}
	_, _, m, err := uc.clientFor(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	if cmd, ok := m.Command(req.Command); ok && riskNeedsTotp(cmd.Risk) && u.TotpSecret != "" && u.TotpEnabledAt != nil {
		if strings.TrimSpace(req.TotpCode) == "" {
			return nil, ErrPassportTotpRequired
		}
		if err := uc.checkSecondFactor(ctx, u, req.TotpCode, ""); err != nil {
			return nil, err
		}
	}
	params := ValueJSON(req.Params)
	if string(params) == "null" {
		params = json.RawMessage("{}")
	}
	reason := strings.TrimSpace(req.Reason)
	if len([]rune(reason)) > 200 {
		reason = string([]rune(reason)[:200])
	}
	client := PassportClientOf(ctx)
	actor := GmActor{
		UserID:   u.ID,
		Name:     displayName(u),
		Kind:     "player",
		Channel:  "passport",
		IP:       client.IP,
		Platform: Platform(client.UserAgent),
	}
	e, _, err := uc.gm.ExecuteSelfService(ctx, actor, req.ServerId, req.Command, l.PlayerID, params, reason, nil)
	if err != nil {
		return nil, err
	}
	out := &v1.ExecutePassportSelfServiceResponse{
		ExecutionId:  e.ID,
		Status:       string(e.Status),
		ErrorCode:    e.ErrorCode,
		ErrorMessage: e.ErrorMessage,
	}
	out.Changes, out.Detail = changesOf(e.Result, l.PlayerID)
	return out, nil
}

func (uc *PassportUsecase) recordOf(e *gen.GmExecution) *v1.PassportGameRecord {
	pid := ""
	if len(e.TargetPids) > 0 {
		pid = e.TargetPids[0]
	}
	changes, _ := changesOf(e.Result, pid)
	return &v1.PassportGameRecord{
		Id:           e.ID,
		Kind:         "self_service",
		Command:      e.Command,
		CommandTitle: e.CommandTitle,
		Status:       string(e.Status),
		Reason:       e.Reason,
		ReviewNote:   e.ReviewNote,
		ReviewedAt:   tsOrNil(e.ReviewedAt),
		CreateTime:   timestamppb.New(e.CreateTime),
		FinishedAt:   tsOrNil(e.FinishedAt),
		Changes:      changes,
		ErrorCode:    e.ErrorCode,
	}
}

func pageOf(page, size int32, def int) (int, int) {
	p, s := int(page), int(size)
	if p <= 0 {
		p = 1
	}
	if s <= 0 || s > 100 {
		s = def
	}
	return p, s
}

// Requests 我的申请（需要审批的自助指令）。
func (uc *PassportUsecase) Requests(ctx context.Context, userID string, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	if _, err := uc.gameLink(ctx, userID, req.ServerId); err != nil {
		return nil, err
	}
	page, size := pageOf(req.Page, req.PageSize, 10)
	list, total, err := uc.repo.ListPlayerExecutions(ctx, PassportExecFilter{UserID: userID, ServerID: req.ServerId, Reviewable: true, Offset: (page - 1) * size, Limit: size})
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListPassportGameRecordsResponse{Total: int32(total)}
	for _, e := range list {
		out.Items = append(out.Items, uc.recordOf(e))
	}
	return out, nil
}

// Records 操作记录：我在这台服务器上的自助操作与兑换码，从新到旧。
func (uc *PassportUsecase) Records(ctx context.Context, userID string, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	if _, err := uc.gameLink(ctx, userID, req.ServerId); err != nil {
		return nil, err
	}
	page, size := pageOf(req.Page, req.PageSize, 10)
	need := page * size
	execs, execTotal, err := uc.repo.ListPlayerExecutions(ctx, PassportExecFilter{UserID: userID, ServerID: req.ServerId, Limit: need})
	if err != nil {
		return nil, ErrSystem(err)
	}
	redeems, redeemTotal, err := uc.repo.ListRedeems(ctx, userID, req.ServerId, 0, need)
	if err != nil {
		return nil, ErrSystem(err)
	}
	var items []*v1.PassportGameRecord
	for _, e := range execs {
		items = append(items, uc.recordOf(e))
	}
	for _, r := range redeems {
		items = append(items, &v1.PassportGameRecord{
			Id:         r.ID,
			Kind:       "redeem",
			Status:     string(r.Result),
			CreateTime: timestamppb.New(r.CreateTime),
			Code:       r.Code,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreateTime.AsTime().After(items[j].CreateTime.AsTime()) })
	out := &v1.ListPassportGameRecordsResponse{Total: int32(execTotal + redeemTotal)}
	from := (page - 1) * size
	for i := from; i < len(items) && i < from+size; i++ {
		out.Items = append(out.Items, items[i])
	}
	return out, nil
}

// Redeem 兑换码（奖励进礼物箱）。
func (uc *PassportUsecase) Redeem(ctx context.Context, userID, serverID, code string) (*v1.RedeemPassportGiftCodeResponse, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	code = strings.TrimSpace(code)
	if code == "" || len(code) > 64 {
		return nil, ErrPassportInvalidArgument
	}
	client := PassportClientOf(ctx)
	res, err := uc.ops.RedeemGiftCode(ctx, GiftRedeemInput{
		ServerID:       serverID,
		PassportUserID: u.ID,
		PassportName:   displayName(u),
		PID:            l.PlayerID,
		Nickname:       l.Nickname,
		Code:           code,
		Channel:        "passport",
		Platform:       Platform(client.UserAgent),
		IP:             client.IP,
	})
	if err != nil {
		return nil, err
	}
	out := &v1.RedeemPassportGiftCodeResponse{Result: res.Result, ExecutionId: res.ExecutionID}
	for _, r := range res.Rewards {
		out.Rewards = append(out.Rewards, &v1.GameReward{ResourceType: r.ResourceType, ResourceId: r.ResourceID, Quantity: r.Quantity})
	}
	return out, nil
}
