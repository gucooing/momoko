package common

type OperationType string

const (
	OperationTypeLogin OperationType = "login" // 登录操作
)

func (o OperationType) String() string {
	return string(o)
}
