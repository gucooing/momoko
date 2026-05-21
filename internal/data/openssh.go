package data

import (
	"context"
	"strings"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/sshhost"
	"momoko/internal/data/ent/gen/user"
	authpkg "momoko/pkg/auth"
	"momoko/pkg/secretbox"
	"momoko/pkg/sshcore"
)

type openSSHRepo struct {
	data *Data
	box  *secretbox.Box
}

func NewOpenSSHRepo(data *Data) biz.OpenSSHRepo {
	return &openSSHRepo{
		data: data,
		box:  secretbox.New(authpkg.AuthSecretKey),
	}
}

func (o *openSSHRepo) GetSSHHosts(ctx context.Context, page, pageSize int64, userID string, keywords, host *string) ([]*gen.SSHHost, int64, error) {
	query := o.accessibleQuery(userID)

	if keywords != nil && *keywords != "" {
		query.Where(
			sshhost.Or(
				sshhost.NameContains(*keywords),
				sshhost.HostContains(*keywords),
				sshhost.TagsContains(*keywords),
			),
		)
	}
	if host != nil && *host != "" {
		query.Where(sshhost.HostContains(*host))
	}

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.
		WithOwner().
		WithSharedUsers().
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		Order(gen.Asc(sshhost.FieldCreateTime)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, int64(total), nil
}

func (o *openSSHRepo) GetSSHHostByUserID(ctx context.Context, userID, hostID string) (*gen.SSHHost, error) {
	return o.accessibleQuery(userID).
		Where(sshhost.IDEQ(hostID)).
		WithOwner().
		WithSharedUsers().
		Only(ctx)
}

func (o *openSSHRepo) GetSSHHostByOwnerID(ctx context.Context, ownerID, hostID string) (*gen.SSHHost, error) {
	return o.data.db.SSHHost.Query().
		Where(
			sshhost.IDEQ(hostID),
			sshhost.HasOwnerWith(user.IDEQ(ownerID)),
		).
		WithOwner().
		WithSharedUsers().
		Only(ctx)
}

func (o *openSSHRepo) CreateSSHHost(ctx context.Context, req *v1.CreateSSHHostRequest, ownerID string, sharedUserIDs []string) (*gen.SSHHost, error) {
	authType, credentialText := createSSHCredential(req)
	credential, err := o.box.Encrypt(credentialText)
	if err != nil {
		return nil, err
	}
	passphrase, err := o.box.Encrypt(req.Passphrase)
	if err != nil {
		return nil, err
	}
	create := o.data.db.SSHHost.Create().
		SetID(uuid.NewString()).
		SetName(req.Name).
		SetHost(req.Host).
		SetPort(int(req.Port)).
		SetUsername(req.Username).
		SetAuthType(sshhost.AuthType(authType)).
		SetCredential(credential).
		SetPassphrase(passphrase).
		SetFingerprint(req.Fingerprint).
		SetRemark(req.Remark).
		SetTags(req.Tags).
		SetOwnerID(ownerID)

	if len(sharedUserIDs) > 0 {
		create.AddSharedUserIDs(sharedUserIDs...)
	}
	item, err := create.Save(ctx)
	if err != nil {
		return nil, err
	}
	return o.GetSSHHostByOwnerID(ctx, ownerID, item.ID)
}

func (o *openSSHRepo) UpdateSSHHost(ctx context.Context, req *v1.UpdateSSHHostRequest, ownerID string) (*gen.SSHHost, error) {
	if req.Id == "" {
		return nil, biz.ErrSSHHostInvalid
	}
	current, err := o.GetSSHHostByOwnerID(ctx, ownerID, req.Id)
	if err != nil {
		return nil, err
	}
	nextAuthType := string(current.AuthType)
	if req.AuthType != nil {
		nextAuthType = toDataSSHAuthType(*req.AuthType)
		if nextAuthType == "" {
			return nil, biz.ErrSSHAuthInvalid
		}
	}

	password := nonBlankOptional(req.Password)
	privateKey := nonBlankOptional(req.PrivateKey)
	credentialText := updateSSHCredential(nextAuthType, password, privateKey)
	switch nextAuthType {
	case sshcore.AuthPassword:
		if privateKey != nil {
			return nil, biz.ErrSSHAuthInvalid
		}
	case sshcore.AuthKey:
		if password != nil {
			return nil, biz.ErrSSHAuthInvalid
		}
	default:
		return nil, biz.ErrSSHAuthInvalid
	}
	if req.AuthType != nil && nextAuthType != string(current.AuthType) && credentialText == nil {
		return nil, biz.ErrSSHCredentialInvalid
	}

	update := o.data.db.SSHHost.UpdateOneID(req.Id).
		Where(sshhost.HasOwnerWith(user.IDEQ(ownerID)))

	if req.Name != nil {
		*req.Name = strings.TrimSpace(*req.Name)
		if *req.Name == "" {
			return nil, biz.ErrSSHHostInvalid
		}
		update.SetName(*req.Name)
	}
	if req.Host != nil {
		*req.Host = strings.TrimSpace(*req.Host)
		if *req.Host == "" {
			return nil, biz.ErrSSHHostInvalid
		}
		update.SetHost(*req.Host)
		update.SetStatus(sshhost.StatusUnknown)
	}
	if req.Port != nil {
		if *req.Port < 1 || *req.Port > 65535 {
			return nil, biz.ErrSSHHostInvalid
		}
		update.SetPort(int(*req.Port))
		update.SetStatus(sshhost.StatusUnknown)
	}
	if req.Username != nil {
		*req.Username = strings.TrimSpace(*req.Username)
		if *req.Username == "" {
			return nil, biz.ErrSSHHostInvalid
		}
		update.SetUsername(*req.Username)
		update.SetStatus(sshhost.StatusUnknown)
	}
	if req.AuthType != nil {
		update.SetAuthType(sshhost.AuthType(nextAuthType))
		update.SetStatus(sshhost.StatusUnknown)
	}
	if credentialText != nil {
		credential, err := o.box.Encrypt(*credentialText)
		if err != nil {
			return nil, err
		}
		update.SetCredential(credential)
		update.SetStatus(sshhost.StatusUnknown)
	}
	if req.Passphrase != nil {
		passphrase, err := o.box.Encrypt(*req.Passphrase)
		if err != nil {
			return nil, err
		}
		update.SetPassphrase(passphrase)
		update.SetStatus(sshhost.StatusUnknown)
	}
	if req.Fingerprint != nil {
		*req.Fingerprint = strings.TrimSpace(*req.Fingerprint)
		update.SetFingerprint(*req.Fingerprint)
		update.SetStatus(sshhost.StatusUnknown)
	}
	if req.Remark != nil {
		update.SetRemark(*req.Remark)
	}
	if req.Tags != nil {
		update.SetTags(*req.Tags)
	}

	if _, err := update.Save(ctx); err != nil {
		return nil, err
	}
	return o.GetSSHHostByOwnerID(ctx, ownerID, req.Id)
}

func (o *openSSHRepo) DeleteSSHHost(ctx context.Context, ownerID, hostID string) error {
	return o.data.db.SSHHost.DeleteOneID(hostID).
		Where(sshhost.HasOwnerWith(user.IDEQ(ownerID))).
		Exec(ctx)
}

func (o *openSSHRepo) ShareSSHHost(ctx context.Context, ownerID, hostID string, userIDs []string) (*gen.SSHHost, error) {
	if err := o.data.db.SSHHost.UpdateOneID(hostID).
		Where(sshhost.HasOwnerWith(user.IDEQ(ownerID))).
		ClearSharedUsers().
		AddSharedUserIDs(userIDs...).
		Exec(ctx); err != nil {
		return nil, err
	}
	return o.GetSSHHostByOwnerID(ctx, ownerID, hostID)
}

func (o *openSSHRepo) UpdateSSHHostStatus(
	ctx context.Context,
	userID string,
	hostID string,
	status v1.SSHHostStatus,
	fingerprint string,
) error {
	entStatus := toDataSSHHostStatus(status)
	if entStatus == "" {
		return biz.ErrSSHHostInvalid
	}

	update := o.data.db.SSHHost.UpdateOneID(hostID).
		Where(
			sshhost.Or(
				sshhost.HasOwnerWith(user.IDEQ(userID)),
				sshhost.HasSharedUsersWith(user.IDEQ(userID)),
			),
		).
		SetStatus(sshhost.Status(entStatus))

	fingerprint = strings.TrimSpace(fingerprint)
	if fingerprint != "" {
		update.SetFingerprint(fingerprint)
	}
	return update.Exec(ctx)
}

func (o *openSSHRepo) GetSSHHostConfigByUserID(ctx context.Context, userID, hostID string) (sshcore.Config, error) {
	item, err := o.GetSSHHostByUserID(ctx, userID, hostID)
	if err != nil {
		return sshcore.Config{}, err
	}
	credential, err := o.box.Decrypt(item.Credential)
	if err != nil {
		return sshcore.Config{}, err
	}
	passphrase, err := o.box.Decrypt(item.Passphrase)
	if err != nil {
		return sshcore.Config{}, err
	}
	return sshcore.Config{
		Host:        item.Host,
		Port:        item.Port,
		Username:    item.Username,
		AuthType:    string(item.AuthType),
		Credential:  credential,
		Passphrase:  passphrase,
		Fingerprint: item.Fingerprint,
	}, nil
}

func (o *openSSHRepo) accessibleQuery(userID string) *gen.SSHHostQuery {
	return o.data.db.SSHHost.Query().Where(
		sshhost.Or(
			sshhost.HasOwnerWith(user.IDEQ(userID)),
			sshhost.HasSharedUsersWith(user.IDEQ(userID)),
		),
	)
}

func createSSHCredential(req *v1.CreateSSHHostRequest) (string, string) {
	switch req.AuthType {
	case v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD:
		return sshcore.AuthPassword, req.Password
	case v1.SSHAuthType_SSH_AUTH_TYPE_KEY:
		return sshcore.AuthKey, req.PrivateKey
	default:
		return "", ""
	}
}

func updateSSHCredential(authType string, password, privateKey *string) *string {
	switch authType {
	case sshcore.AuthPassword:
		return password
	case sshcore.AuthKey:
		return privateKey
	default:
		return nil
	}
}

func nonBlankOptional(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	return value
}

func toDataSSHAuthType(authType v1.SSHAuthType) string {
	switch authType {
	case v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD:
		return sshcore.AuthPassword
	case v1.SSHAuthType_SSH_AUTH_TYPE_KEY:
		return sshcore.AuthKey
	default:
		return ""
	}
}

func toDataSSHHostStatus(status v1.SSHHostStatus) string {
	switch status {
	case v1.SSHHostStatus_SSH_HOST_STATUS_UNKNOWN:
		return "unknown"
	case v1.SSHHostStatus_SSH_HOST_STATUS_ONLINE:
		return "online"
	case v1.SSHHostStatus_SSH_HOST_STATUS_OFFLINE:
		return "offline"
	default:
		return ""
	}
}
