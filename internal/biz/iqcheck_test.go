package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/iqcheckresult"
	"momoko/internal/data/ent/gen/iqchecksite"
	"momoko/internal/data/ent/schema"
)

type checkTestRepo struct {
	IQCheckRepo
	t       *testing.T
	site    *gen.IQCheckSite
	saved   chan *gen.IQCheckResult
	marked  atomic.Int32
	counted atomic.Int32
}

func (r *checkTestRepo) GetSite(context.Context, string) (*gen.IQCheckSite, error) {
	return r.site, nil
}
func (r *checkTestRepo) ListEnabledSites(context.Context) ([]*gen.IQCheckSite, error) {
	return []*gen.IQCheckSite{r.site}, nil
}
func (r *checkTestRepo) ListEnabledPrompts(ctx context.Context) ([]*gen.IQCheckPrompt, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return []*gen.IQCheckPrompt{{ID: "prompt", Title: "test", Content: "hello"}}, nil
}
func (r *checkTestRepo) MarkSiteChecked(ctx context.Context, _ string, _ time.Time, countRun bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.marked.Add(1)
	if countRun {
		r.counted.Add(1)
	}
	return nil
}
func (r *checkTestRepo) CreateResults(ctx context.Context, rows []*gen.IQCheckResult) error {
	if err := ctx.Err(); err != nil {
		r.t.Errorf("saving with canceled context: %v", err)
		return err
	}
	deadline, ok := ctx.Deadline()
	if !ok || time.Until(deadline) > iqCheckStoreTimeout {
		r.t.Error("saving must have a bounded deadline")
	}
	for _, row := range rows {
		r.saved <- row
	}
	return nil
}

func waitCheckResult(t *testing.T, saved <-chan *gen.IQCheckResult) *gen.IQCheckResult {
	t.Helper()
	select {
	case row := <-saved:
		return row
	case <-time.After(3 * time.Second):
		t.Fatal("result was not saved")
		return nil
	}
}

func waitCheckUnlocked(t *testing.T, uc *IQCheckUsecase, siteID string) {
	t.Helper()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	for uc.isRunning(siteID) {
		select {
		case <-timer.C:
			t.Fatal("round did not release the running lock")
		case <-ticker.C:
		}
	}
}

func TestIQCheckProgressCancellationAndUnlock(t *testing.T) {
	var requests atomic.Int32
	slowStarted := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		requests.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		switch body.Model {
		case "slow":
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
			w.(http.Flusher).Flush()
			close(slowStarted)
			<-r.Context().Done()
		case "fast":
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2}}\n\ndata: [DONE]\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "broken":
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"truncated\"}}]}\n\n")
		}
	}))
	t.Cleanup(srv.Close)
	repo := &checkTestRepo{t: t, site: &gen.IQCheckSite{ID: "site", Name: "test", Enabled: true, Endpoint: srv.URL, APIKey: "k", Protocol: iqchecksite.ProtocolChat, Models: []schema.IQCheckModel{{Model: "slow"}, {Model: "fast"}, {Model: "broken"}}}, saved: make(chan *gen.IQCheckResult, 3)}
	uc := &IQCheckUsecase{repo: repo, running: make(map[string]struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	returned := make(chan error, 1)
	go func() { returned <- uc.runDueSites(ctx) }()
	select {
	case err := <-returned:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("scheduler waited for the slow gateway")
	}
	select {
	case <-slowStarted:
	case <-time.After(time.Second):
		t.Fatal("slow model did not start")
	}
	rows := map[string]*gen.IQCheckResult{}
	// Fast and failed models must both become visible before the slow one ends.
	for range 2 {
		row := waitCheckResult(t, repo.saved)
		rows[row.Model] = row
	}
	if row := rows["fast"]; row == nil || row.Status != iqcheckresult.StatusSuccess || row.ResponseText != "ok" || row.InputTokens != 3 || row.OutputTokens != 2 {
		t.Fatalf("fast result: %+v", row)
	}
	if row := rows["broken"]; row == nil || row.Status != iqcheckresult.StatusFailed || row.ResponseText != "truncated" || row.Error == "" {
		t.Fatalf("broken result: %+v", row)
	}
	if err := uc.runDueSites(ctx); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 3 || repo.marked.Load() != 1 || repo.counted.Load() != 1 {
		t.Fatal("scheduled the same round twice")
	}
	cancel()
	row := waitCheckResult(t, repo.saved)
	if row.Model != "slow" || row.Status != iqcheckresult.StatusFailed || row.ResponseText != "partial" || !strings.Contains(row.Error, "context canceled") {
		t.Fatalf("canceled result: %+v", row)
	}
	if row.RoundID != rows["fast"].RoundID || row.PromptContent != rows["fast"].PromptContent {
		t.Fatal("models did not share a round and prompt")
	}
	waitCheckUnlocked(t, uc, repo.site.ID)
}

func TestManualIQCheckSurvivesRequestCancellation(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(srv.Close)
	repo := &checkTestRepo{t: t, site: &gen.IQCheckSite{ID: "manual", Endpoint: srv.URL, APIKey: "k", Protocol: iqchecksite.ProtocolChat, Models: []schema.IQCheckModel{{Model: "m"}}}, saved: make(chan *gen.IQCheckResult, 1)}
	uc := &IQCheckUsecase{repo: repo, running: make(map[string]struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	count, err := uc.TriggerCheck(ctx, repo.site.ID)
	cancel()
	close(release)
	if err != nil || count != 1 {
		t.Fatalf("count=%d error=%v", count, err)
	}
	row := waitCheckResult(t, repo.saved)
	if row.Status != iqcheckresult.StatusSuccess || row.ResponseText != "ok" || repo.counted.Load() != 0 {
		t.Fatalf("manual result: %+v", row)
	}
	waitCheckUnlocked(t, uc, repo.site.ID)
}
