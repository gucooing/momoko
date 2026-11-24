package agentrun

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"momoko/pkg/acp"
)

// —— 向人提问（ACP 表单询问）——
//
// Agent 发 elicitation/create（form）：页面上出一张提问卡片，会话和等批准一样进入「等待」；
// 人答了（或跳过）就把结果按 ACP 的形状回给 Agent。和批准卡片共用 perms：一起计数、一起在停止时取消、一起被 Agent 撤回。

// Elicit Agent 要人填一张表。
func (s *Session) Elicit(reqID string, req *acp.CreateElicitationRequest, reply func(*acp.CreateElicitationResponse)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sideOf(req.SessionID) != nil {
		reply(&acp.CreateElicitationResponse{Action: acp.ElicitationDecline})
		return
	}
	// 和批准一样：v1 的只属于正在进行的一轮；v2 后台的工作也可能问，停下时由 Agent 自己撤回
	if !s.v2 && (s.turn == nil || s.turn.cancelled) {
		reply(&acp.CreateElicitationResponse{Action: acp.ElicitationCancel})
		return
	}
	q := questionOf(req)
	if len(q.Fields) == 0 {
		reply(&acp.CreateElicitationResponse{Action: acp.ElicitationDecline})
		return
	}
	sub := ""
	if req.SessionID != "" && req.SessionID != s.acpID {
		sub = req.SessionID
	}
	s.closeOpen(sub)
	// 和批准卡片一样放在主会话的对话里：子 Agent 问的也要人在主会话里看到
	item := s.newItem(ItemQuestion, "")
	item.Question = q
	s.perms[q.ID] = &pending{item: item, answer: reply, req: reqID}
	s.permReqs[reqID] = q.ID
	s.emit(item, true)
	s.sink.Questioned(q)
}

// Answer 回答一张提问卡片：action 是 accept（answers 是各字段填的值）或 decline（跳过，不回答）。
func (s *Session) Answer(questionID, action string, answers map[string][]string) error {
	s.mu.Lock()
	p := s.perms[questionID]
	if p == nil || p.item.Question == nil {
		s.mu.Unlock()
		return errors.New("这个问题已经回答过了，或者这一轮已经结束")
	}
	q := p.item.Question
	resp := &acp.CreateElicitationResponse{Action: action}
	switch action {
	case acp.ElicitationAccept:
		content, kept, err := answerContent(q.Fields, answers)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		resp.Content, q.Answers = content, kept
	case acp.ElicitationDecline:
	default:
		s.mu.Unlock()
		return errors.New("不认识的回答方式")
	}
	delete(s.perms, questionID)
	delete(s.permReqs, p.req)
	q.Outcome = action
	s.emit(p.item, true)
	waiting := len(s.perms) > 0
	s.mu.Unlock()

	p.answer(resp)
	s.sink.Answered(questionID, action, waiting)
	return nil
}

// questionOf 把 Agent 给的表换成页面上的提问卡片。
func questionOf(req *acp.CreateElicitationRequest) *Question {
	q := &Question{ID: uuid.NewString(), ToolID: req.ToolCallID, Message: req.Message}
	schema := req.RequestedSchema
	required := map[string]bool{}
	for _, key := range schema.Required {
		required[key] = true
	}
	kinds := map[string]string{}
	for _, f := range schema.Properties {
		field := QuestionField{Key: f.Key, Title: f.Title, Description: f.Description, Required: required[f.Key]}
		switch {
		case f.Type == "array":
			field.Kind = FieldMulti
			if f.Items != nil {
				field.Options = questionOptions(f.Items.AnyOf, f.Items.Enum, nil)
			}
		case len(f.OneOf) > 0 || len(f.Enum) > 0:
			field.Kind = FieldSingle
			field.Options = questionOptions(f.OneOf, f.Enum, f.EnumNames)
		case f.Type == "boolean":
			field.Kind = FieldBoolean
		case f.Type == "number":
			field.Kind = FieldNumber
		case f.Type == "integer":
			field.Kind = FieldInteger
		default:
			field.Kind = FieldText
		}
		// 紧跟在一个选择题后面、名字是它加 _custom 的文字格，是那道题的「其他」（Claude 的 AskUserQuestion 这样给）
		if base, ok := strings.CutSuffix(f.Key, "_custom"); ok && field.Kind == FieldText {
			if k := kinds[base]; k == FieldSingle || k == FieldMulti {
				field.For = base
			}
		}
		kinds[f.Key] = field.Kind
		q.Fields = append(q.Fields, field)
	}
	return q
}

func questionOptions(opts []acp.ElicitationOption, enum, names []string) []QuestionOption {
	out := make([]QuestionOption, 0, len(opts)+len(enum))
	for _, o := range opts {
		label := o.Title
		if label == "" {
			label = o.Const
		}
		out = append(out, QuestionOption{Value: o.Const, Label: label, Description: o.Description})
	}
	for i, v := range enum {
		label := v
		if i < len(names) && names[i] != "" {
			label = names[i]
		}
		out = append(out, QuestionOption{Value: v, Label: label})
	}
	return out
}

// answerContent 页面交来的回答按字段的种类换成 ACP 要的值；必填的没填、选了不在候选里的值、数字填错了都报错。
// 返回交给 Agent 的内容，和留在卡片上给人看的回答。
func answerContent(fields []QuestionField, answers map[string][]string) (map[string]any, map[string][]string, error) {
	content := map[string]any{}
	kept := map[string][]string{}
	for _, f := range fields {
		var vals []string
		for _, v := range answers[f.Key] {
			if v = strings.TrimSpace(v); v != "" {
				vals = append(vals, v)
			}
		}
		name := f.Title
		if name == "" {
			name = f.Key
		}
		if len(vals) == 0 {
			if f.Required {
				return nil, nil, errors.New("「" + name + "」还没填")
			}
			continue
		}
		if f.Kind == FieldSingle || f.Kind == FieldMulti {
			for _, v := range vals {
				if !hasOption(f.Options, v) {
					return nil, nil, errors.New("「" + name + "」没有这个选项")
				}
			}
		}
		if f.Kind != FieldMulti {
			vals = vals[:1]
		}
		switch f.Kind {
		case FieldMulti:
			content[f.Key] = vals
		case FieldNumber:
			n, err := strconv.ParseFloat(vals[0], 64)
			if err != nil {
				return nil, nil, errors.New("「" + name + "」要填数字")
			}
			content[f.Key] = n
		case FieldInteger:
			n, err := strconv.ParseInt(vals[0], 10, 64)
			if err != nil {
				return nil, nil, errors.New("「" + name + "」要填整数")
			}
			content[f.Key] = n
		case FieldBoolean:
			content[f.Key] = vals[0] == "true"
		default:
			content[f.Key] = vals[0]
		}
		kept[f.Key] = vals
	}
	return content, kept, nil
}

func hasOption(opts []QuestionOption, v string) bool {
	for _, o := range opts {
		if o.Value == v {
			return true
		}
	}
	return false
}

// markCancelled 卡片不用再回答了（这一轮停了，或者 Agent 撤回了）：收成「已取消」。
func (p *pending) markCancelled() {
	if p.item.Question != nil {
		p.item.Question.Outcome = "cancelled"
	} else if p.item.Permission != nil {
		p.item.Permission.Outcome = "cancelled"
	}
}

// replyCancelled 按「取消」回给 Agent。
func (p *pending) replyCancelled() {
	if p.answer != nil {
		p.answer(&acp.CreateElicitationResponse{Action: acp.ElicitationCancel})
		return
	}
	p.reply(acp.PermissionCancelled())
}
