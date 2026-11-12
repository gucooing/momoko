package data

import (
	"context"
	"strings"
	"time"

	"entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entlink "momoko/internal/data/ent/gen/gameaccountlink"
	entrevision "momoko/internal/data/ent/gen/gameconfigrevision"
	entexec "momoko/internal/data/ent/gen/gmexecution"
	"momoko/internal/data/ent/gen/predicate"
	entselfservice "momoko/internal/data/ent/gen/gmselfservice"
	entpuser "momoko/internal/data/ent/gen/passportuser"
)

type gmRepo struct {
	data *Data
}

func NewGmRepo(data *Data) biz.GmRepo {
	return &gmRepo{data: data}
}

func (r *gmRepo) CreateExecution(ctx context.Context, e *gen.GmExecution) (*gen.GmExecution, error) {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	c := r.data.db.GmExecution.Create().
		SetID(e.ID).
		SetServerID(e.ServerID).
		SetCommand(e.Command).
		SetCommandTitle(e.CommandTitle).
		SetCommandGroup(e.CommandGroup).
		SetRisk(e.Risk).
		SetTargetKind(e.TargetKind).
		SetActorID(e.ActorID).
		SetActorName(e.ActorName).
		SetActorKind(e.ActorKind).
		SetChannel(e.Channel).
		SetClientIP(e.ClientIP).
		SetPlatform(e.Platform).
		SetTargets(e.Targets).
		SetTargetPids(e.TargetPids).
		SetParams(e.Params).
		SetSummary(e.Summary).
		SetReason(e.Reason).
		SetConfirmed(e.Confirmed).
		SetDryRun(e.DryRun).
		SetStatus(e.Status).
		SetRequestID(e.RequestID).
		SetSource(e.Source)
	return c.Save(ctx)
}

func (r *gmRepo) SaveExecution(ctx context.Context, e *gen.GmExecution) (*gen.GmExecution, error) {
	u := r.data.db.GmExecution.UpdateOneID(e.ID).
		SetTargets(e.Targets).
		SetSummary(e.Summary).
		SetConfirmed(e.Confirmed).
		SetStatus(e.Status).
		SetResult(e.Result).
		SetSuccessCount(e.SuccessCount).
		SetFailedCount(e.FailedCount).
		SetBackupRef(e.BackupRef).
		SetErrorCode(e.ErrorCode).
		SetErrorMessage(e.ErrorMessage).
		SetReviewedBy(e.ReviewedBy).
		SetReviewerName(e.ReviewerName).
		SetReviewNote(e.ReviewNote).
		SetDurationMs(e.DurationMs)
	if e.ReviewedAt != nil {
		u.SetReviewedAt(*e.ReviewedAt)
	}
	if e.FinishedAt != nil {
		u.SetFinishedAt(*e.FinishedAt)
	}
	return u.Save(ctx)
}

func (r *gmRepo) GetExecution(ctx context.Context, id string) (*gen.GmExecution, error) {
	return r.data.db.GmExecution.Get(ctx, id)
}

// targetHas 匹配 target_pids JSON 数组里含有 pid 的记录。
func targetHas(pid string) predicate.GmExecution {
	return predicate.GmExecution(func(s *sql.Selector) {
		s.Where(sqljson.ValueContains(s.C(entexec.FieldTargetPids), pid))
	})
}

