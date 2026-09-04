package credential

import (
	"context"
	"errors"
	"time"
)

// CredType 凭证类型
type CredType string

const (
	CredTypeSSHKey   CredType = "ssh_key"
	CredTypePassword CredType = "ssh_password"
	CredTypeToken    CredType = "token"
)

// Credential 凭证聚合根
type Credential struct {
	ID          string
	Name        string
	CredType    CredType
	SecretData  string // 加密存储的密钥/密码
	Passphrase  string // 加密存储的私钥密码短语（可选）
	Fingerprint string // SSH 公钥指纹
	Description string
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// CredentialRepository 凭证仓储接口
type CredentialRepository interface {
	Save(ctx context.Context, entity *Credential) error
	Update(ctx context.Context, entity *Credential) error
	FindByID(ctx context.Context, id string) (*Credential, error)
	Find(ctx context.Context, query CredentialQuery) ([]*Credential, int64, error)
	Delete(ctx context.Context, id string) error
}

// CredentialQuery 查询对象
type CredentialQuery struct {
	CredType string
	Keyword  string
	Page     int
	PageSize int
}

func (q CredentialQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// 领域错误
var (
	ErrCredentialNotFound   = errors.New("凭证不存在")
	ErrCredentialNameExists = errors.New("凭证名称已存在")
)
