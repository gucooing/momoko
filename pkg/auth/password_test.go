package auth

import (
	"testing"
	"time"

	"momoko/internal/data/ent/gen"
)

func TestVerifyPasswordBcrypt(t *testing.T) {
	hash, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if ok, upgrade := VerifyPassword("s3cret-pw", hash); !ok || upgrade {
		t.Fatalf("want ok && !upgrade, got ok=%v upgrade=%v", ok, upgrade)
	}
	if ok, _ := VerifyPassword("wrong", hash); ok {
		t.Fatal("wrong password must not verify")
	}
}

func TestVerifyPasswordLegacyMD5Upgrade(t *testing.T) {
	legacy := legacyEncodeMD5("oldpass")
	ok, upgrade := VerifyPassword("oldpass", legacy)
	if !ok || !upgrade {
		t.Fatalf("legacy md5 must verify with upgrade=true, got ok=%v upgrade=%v", ok, upgrade)
	}
	if ok, _ := VerifyPassword("nope", legacy); ok {
		t.Fatal("wrong legacy password must not verify")
	}
}

func TestValidateSecret(t *testing.T) {
	for _, s := range []string{"", "123456", "gucooing.auth", "GUCOOING.AUTH", "secret", "short"} {
		if err := ValidateSecret(s); err == nil {
			t.Errorf("weak secret %q should be rejected", s)
		}
	}
	for _, s := range []string{"a-strong-random-secret-32chars-xx", "9f8e7d6c5b4a3210ffee"} {
		if err := ValidateSecret(s); err != nil {
			t.Errorf("strong secret %q should be accepted: %v", s, err)
		}
	}
}

func TestJWTSigningKeySeparation(t *testing.T) {
	old := AuthSecretKey
	AuthSecretKey = "a-strong-random-secret-32chars-xx"
	defer func() { AuthSecretKey = old }()

	key := jwtSigningKey()
	if len(key) != 32 {
		t.Fatalf("derived key len = %d, want 32", len(key))
	}
	if string(key) == AuthSecretKey {
		t.Fatal("jwt signing key must differ from the raw secret (which is the secretbox key material)")
	}
}

func TestTokenRoundTripWithDerivedKey(t *testing.T) {
	old := AuthSecretKey
	AuthSecretKey = "a-strong-random-secret-32chars-xx"
	defer func() { AuthSecretKey = old }()

	now := time.Now()
	tok, err := GenerateToken(&gen.Auth{
		UserID:     "u1",
		DeviceID:   "d1",
		SessionID:  "s1",
		CreateTime: now,
		UpdateTime: now,
		ExpiresAt:  now.Add(time.Hour),
	})
	if err != nil {
		t.Fatal(err)
	}
	claims, err := ParseToken(tok)
	if err != nil {
		t.Fatalf("parse with derived key failed: %v", err)
	}
	if claims.UserID != "u1" || claims.DeviceId != "d1" || claims.SessionID != "s1" {
		t.Fatalf("claims mismatch: %+v", claims)
	}

	// 用旧的“原始密钥直签”方式签发的 token 必须无法通过校验（确认密钥确实切换了）。
	AuthSecretKey = "another-strong-secret-value-xxxxx"
	if _, err := ParseToken(tok); err == nil {
		t.Fatal("token signed under a different secret must not verify")
	}
}
