package persistence

import (
	"context"
	"ops-hub/internal/domain/credential"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================
// Assemblers: Credential
// ============================

func credentialEntityToPO(e *credential.Credential) *CredentialPO {
	return &CredentialPO{
		ID:          e.ID,
		Name:        e.Name,
		CredType:    string(e.CredType),
		SecretData:  e.SecretData,
		Passphrase:  e.Passphrase,
		Fingerprint: e.Fingerprint,
		Description: e.Description,
		CreatedBy:   e.CreatedBy,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func credentialPOToEntity(po *CredentialPO) *credential.Credential {
	return &credential.Credential{
		ID:          po.ID,
		Name:        po.Name,
		CredType:    credential.CredType(po.CredType),
		SecretData:  po.SecretData,
		Passphrase:  po.Passphrase,
		Fingerprint: po.Fingerprint,
		Description: po.Description,
		CreatedBy:   po.CreatedBy,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

// CredentialRepository GORM 实现
type CredentialRepository struct {
	db *gorm.DB
}

func NewCredentialRepository(db *gorm.DB) *CredentialRepository {
	return &CredentialRepository{db: db}
}

func (r *CredentialRepository) Save(ctx context.Context, e *credential.Credential) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	po := credentialEntityToPO(e)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *CredentialRepository) Update(ctx context.Context, e *credential.Credential) error {
	e.UpdatedAt = time.Now()
	po := credentialEntityToPO(e)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *CredentialRepository) FindByID(ctx context.Context, id string) (*credential.Credential, error) {
	var po CredentialPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, credential.ErrCredentialNotFound
		}
		return nil, err
	}
	return credentialPOToEntity(&po), nil
}

func (r *CredentialRepository) Find(ctx context.Context, q credential.CredentialQuery) ([]*credential.Credential, int64, error) {
	db := r.db.WithContext(ctx).Model(&CredentialPO{})
	if q.CredType != "" {
		db = db.Where("cred_type = ?", q.CredType)
	}
	if q.Keyword != "" {
		db = db.Where("name ILIKE ?", "%"+q.Keyword+"%")
	}

	var total int64
	db.Count(&total)

	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var pos []CredentialPO
	if err := db.Order("created_at DESC").Offset(q.Offset()).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*credential.Credential, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = credentialPOToEntity(&poCopy)
	}
	return entities, total, nil
}

func (r *CredentialRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&CredentialPO{}, "id = ?", id).Error
}
