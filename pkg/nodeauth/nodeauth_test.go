package nodeauth

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc/metadata"
)

const testSeed = "mk-0123456789abcdefghijklmnopqrstuv"

func signedFor(t *testing.T, ctx context.Context, method string, now time.Time) *Signed {
	t.Helper()
	pairs, err := SignedMetadata(ctx, testSeed, method, now)
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseMetadata(metadata.Pairs(pairs...))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestKeyIDIsStableAndHidesSeed(t *testing.T) {
	a, b := KeyID(testSeed), KeyID(testSeed)
	if a != b {
		t.Fatalf("key id must be deterministic: %s != %s", a, b)
	}
	if KeyID(testSeed+"x") == a {
		t.Fatal("different seeds must not share a key id")
	}
	if len(a) != len(keyIDPrefix)+24 {
		t.Fatalf("unexpected key id length: %s", a)
	}
}

func TestVerifyRoundTrip(t *testing.T) {
	now := time.Now()
	ctx := WithOperator(context.Background(), "管理员 admin")
	s := signedFor(t, ctx, "/v1.System/SystemStatus", now)
	if s.KeyID != KeyID(testSeed) {
		t.Fatal("metadata must carry the derived key id")
	}
	if err := s.Verify(testSeed, "/v1.System/SystemStatus", now); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if s.Operator() != "管理员 admin" {
		t.Fatalf("operator = %q", s.Operator())
	}
}

func TestVerifyRejectsTampering(t *testing.T) {
	now := time.Now()
	s := signedFor(t, context.Background(), "/v1.System/SystemStatus", now)

	if err := s.Verify(testSeed, "/v1.FileManager/BatchDeleteFileSystem", now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("method swap must fail, got %v", err)
	}
	if err := s.Verify("mk-wrong-seed", "/v1.System/SystemStatus", now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("wrong seed must fail, got %v", err)
	}
	forged := *s
	forged.operator = "someone-else"
	if err := forged.Verify(testSeed, "/v1.System/SystemStatus", now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("operator tampering must fail, got %v", err)
	}
	shifted := *s
	shifted.Timestamp++
	if err := shifted.Verify(testSeed, "/v1.System/SystemStatus", now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("timestamp tampering must fail, got %v", err)
	}
}

func TestVerifyClockSkew(t *testing.T) {
	now := time.Now()
	s := signedFor(t, context.Background(), "/v1.System/SystemStatus", now)
	if err := s.Verify(testSeed, "/v1.System/SystemStatus", now.Add(MaxClockSkew-time.Second)); err != nil {
		t.Fatalf("within skew must pass: %v", err)
	}
	if err := s.Verify(testSeed, "/v1.System/SystemStatus", now.Add(MaxClockSkew+time.Second)); !errors.Is(err, ErrClockSkew) {
		t.Fatalf("expired signature must fail, got %v", err)
	}
	if err := s.Verify(testSeed, "/v1.System/SystemStatus", now.Add(-MaxClockSkew-time.Second)); !errors.Is(err, ErrClockSkew) {
		t.Fatalf("future signature must fail, got %v", err)
	}
}

func TestParseMetadataMissingFields(t *testing.T) {
	if _, err := ParseMetadata(metadata.Pairs(MetadataKeyID, "mkid-x")); !errors.Is(err, ErrMissingCredential) {
		t.Fatalf("missing fields must fail, got %v", err)
	}
}

func TestReplayGuard(t *testing.T) {
	g := NewReplayGuard()
	now := time.Now()
	if err := g.Use("k", "n1", now); err != nil {
		t.Fatal(err)
	}
	if err := g.Use("k", "n1", now.Add(time.Second)); !errors.Is(err, ErrReplay) {
		t.Fatalf("reused nonce must fail, got %v", err)
	}
	if err := g.Use("k2", "n1", now); err != nil {
		t.Fatalf("same nonce under another key is independent: %v", err)
	}
	if err := g.Use("k", "n1", now.Add(2*MaxClockSkew+2*time.Minute)); err != nil {
		t.Fatalf("expired record must be forgotten: %v", err)
	}
}
