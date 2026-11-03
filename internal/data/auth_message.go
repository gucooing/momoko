package data

import (
	"context"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/operationlog"
	genuser "momoko/internal/data/ent/gen/user"
	"momoko/internal/data/ent/gen/usermessage"
)

// LoginHistory 返回该用户此前是否成功登录过，以及是否在 deviceID 上成功登录过（按登录操作日志判断）。
func (ar *authRepo) LoginHistory(ctx context.Context, userID, deviceID string) (bool, bool, error) {
	base := ar.data.db.OperationLog.Query().Where(
		operationlog.UserIDEQ(userID),
		operationlog.OperationTypeEQ("OperationTypeAuthLogin"),
		operationlog.SuccessEQ(true),
	)
	anyLogin, err := base.Clone().Exist(ctx)
	if err != nil || !anyLogin {
		return false, false, err
	}
	sameDevice, err := base.Clone().Where(operationlog.DetailContains(`"deviceId":"` + deviceID + `"`)).Exist(ctx)
	if err != nil {
		return true, false, err
	}
	return true, sameDevice, nil
}

func (ar *authRepo) CreateMessages(ctx context.Context, msgs []*biz.UserMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	const batch = 200
	now := time.Now()
	for start := 0; start < len(msgs); start += batch {
		end := min(start+batch, len(msgs))
		builders := make([]*gen.UserMessageCreate, 0, end-start)
		for _, m := range msgs[start:end] {
			category := m.Category
			if category == "" {
				category = biz.MessageCategorySystem
			}
			b := ar.data.db.UserMessage.Create().
				SetID(biz.NewMessageID()).
				SetUserID(m.UserID).
				SetCategory(category).
				SetKind(m.Kind).
				SetTitle(m.Title).
				SetContent(m.Content).
				SetSenderID(m.SenderID).
				SetCreateTime(now).
				SetUpdateTime(now)
			if len(m.Params) > 0 {
				b.SetParams(m.Params)
			}
			builders = append(builders, b)
		}
		if err := ar.data.db.UserMessage.CreateBulk(builders...).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (ar *authRepo) ListMessages(ctx context.Context, userID string, unreadOnly bool, page, pageSize int64) ([]*gen.UserMessage, int64, error) {
	query := ar.data.db.UserMessage.Query().Where(usermessage.UserIDEQ(userID))
	if unreadOnly {
		query = query.Where(usermessage.ReadAtIsNil())
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := query.
		Order(gen.Desc(usermessage.FieldCreateTime), gen.Desc(usermessage.FieldID)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	return rows, int64(total), nil
}

func (ar *authRepo) CountMessages(ctx context.Context, userID string) (int64, int64, error) {
	base := ar.data.db.UserMessage.Query().Where(usermessage.UserIDEQ(userID))
	all, err := base.Clone().Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	unread, err := base.Clone().Where(usermessage.ReadAtIsNil()).Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	return int64(all), int64(unread), nil
}

func (ar *authRepo) ReadMessages(ctx context.Context, userID string, ids []string, all bool) error {
	upd := ar.data.db.UserMessage.Update().Where(usermessage.UserIDEQ(userID), usermessage.ReadAtIsNil())
	if !all {
		upd = upd.Where(usermessage.IDIn(ids...))
	}
	_, err := upd.SetReadAt(time.Now()).Save(ctx)
	return err
}

func (ar *authRepo) DeleteMessages(ctx context.Context, userID string, ids []string, all bool) error {
	del := ar.data.db.UserMessage.Delete().Where(usermessage.UserIDEQ(userID))
	if !all {
		del = del.Where(usermessage.IDIn(ids...))
	}
	_, err := del.Exec(ctx)
	return err
}

func (ar *authRepo) ActiveUserIDs(ctx context.Context) ([]string, error) {
	return ar.data.db.User.Query().Where(genuser.StatusEQ(genuser.StatusActive)).IDs(ctx)
}
