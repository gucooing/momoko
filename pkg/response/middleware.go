package response

import (
	"context"
	stdjson "encoding/json"
	"errors"
	stdhttp "net/http"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const (
	defaultSuccessCode    = stdhttp.StatusOK
	defaultSuccessMessage = "success"
	contentTypeJSON       = "application/json"
)

var protoJSONMarshalOptions = protojson.MarshalOptions{
	EmitUnpopulated: true,
}

// Envelope is the unified HTTP response body.
type Envelope struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// Middleware wraps successful replies with a unified response envelope.
func Middleware() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (any, error) {
			reply, err := handler(ctx, req)
			if err != nil {
				return nil, err
			}
			if reply, ok := unwrapEnvelope(reply); ok {
				return reply, nil
			}
			return Envelope{
				Code:    defaultSuccessCode,
				Message: defaultSuccessMessage,
				Data:    normalizeData(reply),
			}, nil
		}
	}
}

// Error 接口错误。状态码只有两种：业务错误一律 400，凭证失效（token 无效或过期、签名不对）为 401；
// 404、405 等留给框架（路由不存在等），不和业务错误混用。
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e Error) Error() string {
	return e.Message
}

// BadRequest 业务错误（HTTP 400）。
func BadRequest(message string) error {
	return Error{stdhttp.StatusBadRequest, message}
}

// Unauthorized 凭证失效（HTTP 401）：前端据此刷新登录或重新登录。
func Unauthorized(message string) error {
	return Error{stdhttp.StatusUnauthorized, message}
}

// ResponseEncoder encodes successful responses as unified JSON.
func ResponseEncoder(w stdhttp.ResponseWriter, r *stdhttp.Request, v any) error {
	envelope, ok := unwrapEnvelope(v)
	if !ok {
		envelope = Envelope{
			Code:    defaultSuccessCode,
			Message: defaultSuccessMessage,
			Data:    normalizeData(v),
		}
	}
	body, err := marshalEnvelope(envelope)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	_, err = w.Write(body)
	return err
}

// ErrorEncoder encodes errors as unified JSON. 业务代码返回的普通 error 按业务错误（400）处理；
// 框架自己的错误（路由不存在 404、参数校验 400、panic 恢复 500 等）保留原状态码。
func ErrorEncoder(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
	reply, ok := unwrapEnvelope(err)
	if !ok {
		var ke *kerrors.Error
		if !errors.As(err, &ke) {
			reply, ok = unwrapEnvelope(BadRequest(err.Error()))
		}
	}
	if !ok {
		se := kerrors.FromError(err)
		message := se.Message
		if message == "" {
			message = stdhttp.StatusText(int(se.Code))
			if message == "" {
				message = "error"
			}
		}
		reply = Envelope{
			Code:    int(se.Code),
			Message: message,
			Data:    emptyData(),
		}
	}

	body, marshalErr := marshalEnvelope(reply)
	if marshalErr != nil {
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(reply.Code)
	_, _ = w.Write(body)
}

// WriteError writes a unified error response and is useful in HTTP filters.
func WriteError(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
	ErrorEncoder(w, r, err)
}

func Write(w stdhttp.ResponseWriter, r *stdhttp.Request, body []byte) {
	w.WriteHeader(stdhttp.StatusOK)
	_, _ = w.Write(body)
}

func unwrapEnvelope(v any) (Envelope, bool) {
	switch env := v.(type) {
	case Error:
		return Envelope{
			Code:    env.Code,
			Message: env.Message,
			Data:    nil,
		}, true
	case Envelope:
		return env, true
	case *Envelope:
		if env == nil {
			return Envelope{}, false
		}
		return *env, true
	default:
		return Envelope{}, false
	}
}

func marshalEnvelope(envelope Envelope) ([]byte, error) {
	rawData, err := marshalData(normalizeData(envelope.Data))
	if err != nil {
		return nil, err
	}
	payload := struct {
		Code    int                `json:"code"`
		Message string             `json:"message"`
		Data    stdjson.RawMessage `json:"data"`
	}{
		Code:    envelope.Code,
		Message: envelope.Message,
		Data:    stdjson.RawMessage(rawData),
	}
	return stdjson.Marshal(payload)
}

func normalizeData(data any) any {
	if data == nil {
		return emptyData()
	}
	return data
}

func marshalData(data any) ([]byte, error) {
	switch value := data.(type) {
	case stdjson.Marshaler:
		return value.MarshalJSON()
	case proto.Message:
		return protoJSONMarshalOptions.Marshal(value)
	default:
		return stdjson.Marshal(value)
	}
}

func emptyData() map[string]any {
	return map[string]any{}
}