func (r *gmRepo) ListExecutions(ctx context.Context, f biz.GmExecutionFilter) ([]*gen.GmExecution, int, error) {
	q := r.data.db.GmExecution.Query().Where(entexec.ServerID(f.ServerID))
	if !f.IncludeDryRun {
		q = q.Where(entexec.DryRun(false))
	}
	if f.ActorKind != "" {
		q = q.Where(entexec.ActorKindEQ(entexec.ActorKind(f.ActorKind)))
	}
	if f.ActorID != "" {
		q = q.Where(entexec.ActorID(f.ActorID))
	}
	if len(f.Statuses) > 0 {
		statuses := make([]entexec.Status, 0, len(f.Statuses))
		for _, s := range f.Statuses {
			statuses = append(statuses, entexec.Status(s))
		}
		q = q.Where(entexec.StatusIn(statuses...))
	}
	if f.Risk != "" {
		q = q.Where(entexec.Risk(f.Risk))
	}
	if f.Command != "" {
		q = q.Where(entexec.Command(f.Command))
	}
	if f.PID != "" {
		q = q.Where(targetHas(f.PID))
	}
	if f.Since != nil {
		q = q.Where(entexec.CreateTimeGTE(*f.Since))
	}
	if f.Until != nil {
		q = q.Where(entexec.CreateTimeLT(*f.Until))
	}
	if f.Q != "" {
		kw := f.Q
		q = q.Where(entexec.Or(
			entexec.CommandContainsFold(kw),
			entexec.CommandTitleContainsFold(kw),
			entexec.ActorNameContainsFold(kw),
			entexec.RequestIDContainsFold(kw),
			targetHas(kw),
			predicate.GmExecution(func(s *sql.Selector) {
				s.Where(sql.ContainsFold(s.C(entexec.FieldTargets), kw))
			}),
		))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	size := f.Size
	if size <= 0 || size > 500 {
		size = 20
	}
	page := f.Page
	if page <= 0 {
		page = 1
	}
	items, err := q.Order(entexec.ByCreateTime(sql.OrderDesc())).Offset((page - 1) * size).Limit(size).All(ctx)
	return items, total, err
}

func (r *gmRepo) CountExecutions(ctx context.Context, serverID string, since, until *time.Time) (map[string]int32, error) {
	base := func() *gen.GmExecutionQuery {
		q := r.data.db.GmExecution.Query().Where(entexec.ServerID(serverID), entexec.DryRun(false))
		if since != nil {
			q = q.Where(entexec.CreateTimeGTE(*since))
		}
		if until != nil {
			q = q.Where(entexec.CreateTimeLT(*until))
		}
		return q
	}
	out := map[string]int32{}
	counts := []struct {
		key string
		q   *gen.GmExecutionQuery
	}{
		{"all", base()},
		{"admin", base().Where(entexec.ActorKindEQ(entexec.ActorKindAdmin))},
		{"player", base().Where(entexec.ActorKindEQ(entexec.ActorKindPlayer))},
		{"pending", base().Where(entexec.StatusEQ(entexec.StatusPending))},
		{"failed", base().Where(entexec.StatusIn(entexec.StatusFailed, entexec.StatusPartial, entexec.StatusRejected))},
	}
	for _, c := range counts {
		n, err := c.q.Count(ctx)
		if err != nil {
			return nil, err
		}
		out[c.key] = int32(n)
	}
	return out, nil
}

func (r *gmRepo) ListPending(ctx context.Context, serverID string, limit int) ([]*gen.GmExecution, error) {
	return r.data.db.GmExecution.Query().
		Where(entexec.ServerID(serverID), entexec.StatusEQ(entexec.StatusPending)).
		Order(entexec.ByCreateTime()).
		Limit(limit).
		All(ctx)
}

func (r *gmRepo) CountPending(ctx context.Context, serverID string) (int, error) {
	return r.data.db.GmExecution.Query().
		Where(entexec.ServerID(serverID), entexec.StatusEQ(entexec.StatusPending)).
		Count(ctx)
}

func (r *gmRepo) CountByTarget(ctx context.Context, serverID, pid string) (int, error) {
	return r.data.db.GmExecution.Query().
		Where(entexec.ServerID(serverID), entexec.DryRun(false), targetHas(pid)).
		Count(ctx)
}

// usedStatuses 计入次数限制的状态：成功、部分成功、执行中、待审批。
var usedStatuses = []entexec.Status{entexec.StatusSuccess, entexec.StatusPartial, entexec.StatusRunning, entexec.StatusPending}

func (r *gmRepo) CountUses(ctx context.Context, serverID, actorID, command string, since time.Time) (int, error) {
	return r.data.db.GmExecution.Query().
		Where(
			entexec.ServerID(serverID),
			entexec.ActorID(actorID),
			entexec.ActorKindEQ(entexec.ActorKindPlayer),
			entexec.Command(command),
			entexec.DryRun(false),
			entexec.StatusIn(usedStatuses...),
			entexec.CreateTimeGTE(since),
		).
		Count(ctx)
}

func (r *gmRepo) LastUse(ctx context.Context, serverID, actorID, command string) (*gen.GmExecution, error) {
	e, err := r.data.db.GmExecution.Query().
		Where(
			entexec.ServerID(serverID),
			entexec.ActorID(actorID),
			entexec.ActorKindEQ(entexec.ActorKindPlayer),
			entexec.Command(command),
			entexec.DryRun(false),
			entexec.StatusIn(usedStatuses...),
		).
		Order(entexec.ByCreateTime(sql.OrderDesc())).
		First(ctx)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	return e, err
}

func (r *gmRepo) PriorUses(ctx context.Context, serverID, command, pid string) (int, error) {
	return r.data.db.GmExecution.Query().
		Where(
			entexec.ServerID(serverID),
			entexec.Command(command),
			entexec.DryRun(false),
			entexec.StatusIn(entexec.StatusSuccess, entexec.StatusPartial),
			targetHas(pid),
		).
		Count(ctx)
}

func (r *gmRepo) ListSelfService(ctx context.Context, serverID string) ([]*gen.GmSelfService, error) {
	return r.data.db.GmSelfService.Query().Where(entselfservice.ServerID(serverID)).All(ctx)
}

func (r *gmRepo) GetSelfService(ctx context.Context, serverID, command string) (*gen.GmSelfService, error) {
	p, err := r.data.db.GmSelfService.Query().
		Where(entselfservice.ServerID(serverID), entselfservice.Command(command)).
		Only(ctx)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	return p, err
}

func (r *gmRepo) SaveSelfService(ctx context.Context, p *gen.GmSelfService) (*gen.GmSelfService, error) {
	err := r.data.db.GmSelfService.Create().
		SetID(uuid.NewString()).
		SetServerID(p.ServerID).
		SetCommand(p.Command).
		SetEnabled(p.Enabled).
		SetDailyLimit(p.DailyLimit).
		SetCooldownSec(p.CooldownSec).
		SetNeedApproval(p.NeedApproval).
		SetPlayerNote(p.PlayerNote).
		SetUpdatedBy(p.UpdatedBy).
		SetUpdatedByName(p.UpdatedByName).
		OnConflictColumns(entselfservice.FieldServerID, entselfservice.FieldCommand).
		Update(func(u *gen.GmSelfServiceUpsert) {
			u.UpdateEnabled().
				UpdateDailyLimit().
				UpdateCooldownSec().
				UpdateNeedApproval().
				UpdatePlayerNote().
				UpdateUpdatedBy().
				UpdateUpdatedByName().
				SetUpdateTime(time.Now())
		}).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	return r.GetSelfService(ctx, p.ServerID, p.Command)
}

func (r *gmRepo) GetConfigDraft(ctx context.Context, serverID, domain string) (*gen.GameConfigRevision, error) {
	d, err := r.data.db.GameConfigRevision.Query().
		Where(entrevision.ServerID(serverID), entrevision.Domain(domain), entrevision.StateEQ(entrevision.StateDraft)).
		Order(entrevision.ByUpdateTime(sql.OrderDesc())).
		First(ctx)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	return d, err
}

func (r *gmRepo) SaveConfigDraft(ctx context.Context, rev *gen.GameConfigRevision) (*gen.GameConfigRevision, error) {
	existing, err := r.GetConfigDraft(ctx, rev.ServerID, rev.Domain)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return r.data.db.GameConfigRevision.UpdateOneID(existing.ID).
			SetContent(rev.Content).
			SetBaseRevision(rev.BaseRevision).
			SetAuthorID(rev.AuthorID).
			SetAuthorName(rev.AuthorName).
			SetComment(rev.Comment).
			Save(ctx)
	}
	return r.data.db.GameConfigRevision.Create().
		SetID(uuid.NewString()).
		SetServerID(rev.ServerID).
		SetDomain(rev.Domain).
		SetContent(rev.Content).
		SetBaseRevision(rev.BaseRevision).
		SetAuthorID(rev.AuthorID).
		SetAuthorName(rev.AuthorName).
		SetComment(rev.Comment).
		SetState(entrevision.StateDraft).
		Save(ctx)
}

