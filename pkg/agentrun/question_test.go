package agentrun

import (
	"encoding/json"
	"reflect"
	"testing"

	"momoko/pkg/acp"
)

// claude-agent-acp 把 AskUserQuestion（两个问题：一个单选、一个多选）发成的 elicitation/create
const askUserQuestion = `{
  "mode": "form", "sessionId": "s1", "toolCallId": "toolu_1",
  "message": "Please answer the following questions.",
  "requestedSchema": {"type": "object", "properties": {
    "question_0": {"type": "string", "title": "方案", "description": "用哪个方案？",
      "oneOf": [{"const": "A", "title": "A", "description": "快"}, {"const": "B", "title": "B"}]},
    "question_0_custom": {"type": "string", "title": "Other", "description": "Type your own answer"},
    "question_1": {"type": "array", "title": "范围", "description": "改哪些？",
      "items": {"anyOf": [{"const": "前端", "title": "前端"}, {"const": "后端", "title": "后端"}]}},
    "question_1_custom": {"type": "string", "title": "Other"},
    "count": {"type": "integer", "title": "几个"}
  }, "required": ["count"]}
}`

func TestQuestionOf(t *testing.T) {
	req := &acp.CreateElicitationRequest{}
	if err := json.Unmarshal([]byte(askUserQuestion), req); err != nil {
		t.Fatal(err)
	}
	q := questionOf(req)
	var keys, kinds, fors []string
	for _, f := range q.Fields {
		keys, kinds, fors = append(keys, f.Key), append(kinds, f.Kind), append(fors, f.For)
	}
	if want := []string{"question_0", "question_0_custom", "question_1", "question_1_custom", "count"}; !reflect.DeepEqual(keys, want) {
		t.Fatalf("keys = %v, want %v（要保留 Agent 给的顺序）", keys, want)
	}
	if want := []string{FieldSingle, FieldText, FieldMulti, FieldText, FieldInteger}; !reflect.DeepEqual(kinds, want) {
		t.Fatalf("kinds = %v, want %v", kinds, want)
	}
	if want := []string{"", "question_0", "", "question_1", ""}; !reflect.DeepEqual(fors, want) {
		t.Fatalf("for = %v, want %v", fors, want)
	}
	if q.ToolID != "toolu_1" || q.Fields[0].Options[0].Description != "快" || !q.Fields[4].Required {
		t.Fatalf("question = %+v", q)
	}

	content, kept, err := answerContent(q.Fields, map[string][]string{
		"question_0": {"B"}, "question_1": {"前端", "后端"}, "question_1_custom": {"  还有文档 "}, "count": {"3"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"question_0": "B", "question_1": []string{"前端", "后端"}, "question_1_custom": "还有文档", "count": int64(3)}
	if !reflect.DeepEqual(content, want) {
		t.Fatalf("content = %#v, want %#v", content, want)
	}
	if kept["question_1_custom"][0] != "还有文档" {
		t.Fatalf("kept = %v", kept)
	}

	if _, _, err := answerContent(q.Fields, map[string][]string{"question_0": {"C"}, "count": {"1"}}); err == nil {
		t.Fatal("不在候选里的选项要报错")
	}
	if _, _, err := answerContent(q.Fields, map[string][]string{"question_0": {"A"}}); err == nil {
		t.Fatal("必填的没填要报错")
	}
	if _, _, err := answerContent(q.Fields, map[string][]string{"count": {"三"}}); err == nil {
		t.Fatal("整数填错要报错")
	}
}
