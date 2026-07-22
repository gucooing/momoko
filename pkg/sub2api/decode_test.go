package sub2api

import (
	"testing"
	"time"
)

func TestDecodeUsageList_TypedEnvelope(t *testing.T) {
	body := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"items": [
				{
					"id": 42,
					"request_id": "req-abc",
					"model": "claude-sonnet-4",
					"inbound_endpoint": "/v1/messages",
					"group_id": 7,
					"group": {"id": 7, "name": "default"},
					"input_tokens": 100,
					"output_tokens": 50,
					"cache_creation_tokens": 10,
					"cache_read_tokens": 5,
					"total_cost": 0.012,
					"actual_cost": 0.01,
					"duration_ms": 1200,
					"first_token_ms": 200,
					"reasoning_effort": "high",
					"user_agent": "claude-cli",
					"created_at": "2026-07-14T10:00:00Z",
					"account": {"id": 3, "name": "acc-1"}
				}
			],
			"total": 1,
			"page": 1,
			"page_size": 100,
			"pages": 1
		}
	}`)

	result, err := decodeUsageList(body)
	if err != nil {
		t.Fatalf("decodeUsageList: %v", err)
	}
	if result.Total != 1 || len(result.Records) != 1 {
		t.Fatalf("unexpected size total=%d len=%d", result.Total, len(result.Records))
	}
	rec := result.Records[0]
	if rec.ID != "req-abc" {
		t.Errorf("id=%q", rec.ID)
	}
	if rec.Model != "claude-sonnet-4" {
		t.Errorf("model=%q", rec.Model)
	}
	if rec.Endpoint != "/v1/messages" {
		t.Errorf("endpoint=%q", rec.Endpoint)
	}
	if rec.GroupID != "7" {
		t.Errorf("groupID=%q", rec.GroupID)
	}
	if rec.GroupName != "default" {
		t.Errorf("group=%q", rec.GroupName)
	}
	if rec.AccountName != "acc-1" {
		t.Errorf("account=%q", rec.AccountName)
	}
	if rec.TokenCount != 165 {
		t.Errorf("tokenCount=%d", rec.TokenCount)
	}
	if rec.OutputTokens != 50 {
		t.Errorf("outputTokens=%d", rec.OutputTokens)
	}
	if rec.LatencyMS != 1200 {
		t.Errorf("latency=%d", rec.LatencyMS)
	}
	if rec.FirstTokenMS != 200 {
		t.Errorf("firstToken=%d", rec.FirstTokenMS)
	}
	// TPS = 50 / ((1200-200)/1000) = 50
	if rec.TPS != 50 {
		t.Errorf("tps=%v want 50", rec.TPS)
	}
	if rec.Cost != 0.01 {
		t.Errorf("cost=%v", rec.Cost)
	}
	if !rec.Success || rec.Status != "success" {
		t.Errorf("success/status=%v/%q", rec.Success, rec.Status)
	}
	if rec.ReasoningEffort != "high" {
		t.Errorf("reasoning=%q", rec.ReasoningEffort)
	}
	if rec.UserAgent != "claude-cli" {
		t.Errorf("ua=%q", rec.UserAgent)
	}
	wantTime, _ := time.Parse(time.RFC3339, "2026-07-14T10:00:00Z")
	if !rec.RequestTime.Equal(wantTime) {
		t.Errorf("time=%v", rec.RequestTime)
	}
}

func TestDecodeUpstreamErrorList_TypedEnvelope(t *testing.T) {
	body := []byte(`{
		"code": 0,
		"message": "success",
		"data": {
			"items": [
				{
					"id": 99,
					"created_at": "2026-07-14T11:00:00Z",
					"status_code": 529,
					"message": "overloaded",
					"request_id": "up-1",
					"account_name": "acc-x",
					"group_id": 2,
					"group_name": "premium",
					"upstream_endpoint": "/v1/responses",
					"upstream_model": "gpt-5",
					"upstream_latency_ms": 800
				}
			],
			"total": 1,
			"page": 1,
			"page_size": 50,
			"pages": 1
		}
	}`)

	result, err := decodeUpstreamErrorList(body)
	if err != nil {
		t.Fatalf("decodeUpstreamErrorList: %v", err)
	}
	if len(result.Records) != 1 {
		t.Fatalf("len=%d", len(result.Records))
	}
	rec := result.Records[0]
	if rec.ID != "upstream:99" {
		t.Errorf("id=%q", rec.ID)
	}
	if rec.Status != StatusUpstreamError || rec.Success {
		t.Errorf("status/success=%q/%v", rec.Status, rec.Success)
	}
	if rec.GroupID != "2" {
		t.Errorf("groupID=%q", rec.GroupID)
	}
	if rec.GroupName != "premium" {
		t.Errorf("group=%q", rec.GroupName)
	}
	if rec.Model != "gpt-5" {
		t.Errorf("model=%q", rec.Model)
	}
	if rec.Endpoint != "/v1/responses" {
		t.Errorf("endpoint=%q", rec.Endpoint)
	}
	if rec.HTTPStatus != 529 {
		t.Errorf("http=%d", rec.HTTPStatus)
	}
	if rec.ErrorMessage != "overloaded" {
		t.Errorf("err=%q", rec.ErrorMessage)
	}
	if rec.LatencyMS != 800 {
		t.Errorf("latency=%d", rec.LatencyMS)
	}
}

func TestPerRequestTPS(t *testing.T) {
	cases := []struct {
		name                 string
		output, dur, firstMS int64
		want                 float64
	}{
		{name: "gen = duration - first", output: 50, dur: 1200, firstMS: 200, want: 50},
		{name: "no first token falls back to full duration", output: 100, dur: 2000, firstMS: 0, want: 50},
		{name: "first >= duration", output: 100, dur: 500, firstMS: 500, want: 0},
		{name: "first > duration", output: 100, dur: 400, firstMS: 500, want: 0},
		{name: "zero output", output: 0, dur: 1000, firstMS: 100, want: 0},
		{name: "zero duration", output: 100, dur: 0, firstMS: 0, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := perRequestTPS(tc.output, tc.dur, tc.firstMS)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestDecodeUsageList_ErrorCode(t *testing.T) {
	body := []byte(`{"code":401,"message":"unauthorized","data":null}`)
	_, err := decodeUsageList(body)
	if err == nil || err.Error() != "unauthorized" {
		t.Fatalf("err=%v", err)
	}
}