func (r *gmRepo) DeleteConfigDraft(ctx context.Context, serverID, domain string) error {
	_, err := r.data.db.GameConfigRevision.Delete().
		Where(entrevision.ServerID(serverID), entrevision.Domain(domain), entrevision.StateEQ(entrevision.StateDraft)).
		Exec(ctx)
	return err
}

func (r *gmRepo) PublishConfig(ctx context.Context, rev *gen.GameConfigRevision) (*gen.GameConfigRevision, error) {
	var out *gen.GameConfigRevision
	err := WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		last, err := tx.GameConfigRevision.Query().
			Where(entrevision.ServerID(rev.ServerID), entrevision.Domain(rev.Domain), entrevision.StateEQ(entrevision.StatePublished)).
			Order(entrevision.BySeq(sql.OrderDesc())).
			First(ctx)
		seq := 1
		if err == nil {
			seq = last.Seq + 1
		} else if !gen.IsNotFound(err) {
			return err
		}
		c := tx.GameConfigRevision.Create().
			SetID(uuid.NewString()).
			SetServerID(rev.ServerID).
			SetDomain(rev.Domain).
			SetSeq(seq).
			SetRevision(rev.Revision).
			SetBaseRevision(rev.BaseRevision).
			SetContent(rev.Content).
			SetAuthorID(rev.AuthorID).
			SetAuthorName(rev.AuthorName).
			SetComment(rev.Comment).
			SetState(entrevision.StatePublished).
			SetExecutionID(rev.ExecutionID)
		if rev.PublishedAt != nil {
			c.SetPublishedAt(*rev.PublishedAt)
		}
		if out, err = c.Save(ctx); err != nil {
			return err
		}
		_, err = tx.GameConfigRevision.Delete().
			Where(entrevision.ServerID(rev.ServerID), entrevision.Domain(rev.Domain), entrevision.StateEQ(entrevision.StateDraft)).
			Exec(ctx)
		return err
	})
	return out, err
}

