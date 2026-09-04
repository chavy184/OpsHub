package credential

import (
	"context"
	"crypto/md5"
	"fmt"
	"ops-hub/internal/domain/credential"
	"ops-hub/internal/infrastructure/crypto"
	"strings"
	"time"
)

// ============================
// DTO
// ============================

type CreateCredentialCmd struct {
	Name        string `json:"name" binding:"required"`
	CredType    string `json:"cred_type"`
	SecretData  string `json:"secret_data" binding:"required"` // 明文，保存时加密
	Passphrase  string `json:"passphrase"`                     // SSH 私钥密码短语（可选）
	Description string `json:"description"`
	CreatedBy   string `json:"created_by"`
}

type UpdateCredentialCmd struct {
	ID          string `json:"-"`
	Name        string `json:"name"`
	SecretData  string `json:"secret_data"` // 若为空则不更新
	Passphrase  string `json:"passphrase"`  // SSH 私钥密码短语
	Description string `json:"description"`
}

type CredentialQueryCmd struct {
	CredType string `form:"cred_type"`
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type CredentialDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CredType    string `json:"cred_type"`
	Fingerprint string `json:"fingerprint"`
	Description string `json:"description"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toDTO(e *credential.Credential) *CredentialDTO {
	return &CredentialDTO{
		ID:          e.ID,
		Name:        e.Name,
		CredType:    string(e.CredType),
		Fingerprint: e.Fingerprint,
		Description: e.Description,
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt.Format(time.DateTime),
		UpdatedAt:   e.UpdatedAt.Format(time.DateTime),
	}
}

// ============================
// UseCase
// ============================

type UseCase struct {
	repo      credential.CredentialRepository
	encryptor crypto.Encryptor
}

func NewUseCase(repo credential.CredentialRepository, encryptor crypto.Encryptor) *UseCase {
	return &UseCase{repo: repo, encryptor: encryptor}
}

// Create 创建凭证
func (uc *UseCase) Create(ctx context.Context, cmd CreateCredentialCmd) (*CredentialDTO, error) {
	// 加密密钥数据
	encrypted, err := uc.encryptor.Encrypt(cmd.SecretData)
	if err != nil {
		return nil, fmt.Errorf("凭证加密失败: %w", err)
	}

	credType := credential.CredType(cmd.CredType)
	if credType == "" {
		credType = credential.CredTypeSSHKey
	}

	// 计算 SSH 密钥指纹（MD5）
	fingerprint := ""
	if credType == credential.CredTypeSSHKey {
		h := md5.Sum([]byte(cmd.SecretData))
		parts := make([]string, 16)
		for i, b := range h {
			parts[i] = fmt.Sprintf("%02x", b)
		}
		fingerprint = strings.Join(parts, ":")
	}

	e := &credential.Credential{
		Name:        cmd.Name,
		CredType:    credType,
		SecretData:  encrypted,
		Fingerprint: fingerprint,
		Description: cmd.Description,
		CreatedBy:   cmd.CreatedBy,
	}

	// 加密存储 passphrase
	if cmd.Passphrase != "" {
		encPassphrase, ppErr := uc.encryptor.Encrypt(cmd.Passphrase)
		if ppErr != nil {
			return nil, fmt.Errorf("密码短语加密失败: %w", ppErr)
		}
		e.Passphrase = encPassphrase
	}

	if err := uc.repo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("保存凭证失败: %w", err)
	}

	return toDTO(e), nil
}

// Update 更新凭证
func (uc *UseCase) Update(ctx context.Context, cmd UpdateCredentialCmd) (*CredentialDTO, error) {
	e, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, credential.ErrCredentialNotFound
	}

	if cmd.Name != "" {
		e.Name = cmd.Name
	}
	if cmd.Description != "" {
		e.Description = cmd.Description
	}
	if cmd.SecretData != "" {
		encrypted, encErr := uc.encryptor.Encrypt(cmd.SecretData)
		if encErr != nil {
			return nil, fmt.Errorf("凭证加密失败: %w", encErr)
		}
		e.SecretData = encrypted
	}
	if cmd.Passphrase != "" {
		encPassphrase, ppErr := uc.encryptor.Encrypt(cmd.Passphrase)
		if ppErr != nil {
			return nil, fmt.Errorf("密码短语加密失败: %w", ppErr)
		}
		e.Passphrase = encPassphrase
	} else if cmd.Passphrase == "" && cmd.SecretData != "" {
		// 如果更新了密钥但未提供 passphrase，清空旧的 passphrase
		e.Passphrase = ""
	}

	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新凭证失败: %w", err)
	}

	return toDTO(e), nil
}

// Get 获取凭证
func (uc *UseCase) Get(ctx context.Context, id string) (*CredentialDTO, error) {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, credential.ErrCredentialNotFound
	}
	return toDTO(e), nil
}

// List 查询凭证列表
func (uc *UseCase) List(ctx context.Context, cmd CredentialQueryCmd) ([]*CredentialDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}

	entities, total, err := uc.repo.Find(ctx, credential.CredentialQuery{
		CredType: cmd.CredType,
		Keyword:  cmd.Keyword,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*CredentialDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toDTO(e)
	}
	return dtos, total, nil
}

// Delete 删除凭证
func (uc *UseCase) Delete(ctx context.Context, id string) error {
	if _, err := uc.repo.FindByID(ctx, id); err != nil {
		return credential.ErrCredentialNotFound
	}
	return uc.repo.Delete(ctx, id)
}
