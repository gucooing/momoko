package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const redactedValue = "***"

var sensitiveKeys = map[string]struct{}{
	"accesstoken":   {},
	"authorization": {},
	"credential":    {},
	"newpassword":   {},
	"oldpassword":   {},
	"passphrase":    {},
	"password":      {},
	"refreshtoken":  {},
	"secret":        {},
	"token":         {},
}

type Detail struct {
	Method     string `json:"method,omitempty"`
	Path       string `json:"path,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Request    any    `json:"request,omitempty"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms"`
}

func (detail *Detail) MarshalDetail() string {
	detail.Request = NormalizeRequest(detail.Request)
	body, err := json.Marshal(detail)
	if err != nil {
		return "{}"
	}
	return string(body)
}

func NormalizeRequest(req any) any {
	if req == nil {
		return nil
	}

	body, err := marshalRequest(req)
	if err != nil {
		return fmt.Sprintf("%T", req)
	}

	var data any
	if err = json.Unmarshal(body, &data); err != nil {
		return string(body)
	}
	return redact(data)
}

func RequestString(detail string, keys ...string) string {
	var data map[string]any
	if err := json.Unmarshal([]byte(detail), &data); err != nil {
		return ""
	}

	req, ok := data["request"].(map[string]any)
	if !ok {
		return ""
	}

	for _, key := range keys {
		if value, ok := req[key]; ok {
			if s, ok := value.(string); ok {
				return s
			}
		}
	}
	return ""
}

func marshalRequest(req any) ([]byte, error) {
	if msg, ok := req.(proto.Message); ok {
		return protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(msg)
	}
	return json.Marshal(req)
}

func redact(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if isSensitiveKey(key) {
				v[key] = redactedValue
				continue
			}
			v[key] = redact(item)
		}
		return v
	case []any:
		for i, item := range v {
			v[i] = redact(item)
		}
		return v
	default:
		return v
	}
}

func isSensitiveKey(key string) bool {
	normalized := strings.NewReplacer("_", "", "-", "", ".", "").Replace(strings.ToLower(key))
	_, ok := sensitiveKeys[normalized]
	return ok
}