func (r *gmRepo) ListConfigRevisions(ctx context.Context, serverID, domain string, page, size int) ([]*gen.GameConfigRevision, int, error) {
	q := r.data.db.GameConfigRevision.Query().Where(entrevision.ServerID(serverID), entrevision.Domain(domain))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if size <= 0 || size > 200 {
		size = 20
	}
	if page <= 0 {
		page = 1
	}
	items, err := q.Select(
		entrevision.FieldID, entrevision.FieldServerID, entrevision.FieldDomain, entrevision.FieldSeq,
		entrevision.FieldRevision, entrevision.FieldBaseRevision, entrevision.FieldAuthorID, entrevision.FieldAuthorName,
		entrevision.FieldComment, entrevision.FieldState, entrevision.FieldPublishedAt, entrevision.FieldExecutionID,
		entrevision.FieldCreateTime, entrevision.FieldUpdateTime,
	).
		Order(entrevision.ByCreateTime(sql.OrderDesc())).
		Offset((page - 1) * size).Limit(size).
		All(ctx)
	return items, total, err
}

func (r *gmRepo) GetConfigRevision(ctx context.Context, id string) (*gen.GameConfigRevision, error) {
	return r.data.db.GameConfigRevision.Get(ctx, id)
}

func (r *gmRepo) LastPublished(ctx context.Context, serverID string) (map[string]*gen.GameConfigRevision, error) {
	items, err := r.data.db.GameConfigRevision.Query().
		Where(entrevision.ServerID(serverID), entrevision.StateEQ(entrevision.StatePublished)).
		Select(
			entrevision.FieldID, entrevision.FieldDomain, entrevision.FieldSeq, entrevision.FieldRevision,
			entrevision.FieldAuthorName, entrevision.FieldPublishedAt, entrevision.FieldCreateTime,
		).
		Order(entrevision.BySeq()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := map[string]*gen.GameConfigRevision{}
	for _, it := range items {
		out[it.Domain] = it
	}
	return out, nil
}

// gmLinksOf 把绑定行与通行证账号（passport_users）组合成 GmLink。
func (r *gmRepo) gmLinksOf(ctx context.Context, links []*gen.GameAccountLink) ([]*biz.GmLink, error) {
	if len(links) == 0 {
		return nil, nil
	}
	ids := make([]string, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.PassportUserID)
	}
	users, err := r.data.db.PassportUser.Query().Where(entpuser.IDIn(ids...)).All(ctx)
	if err != nil {
		return nil, err
	}
	byID := map[string]*gen.PassportUser{}
	for _, u := range users {
		byID[u.ID] = u
	}
	out := make([]*biz.GmLink, 0, len(links))
	for _, l := range links {
		gl := &biz.GmLink{PID: l.PlayerID, PassportUserID: l.PassportUserID, Method: string(l.Method), Nickname: l.Nickname, LinkedAt: l.LinkedAt}
		if u, ok := byID[l.PassportUserID]; ok {
			gl.Username, gl.Name = u.Username, u.Name
		}
		out = append(out, gl)
	}
	return out, nil
}

func (r *gmRepo) LinksByPIDs(ctx context.Context, serverID string, pids []string) ([]*biz.GmLink, error) {
	if len(pids) == 0 {
		return nil, nil
	}
	links, err := r.data.db.GameAccountLink.Query().
		Where(entlink.ServerID(serverID), entlink.StateEQ(entlink.StateActive), entlink.PlayerIDIn(pids...)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return r.gmLinksOf(ctx, links)
}

func (r *gmRepo) LinkedPIDs(ctx context.Context, serverID string) (map[string]struct{}, error) {
	pids, err := r.data.db.GameAccountLink.Query().
		Where(entlink.ServerID(serverID), entlink.StateEQ(entlink.StateActive), entlink.PlayerIDNEQ("")).
		Select(entlink.FieldPlayerID).
		Strings(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(pids))
	for _, p := range pids {
		out[p] = struct{}{}
	}
	return out, nil
}

func (r *gmRepo) FindLinksByUser(ctx context.Context, serverID, q string, limit int) ([]*biz.GmLink, error) {
	q = strings.TrimSpace(q)
	if q == "" {
		return nil, nil
	}
	userIDs, err := r.data.db.PassportUser.Query().
		Where(entpuser.Or(entpuser.UsernameContainsFold(q), entpuser.NameContainsFold(q))).
		Limit(200).
		IDs(ctx)
	if err != nil || len(userIDs) == 0 {
		return nil, err
	}
	links, err := r.data.db.GameAccountLink.Query().
		Where(
			entlink.ServerID(serverID),
			entlink.StateEQ(entlink.StateActive),
			entlink.PassportUserIDIn(userIDs...),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return r.gmLinksOf(ctx, links)
}
