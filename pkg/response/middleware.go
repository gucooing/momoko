package response

import (
	"context"
	stdjson "encoding/json"
	stdhttp "net/http"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
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
			if _, ok := unwrapEnvelope(reply); ok {
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

// ResponseEncoder encodes successful responses as unified JSON.
func ResponseEncoder(w stdhttp.ResponseWriter, r *stdhttp.Request, v any) error {
	if v == nil {
		return nil
	}
	envelope, ok := unwrapEnvelope(v)
	if !ok {
		return khttp.DefaultResponseEncoder(w, r, v)
	}
	body, err := marshalEnvelope(envelope)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	_, err = w.Write(body)
	return err
}

// ErrorEncoder encodes errors as unified JSON.
func ErrorEncoder(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
	se := kerrors.FromError(err)
	message := se.Message
	if message == "" {
		message = stdhttp.StatusText(int(se.Code))
		if message == "" {
			message = "error"
		}
	}
	body, marshalErr := marshalEnvelope(Envelope{
		Code:    int(se.Code),
		Message: message,
		Data:    emptyData(),
	})
	if marshalErr != nil {
		w.WriteHeader(stdhttp.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(int(se.Code))
	_, _ = w.Write(body)
}

// WriteError writes a unified error response and is useful in HTTP filters.
func WriteError(w stdhttp.ResponseWriter, r *stdhttp.Request, err error) {
	ErrorEncoder(w, r, err)
}

func unwrapEnvelope(v any) (Envelope, bool) {
	switch env := v.(type) {
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
