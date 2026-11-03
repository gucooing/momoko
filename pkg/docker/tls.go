package docker

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"
	"time"

	registrytypes "github.com/docker/docker/api/types/registry"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

// buildTLSConfig 用配置里保存的 PEM 内容（CA / 客户端证书 / 客户端私钥）构造双向 TLS 配置。
func buildTLSConfig(cfg *v1.DockerConfigInfo) (*tls.Config, error) {
	if strings.TrimSpace(cfg.TlsCert) == "" || strings.TrimSpace(cfg.TlsKey) == "" {
		return nil, errors.New("docker TLS 需要客户端证书和客户端私钥")
	}
	if strings.TrimSpace(cfg.TlsCa) == "" && !cfg.TlsSkipVerify {
		return nil, errors.New("docker TLS 需要 CA 证书（或开启跳过证书校验）")
	}
	pair, err := tls.X509KeyPair([]byte(cfg.TlsCert), []byte(cfg.TlsKey))
	if err != nil {
		return nil, fmt.Errorf("docker TLS 客户端证书与私钥无效: %w", err)
	}
	tlsCfg := &tls.Config{
		Certificates:       []tls.Certificate{pair},
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: cfg.TlsSkipVerify, //nolint:gosec // 用户显式开启，仅用于调试自签名证书
	}
	if strings.TrimSpace(cfg.TlsCa) != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(cfg.TlsCa)) {
			return nil, errors.New("docker TLS CA 证书无效")
		}
		tlsCfg.RootCAs = pool
	}
	return tlsCfg, nil
}

// TLSCertMeta 解析证书 PEM 的展示信息（CN、有效期、算法）。
func TLSCertMeta(content string) *v1.DockerTLSFileMeta {
	if strings.TrimSpace(content) == "" {
		return &v1.DockerTLSFileMeta{}
	}
	meta := &v1.DockerTLSFileMeta{Present: true, Size: int64(len(content))}
	block, _ := pem.Decode([]byte(content))
	if block == nil {
		meta.Error = "不是有效的 PEM 文件"
		return meta
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		meta.Error = "证书解析失败"
		return meta
	}
	meta.Subject = cert.Subject.CommonName
	if meta.Subject == "" {
		meta.Subject = cert.Subject.String()
	}
	meta.NotAfter = timestamppb.New(cert.NotAfter)
	meta.Algorithm = publicKeyAlgorithm(cert.PublicKey)
	return meta
}

// TLSKeyMeta 解析私钥 PEM 的展示信息（大小、算法），不返回内容本身。
func TLSKeyMeta(content string) *v1.DockerTLSFileMeta {
	if strings.TrimSpace(content) == "" {
		return &v1.DockerTLSFileMeta{}
	}
	meta := &v1.DockerTLSFileMeta{Present: true, Size: int64(len(content))}
	block, _ := pem.Decode([]byte(content))
	if block == nil {
		meta.Error = "不是有效的 PEM 文件"
		return meta
	}
	var key any
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "EC PRIVATE KEY":
		key, err = x509.ParseECPrivateKey(block.Bytes)
	default:
		key, err = x509.ParsePKCS8PrivateKey(block.Bytes)
	}
	if err != nil {
		meta.Error = "私钥解析失败"
		return meta
	}
	switch k := key.(type) {
	case *rsa.PrivateKey:
		meta.Algorithm = publicKeyAlgorithm(&k.PublicKey)
	case *ecdsa.PrivateKey:
		meta.Algorithm = publicKeyAlgorithm(&k.PublicKey)
	case ed25519.PrivateKey:
		meta.Algorithm = "Ed25519"
	}
	return meta
}

func publicKeyAlgorithm(pub any) string {
	switch k := pub.(type) {
	case *rsa.PublicKey:
		return fmt.Sprintf("RSA %d", k.N.BitLen())
	case *ecdsa.PublicKey:
		return "ECDSA " + k.Curve.Params().Name
	case ed25519.PublicKey:
		return "Ed25519"
	default:
		return ""
	}
}

// RegistryLogin 通过引擎登录仓库验证凭据；返回带验证结果的凭据（不含密码 / token）。
func (m *Manager) RegistryLogin(ctx context.Context, auth *v1.DockerRegistryAuth) (*v1.DockerRegistryAuth, error) {
	if auth == nil || strings.TrimSpace(auth.ServerAddress) == "" {
		return nil, errors.New("仓库地址不能为空")
	}
	cli, err := m.getClient()
	if err != nil {
		return nil, err
	}
	ctx, cancel := m.withTimeout(ctx)
	defer cancel()
	result := &v1.DockerRegistryAuth{
		ServerAddress: strings.TrimSpace(auth.ServerAddress),
		Username:      auth.Username,
		VerifiedAt:    timestamppb.New(time.Now()),
	}
	_, err = cli.RegistryLogin(ctx, registrytypes.AuthConfig{
		Username:      auth.Username,
		Password:      auth.Password,
		ServerAddress: result.ServerAddress,
		IdentityToken: auth.Token,
	})
	if err != nil {
		result.VerifyError = err.Error()
		m.recordEvent("warn", "registry_failed", result.ServerAddress+" · "+err.Error())
		return result, nil
	}
	result.Verified = true
	m.recordEvent("ok", "registry_ok", result.ServerAddress)
	return result, nil
}
