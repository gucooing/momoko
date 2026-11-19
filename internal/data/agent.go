package data

import (
	"context"
	"strings"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/agentevent"
	"momoko/internal/data/ent/gen/agentmcpserver"
	"momoko/internal/data/ent/gen/agentprovider"
	"momoko/internal/data/ent/gen/agentruntime"
	"momoko/internal/data/ent/gen/agentsession"
	"momoko/internal/data/ent/gen/agentskill"
	"momoko/internal/data/ent/gen/agentworkspace"
	"momoko/internal/data/ent/gen/predicate"
	"momoko/internal/data/ent/gen/user"
)

type agentRepo struct {
	data *Data
}

func NewAgentRepo(data *Data) biz.AgentRepo {
	return &agentRepo{data: data}
}

// —— 运行时 ——

func (r *agentRepo) ListAgentRuntimes(ctx context.Context) ([]*gen.AgentRuntime, error) {
	return r.data.db.AgentRuntime.Query().
		Order(gen.Asc(agentruntime.FieldSort), gen.Asc(agentruntime.FieldCreateTime)).
		All(ctx)
}

func (r *agentRepo) GetAgentRuntime(ctx context.Context, id string) (*gen.AgentRuntime, error) {
	return r.data.db.AgentRuntime.Get(ctx, id)
}

func (r *agentRepo) CountAgentRuntimes(ctx context.Context) (int, error) {
	return r.data.db.AgentRuntime.Query().Count(ctx)
}

