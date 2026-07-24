package auth

import (
	"testing"

	"momoko/internal/data/ent/gen"
)

func TestVerifyPasswordBcrypt(t *testing.T) {
	hash, err := HashPassword("s3cret-pw")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("s3cret-pw", hash) {
		t.Fatal("bcrypt hash should verify")
	}
	if VerifyPassword("wrong", hash) {
		t.Fatal("wrong password must not verify")
	}
	if VerifyPassword("s3cret-pw", "") {
		t.Fatal("empty stored hash must not verify")
	}
}

func TestVerifyPasswordRejectsNonBcrypt(t *testing.T) {
	if VerifyPassword("123456", "e10adc3949ba59abbe56e057f20f883e") {
		t.Fatal("legacy md5 must not verify")
	}
}

func TestValidateSecret(t *testing.T) {
	for _, s := range []string{"", "short"} {
		if err := ValidateSecret(s); err == nil {
			t.Errorf("secret %q should be rejected", s)
		}
	}
	for _, s := range []string{"12345678", "a-strong-random-secret-32chars-xx"} {
		if err := ValidateSecret(s); err != nil {
			t.Errorf("secret %q should be accepted: %v", s, err)
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
		t.Fatal("jwt signing key must differ from the raw secret")
	}
}

func TestTokenRoundTripWithNoise(t *testing.T) {
	old := AuthSecretKey
	AuthSecretKey = "a-strong-random-secret-32chars-xx"
	defer func() { AuthSecretKey = old }()

	accessNoise, err := NewNoise()
	if err != nil {
		t.Fatal(err)
	}
	refreshNoise, err := NewNoise()
	if err != nil {
		t.Fatal(err)
	}
	session := &gen.Auth{
		UserID:       "u1",
		DeviceID:     "d1",
		SessionID:    "s1",
		AccessNoise:  accessNoise,
		RefreshNoise: refreshNoise,
	}

	at, err := GenerateAccessToken(session)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := GenerateRefreshToken(session)
	if err != nil {
		t.Fatal(err)
	}
	if at == rt {
		t.Fatal("access and refresh tokens must differ")
	}

	ac, err := ParseToken(at)
	if err != nil {
		t.Fatalf("parse access: %v", err)
	}
	if ac.Kind != TokenKindAccess || ac.Noise != accessNoise || ac.SessionID != "s1" {
		t.Fatalf("access claims mismatch: %+v", ac)
	}
	rc, err := ParseToken(rt)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	if rc.Kind != TokenKindRefresh || rc.Noise != refreshNoise {
		t.Fatalf("refresh claims mismatch: %+v", rc)
	}

	// 改噪声后旧 token 的签名仍可解析，但 noise 值会与库不一致（业务层校验）；
	// 改密钥后必须直接解析失败。
	AuthSecretKey = "another-strong-secret-value-xxxxx"
	if _, err := ParseToken(at); err == nil {
		t.Fatal("token signed under a different secret must not verify")
	}
}
