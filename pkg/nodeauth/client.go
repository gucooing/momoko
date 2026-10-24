package nodeauth

import (
	"context"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// Credential 是主控持有的签名种子，为每次 gRPC 调用附加一次性签名。
type Credential struct {
	seed string
}

func NewCredential(seed string) *Credential {
	return &Credential{seed: seed}
}

// KeyID 返回种子对应的公开密钥标识。
func (c *Credential) KeyID() string {
	return KeyID(c.seed)
}

func (c *Credential) sign(ctx context.Context, method string) (context.Context, error) {
	pairs, err := SignedMetadata(ctx, c.seed, method, time.Now())
	if err != nil {
		return nil, err
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...), nil
}

// UnaryClientInterceptor 为一元调用签名。
func (c *Credential) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, err := c.sign(ctx, method)
		if err != nil {
			return err
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientInterceptor 为流式调用签名（签名在建流时校验一次）。
func (c *Credential) StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, err := c.sign(ctx, method)
		if err != nil {
			return nil, err
		}
		return streamer(ctx, desc, cc, method, opts...)
	}
}