func (r *agentRepo) SaveAgentRuntime(ctx context.Context, rt *gen.AgentRuntime) error {
	return r.data.db.AgentRuntime.Create().
		SetID(rt.ID).
		SetName(rt.Name).
		SetKind(rt.Kind).
		SetManaged(rt.Managed).
		SetCommand(rt.Command).
		SetVersion(rt.Version).
		SetReady(rt.Ready).
		SetAgentName(rt.AgentName).
		SetAgentVersion(rt.AgentVersion).
		SetError(rt.Error).
		SetNillableDetectedAt(rt.DetectedAt).
		SetCaps(rt.Caps).
		SetAuthLabel(rt.AuthLabel).
		SetAuthDetail(rt.AuthDetail).
		SetLoggedIn(rt.LoggedIn).
		SetCatalog(rt.Catalog).
		SetSort(rt.Sort).
		OnConflictColumns(agentruntime.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (r *agentRepo) DeleteAgentRuntime(ctx context.Context, id string) error {
	return r.data.db.AgentRuntime.DeleteOneID(id).Exec(ctx)
}

// —— 模型供应商 ——

func (r *agentRepo) ListAgentProviders(ctx context.Context) ([]*gen.AgentProvider, error) {
	return r.data.db.AgentProvider.Query().Order(gen.Asc(agentprovider.FieldCreateTime)).All(ctx)
}

func (r *agentRepo) GetAgentProvider(ctx context.Context, id string) (*gen.AgentProvider, error) {
	return r.data.db.AgentProvider.Get(ctx, id)
}

func (r *agentRepo) SaveAgentProvider(ctx context.Context, p *gen.AgentProvider) error {
	return r.data.db.AgentProvider.Create().
		SetID(p.ID).
		SetVendor(p.Vendor).
		SetName(p.Name).
		SetStatus(p.Status).
		SetStatusText(p.StatusText).
		SetBaseURL(p.BaseURL).
		SetAPIKey(p.APIKey).
		SetModels(p.Models).
		SetMainModel(p.MainModel).
		SetFastModel(p.FastModel).
		SetEnv(p.Env).
		SetTestLatencyMs(p.TestLatencyMs).
		SetNillableTestedAt(p.TestedAt).
		OnConflictColumns(agentprovider.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (r *agentRepo) DeleteAgentProvider(ctx context.Context, id string) error {
	return r.data.db.AgentProvider.DeleteOneID(id).Exec(ctx)
}

// —— 工作区 ——

func (r *agentRepo) ListAgentWorkspaces(ctx context.Context) ([]*gen.AgentWorkspace, error) {
	return r.data.db.AgentWorkspace.Query().
		Order(gen.Asc(agentworkspace.FieldCreateTime)).
		All(ctx)
}

func (r *agentRepo) GetAgentWorkspace(ctx context.Context, id string) (*gen.AgentWorkspace, error) {
	return r.data.db.AgentWorkspace.Get(ctx, id)
}

func (r *agentRepo) FindAgentWorkspaceByPath(ctx context.Context, path string) (*gen.AgentWorkspace, error) {
	return r.data.db.AgentWorkspace.Query().Where(agentworkspace.PathEQ(path)).Only(ctx)
}

func (r *agentRepo) SaveAgentWorkspace(ctx context.Context, w *gen.AgentWorkspace) error {
	return r.data.db.AgentWorkspace.Create().
		SetID(w.ID).
		SetUserID(w.UserID).
		SetName(w.Name).
		SetPath(w.Path).
		SetGitURL(w.GitURL).
		SetCloneStatus(w.CloneStatus).
		SetCloneError(w.CloneError).
		SetRuntimeID(w.RuntimeID).
		SetProviderID(w.ProviderID).
		SetModel(w.Model).
		SetEffort(w.Effort).
		SetMode(w.Mode).
		SetMcpIds(nonNilStrings(w.McpIds)).
		OnConflictColumns(agentworkspace.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (r *agentRepo) DeleteAgentWorkspace(ctx context.Context, id string) error {
	return r.data.db.AgentWorkspace.DeleteOneID(id).Exec(ctx)
}

// AgentWorkspaceStats 每个工作区的会话数和最近活动时间，一条分组查询算完。
func (r *agentRepo) AgentWorkspaceStats(ctx context.Context, ids []string) (map[string]biz.AgentWorkspaceStat, error) {
	out := map[string]biz.AgentWorkspaceStat{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		WorkspaceID string `json:"workspace_id"`
		Count       int    `json:"count"`
		Active      any    `json:"active"`
	}
	err := r.data.db.AgentSession.Query().
		Where(agentsession.WorkspaceIDIn(ids...)).
		GroupBy(agentsession.FieldWorkspaceID).
		Aggregate(gen.Count(), gen.As(gen.Max(agentsession.FieldActiveAt), "active")).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.WorkspaceID] = biz.AgentWorkspaceStat{Sessions: row.Count, ActiveAt: scanTime(row.Active)}
	}
	return out, nil
}

// scanTime 聚合函数返回的时间：PostgreSQL、MySQL 的驱动直接给 time.Time，SQLite 给它存的文本。
func scanTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		parsed, _ := time.Parse("2006-01-02 15:04:05.999999999-07:00", t)
		return parsed
	}
	return time.Time{}
}

// —— MCP ——

func (r *agentRepo) ListAgentMcpServers(ctx context.Context) ([]*gen.AgentMcpServer, error) {
	return r.data.db.AgentMcpServer.Query().
		Order(gen.Asc(agentmcpserver.FieldCreateTime)).
		All(ctx)
}

func (r *agentRepo) ListAgentMcpServersByIDs(ctx context.Context, ids []string) ([]*gen.AgentMcpServer, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return r.data.db.AgentMcpServer.Query().
		Where(agentmcpserver.IDIn(ids...)).
		Order(gen.Asc(agentmcpserver.FieldCreateTime)).
		All(ctx)
}

func (r *agentRepo) AgentMcpServerNames(ctx context.Context) ([]string, error) {
	return r.data.db.AgentMcpServer.Query().Select(agentmcpserver.FieldName).Strings(ctx)
}

func (r *agentRepo) GetAgentMcpServer(ctx context.Context, id string) (*gen.AgentMcpServer, error) {
	return r.data.db.AgentMcpServer.Get(ctx, id)
}

func (r *agentRepo) FindAgentMcpServerByName(ctx context.Context, name string) (*gen.AgentMcpServer, error) {
	return r.data.db.AgentMcpServer.Query().Where(agentmcpserver.NameEQ(name)).Only(ctx)
}

func (r *agentRepo) SaveAgentMcpServer(ctx context.Context, m *gen.AgentMcpServer) error {
	return r.data.db.AgentMcpServer.Create().
		SetID(m.ID).
		SetName(m.Name).
		SetDescription(m.Description).
		SetTransport(m.Transport).
		SetCommand(m.Command).
		SetArgs(nonNilStrings(m.Args)).
		SetEnv(m.Env).
		SetURL(m.URL).
		SetHeaders(m.Headers).
		SetDefaultOn(m.DefaultOn).
		SetStatus(m.Status).
		SetStatusText(m.StatusText).
		SetToolCount(m.ToolCount).
		SetNillableTestedAt(m.TestedAt).
		OnConflictColumns(agentmcpserver.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (r *agentRepo) DeleteAgentMcpServer(ctx context.Context, id string) error {
	return r.data.db.AgentMcpServer.DeleteOneID(id).Exec(ctx)
}

// —— 技能 ——

func (r *agentRepo) ListAgentSkills(ctx context.Context) ([]*gen.AgentSkill, error) {
	return r.data.db.AgentSkill.Query().
		Order(gen.Asc(agentskill.FieldCreateTime)).
		All(ctx)
}

func (r *agentRepo) ListEnabledAgentSkills(ctx context.Context) ([]*gen.AgentSkill, error) {
	return r.data.db.AgentSkill.Query().
		Where(agentskill.EnabledEQ(true)).
		Order(gen.Asc(agentskill.FieldName)).
		All(ctx)
}

func (r *agentRepo) GetAgentSkill(ctx context.Context, id string) (*gen.AgentSkill, error) {
	return r.data.db.AgentSkill.Get(ctx, id)
}

func (r *agentRepo) FindAgentSkillByName(ctx context.Context, name string) (*gen.AgentSkill, error) {
	return r.data.db.AgentSkill.Query().Where(agentskill.NameEQ(name)).Only(ctx)
}

func (r *agentRepo) SaveAgentSkill(ctx context.Context, s *gen.AgentSkill) error {
	return r.data.db.AgentSkill.Create().
		SetID(s.ID).
		SetName(s.Name).
		SetDescription(s.Description).
		SetSource(s.Source).
		SetLocation(s.Location).
		SetEnabled(s.Enabled).
		SetFileCount(s.FileCount).
		OnConflictColumns(agentskill.FieldID).
		UpdateNewValues().
		Exec(ctx)
}

func (r *agentRepo) DeleteAgentSkill(ctx context.Context, id string) error {
	return r.data.db.AgentSkill.DeleteOneID(id).Exec(ctx)
}

// —— 会话 ——

func agentSessionCreate(c *gen.AgentSessionClient, s *gen.AgentSession) *gen.AgentSessionCreate {
	return c.Create().
		SetID(s.ID).
		SetUserID(s.UserID).
		SetTitle(s.Title).
		SetTitleCustom(s.TitleCustom).
		SetRuntimeID(s.RuntimeID).
		SetWorkspaceID(s.WorkspaceID).
		SetCwd(s.Cwd).
		SetExtraDirs(nonNilStrings(s.ExtraDirs)).
		SetWorktree(s.Worktree).
		SetProviderID(s.ProviderID).
		SetModel(s.Model).
		SetEffort(s.Effort).
		SetMode(s.Mode).
		SetStatus(s.Status).
		SetAcpSessionID(s.AcpSessionID).
		SetParentID(s.ParentID).
		SetDetached(s.Detached).
		SetActiveAt(s.ActiveAt).
		SetContextUsed(s.ContextUsed).
		SetContextSize(s.ContextSize).
		SetMcpIds(nonNilStrings(s.McpIds)).
		SetState(s.State).
		SetOrigin(s.Origin)
}

func (r *agentRepo) CreateAgentSession(ctx context.Context, s *gen.AgentSession) (*gen.AgentSession, error) {
	return agentSessionCreate(r.data.db.AgentSession, s).Save(ctx)
}

// ForkAgentSession 建复刻出来的会话，并把原会话的事件整份带过去（分批读写），在一个事务里做完。
func (r *agentRepo) ForkAgentSession(ctx context.Context, fork *gen.AgentSession, from string) (*gen.AgentSession, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := agentSessionCreate(tx.AgentSession, fork).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	const batch = 500
	var last int64 = -1
	for {
		rows, err := tx.AgentEvent.Query().
			Where(agentevent.SessionIDEQ(from), agentevent.SeqGT(last)).
			Order(gen.Asc(agentevent.FieldSeq)).
			Limit(batch).
			All(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if len(rows) == 0 {
			break
		}
		builders := make([]*gen.AgentEventCreate, 0, len(rows))
		for _, e := range rows {
			builders = append(builders, tx.AgentEvent.Create().
				SetSessionID(fork.ID).SetSeq(e.Seq).SetRev(e.Rev).SetType(e.Type).SetSub(e.Sub).
				SetPayload(e.Payload).SetCreateTime(e.CreateTime))
			last = e.Seq
		}
		if err := tx.AgentEvent.CreateBulk(builders...).Exec(ctx); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		if len(rows) < batch {
			break
		}
	}
	return created, tx.Commit()
}

func (r *agentRepo) GetAgentSession(ctx context.Context, id string) (*gen.AgentSession, error) {
	return r.data.db.AgentSession.Get(ctx, id)
}

// UpdateAgentSession 改一个会话并返回改后的样子；fn 里加了条件而条件不满足时返回 NotFound。
func (r *agentRepo) UpdateAgentSession(ctx context.Context, id string, fn func(u *gen.AgentSessionUpdateOne)) (*gen.AgentSession, error) {
	update := r.data.db.AgentSession.UpdateOneID(id)
	fn(update)
	return update.Save(ctx)
}

func agentSessionScope(f biz.AgentSessionFilter) []predicate.AgentSession {
	var where []predicate.AgentSession
	if f.UserID != "" {
		where = append(where, agentsession.UserIDEQ(f.UserID))
	}
	if kw := strings.TrimSpace(f.Keyword); kw != "" {
		where = append(where, agentsession.TitleContainsFold(kw))
	}
	return where
}

// ListAgentSessions 状态筛选、标题搜索、分页都在 SQL 里；各状态的数量用一条分组查询（不受状态筛选影响）。
func (r *agentRepo) ListAgentSessions(ctx context.Context, f biz.AgentSessionFilter) ([]*gen.AgentSession, int64, biz.AgentSessionCounts, error) {
	scope := agentSessionScope(f)
	counts := biz.AgentSessionCounts{}
	var grouped []struct {
		Status string `json:"status"`
		Count  int64  `json:"count"`
	}
	if err := r.data.db.AgentSession.Query().Where(scope...).
		GroupBy(agentsession.FieldStatus).
		Aggregate(gen.Count()).
		Scan(ctx, &grouped); err != nil {
		return nil, 0, counts, err
	}
	byStatus := map[agentsession.Status]int64{}
	for _, g := range grouped {
		counts.All += g.Count
		byStatus[agentsession.Status(g.Status)] = g.Count
	}
	counts.Running, counts.Waiting = byStatus[agentsession.StatusRunning], byStatus[agentsession.StatusWaiting]
	counts.Idle, counts.Error = byStatus[agentsession.StatusIdle], byStatus[agentsession.StatusError]
	counts.Archived = byStatus[agentsession.StatusArchived]

	query := r.data.db.AgentSession.Query().Where(scope...)
	total := counts.All
	if f.Status != "" && f.Status != "all" {
		status := agentsession.Status(f.Status)
		if err := agentsession.StatusValidator(status); err != nil {
			return nil, 0, counts, err
		}
		query = query.Where(agentsession.StatusEQ(status))
		total = byStatus[status]
	}
	rows, err := query.
		Order(gen.Desc(agentsession.FieldActiveAt), gen.Desc(agentsession.FieldID)).
		Offset((f.Page - 1) * f.PageSize).
		Limit(f.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, counts, err
	}
	return rows, total, counts, nil
}

func (r *agentRepo) ListAgentSidebarSessions(ctx context.Context, userID, keyword string, limit int) ([]*gen.AgentSession, error) {
	query := r.data.db.AgentSession.Query().
		Where(agentsession.UserIDEQ(userID), agentsession.StatusNEQ(agentsession.StatusArchived))
	if kw := strings.TrimSpace(keyword); kw != "" {
		query = query.Where(agentsession.TitleContainsFold(kw))
	}
	return query.
		Order(gen.Desc(agentsession.FieldActiveAt), gen.Desc(agentsession.FieldID)).
		Limit(limit).
		All(ctx)
}

// ListAgentSessionsByIDs 这些 id 里的会话；userID 不为空时只要这个人的。
func (r *agentRepo) ListAgentSessionsByIDs(ctx context.Context, ids []string, userID string) ([]*gen.AgentSession, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	query := r.data.db.AgentSession.Query().Where(agentsession.IDIn(ids...))
	if userID != "" {
		query = query.Where(agentsession.UserIDEQ(userID))
	}
	return query.All(ctx)
}

// ListArchivedAgentSessions 已归档的会话；userID 不为空时只要这个人的。
func (r *agentRepo) ListArchivedAgentSessions(ctx context.Context, userID string) ([]*gen.AgentSession, error) {
	query := r.data.db.AgentSession.Query().Where(agentsession.StatusEQ(agentsession.StatusArchived))
	if userID != "" {
		query = query.Where(agentsession.UserIDEQ(userID))
	}
	return query.All(ctx)
}

// ListAgentForks 这些会话直接复刻出来、还没移到外层的会话。
func (r *agentRepo) ListAgentForks(ctx context.Context, parentIDs []string) ([]*gen.AgentSession, error) {
	if len(parentIDs) == 0 {
		return nil, nil
	}
	return r.data.db.AgentSession.Query().
		Where(agentsession.ParentIDIn(parentIDs...), agentsession.DetachedEQ(false)).
		All(ctx)
}

func (r *agentRepo) CountAgentForks(ctx context.Context, parentIDs []string) (map[string]int, error) {
	out := map[string]int{}
	if len(parentIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		ParentID string `json:"parent_id"`
		Count    int    `json:"count"`
	}
	err := r.data.db.AgentSession.Query().
		Where(agentsession.ParentIDIn(parentIDs...), agentsession.DetachedEQ(false)).
		GroupBy(agentsession.FieldParentID).
		Aggregate(gen.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.ParentID] = row.Count
	}
	return out, nil
}

func (r *agentRepo) DetachAgentForks(ctx context.Context, parentIDs []string) error {
	if len(parentIDs) == 0 {
		return nil
	}
	return r.data.db.AgentSession.Update().
		Where(agentsession.ParentIDIn(parentIDs...), agentsession.DetachedEQ(false)).
		SetDetached(true).
		Exec(ctx)
}

// ArchiveAgentSessions 归档只动空闲和出错的（运行中、等待批准的不归档），取消归档回到空闲。返回改了的会话。
func (r *agentRepo) ArchiveAgentSessions(ctx context.Context, ids []string, archived bool) ([]*gen.AgentSession, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	from := []agentsession.Status{agentsession.StatusArchived}
	to := agentsession.StatusIdle
	if archived {
		from, to = []agentsession.Status{agentsession.StatusIdle, agentsession.StatusError}, agentsession.StatusArchived
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	where := []predicate.AgentSession{agentsession.IDIn(ids...), agentsession.StatusIn(from...)}
	changed, err := tx.AgentSession.Query().Where(where...).IDs(ctx)
	if err == nil && len(changed) > 0 {
		err = tx.AgentSession.Update().Where(agentsession.IDIn(changed...)).SetStatus(to).Exec(ctx)
	}
	var rows []*gen.AgentSession
	if err == nil && len(changed) > 0 {
		rows, err = tx.AgentSession.Query().Where(agentsession.IDIn(changed...)).All(ctx)
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return rows, tx.Commit()
}

func (r *agentRepo) DeleteAgentSessions(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	if _, err := tx.AgentEvent.Delete().Where(agentevent.SessionIDIn(ids...)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.AgentSession.Delete().Where(agentsession.IDIn(ids...)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ResetLiveAgentSessions Momoko 重启后 Agent 进程都没了：把还标着运行中、等待批准的会话改回空闲。
func (r *agentRepo) ResetLiveAgentSessions(ctx context.Context) error {
	return r.data.db.AgentSession.Update().
		Where(agentsession.StatusIn(agentsession.StatusRunning, agentsession.StatusWaiting)).
		SetStatus(agentsession.StatusIdle).
		Exec(ctx)
}

// —— 事件 ——

// SaveAgentEvents 按（会话，序号）批量写入或更新事件。
func (r *agentRepo) SaveAgentEvents(ctx context.Context, events []*gen.AgentEvent) error {
	if len(events) == 0 {
		return nil
	}
	builders := make([]*gen.AgentEventCreate, 0, len(events))
	for _, e := range events {
		builders = append(builders, r.data.db.AgentEvent.Create().
			SetSessionID(e.SessionID).SetSeq(e.Seq).SetRev(e.Rev).SetType(e.Type).SetSub(e.Sub).SetPayload(e.Payload))
	}
	return r.data.db.AgentEvent.CreateBulk(builders...).
		OnConflictColumns(agentevent.FieldSessionID, agentevent.FieldSeq).
		Update(func(u *gen.AgentEventUpsert) {
			u.UpdateRev()
			u.UpdateType()
			u.UpdateSub()
			u.UpdatePayload()
		}).
		Exec(ctx)
}

// ListAgentEventsAfter 修订号大于 afterRev 的事件（断线重连补发、导出），按序号排。
func (r *agentRepo) ListAgentEventsAfter(ctx context.Context, sessionID string, afterRev int64) ([]*gen.AgentEvent, error) {
	return r.data.db.AgentEvent.Query().
		Where(agentevent.SessionIDEQ(sessionID), agentevent.RevGT(afterRev)).
		Order(gen.Asc(agentevent.FieldSeq)).
		All(ctx)
}

// ListAgentEventsBefore 序号小于 beforeSeq（0 = 不限）的最后 limit 条事件，按序号排；more 表示更早的还有。
func (r *agentRepo) ListAgentEventsBefore(ctx context.Context, sessionID string, beforeSeq int64, limit int) ([]*gen.AgentEvent, bool, error) {
	query := r.data.db.AgentEvent.Query().Where(agentevent.SessionIDEQ(sessionID))
	if beforeSeq > 0 {
		query = query.Where(agentevent.SeqLT(beforeSeq))
	}
	rows, err := query.Order(gen.Desc(agentevent.FieldSeq)).Limit(limit + 1).All(ctx)
	if err != nil {
		return nil, false, err
	}
	more := len(rows) > limit
	if more {
		rows = rows[:limit]
	}
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, more, nil
}

func (r *agentRepo) MaxAgentEvent(ctx context.Context, sessionID string) (int64, int64, error) {
	var rows []struct {
		Seq *int64 `json:"seq"`
		Rev *int64 `json:"rev"`
	}
	err := r.data.db.AgentEvent.Query().
		Where(agentevent.SessionIDEQ(sessionID)).
		Aggregate(gen.As(gen.Max(agentevent.FieldSeq), "seq"), gen.As(gen.Max(agentevent.FieldRev), "rev")).
		Scan(ctx, &rows)
	if err != nil || len(rows) == 0 || rows[0].Seq == nil {
		return 0, 0, err
	}
	return *rows[0].Seq, *rows[0].Rev, nil
}

// —— 统计与用户 ——

// AgentCounts 管理页各标签上的数量：每张表一条分组查询。
func (r *agentRepo) AgentCounts(ctx context.Context) (*biz.AgentCounts, error) {
	db := r.data.db
	out := &biz.AgentCounts{}
	var sessions []struct {
		Status agentsession.Status `json:"status"`
		Count  int                 `json:"count"`
	}
	if err := db.AgentSession.Query().GroupBy(agentsession.FieldStatus).Aggregate(gen.Count()).Scan(ctx, &sessions); err != nil {
		return nil, err
	}
	for _, g := range sessions {
		out.Sessions += g.Count
		switch g.Status {
		case agentsession.StatusRunning:
			out.SessionsRunning = g.Count
		case agentsession.StatusWaiting:
			out.SessionsWaiting = g.Count
		}
	}
	var err error
	if out.Workspaces, err = db.AgentWorkspace.Query().Count(ctx); err != nil {
		return nil, err
	}
	if out.Providers, err = db.AgentProvider.Query().Count(ctx); err != nil {
		return nil, err
	}
	var mcps []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	if err := db.AgentMcpServer.Query().GroupBy(agentmcpserver.FieldStatus).Aggregate(gen.Count()).Scan(ctx, &mcps); err != nil {
		return nil, err
	}
	for _, g := range mcps {
		out.McpServers += g.Count
		if g.Status == "ok" {
			out.McpOK = g.Count
		}
	}
	var skills []struct {
		Enabled bool `json:"enabled"`
		Count   int  `json:"count"`
	}
	if err := db.AgentSkill.Query().GroupBy(agentskill.FieldEnabled).Aggregate(gen.Count()).Scan(ctx, &skills); err != nil {
		return nil, err
	}
	for _, g := range skills {
		out.Skills += g.Count
		if g.Enabled {
			out.SkillsEnabled = g.Count
		}
	}
	var runtimes []struct {
		Ready bool `json:"ready"`
		Count int  `json:"count"`
	}
	if err := db.AgentRuntime.Query().GroupBy(agentruntime.FieldReady).Aggregate(gen.Count()).Scan(ctx, &runtimes); err != nil {
		return nil, err
	}
	for _, g := range runtimes {
		out.Runtimes += g.Count
		if g.Ready {
			out.RuntimesReady = g.Count
		}
	}
	return out, nil
}

// AgentUsers 这些用户的名称和用户名（怎么显示由页面决定）。
func (r *agentRepo) AgentUsers(ctx context.Context, ids []string) ([]biz.AgentUser, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.data.db.User.Query().
		Where(user.IDIn(ids...)).
		Select(user.FieldID, user.FieldName, user.FieldUsername).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]biz.AgentUser, 0, len(rows))
	for _, u := range rows {
		out = append(out, biz.AgentUser{ID: u.ID, Name: u.Name, Username: u.Username})
	}
	return out, nil
}

func nonNilStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
