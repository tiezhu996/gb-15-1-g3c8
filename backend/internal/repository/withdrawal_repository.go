package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WithdrawalRepository 撤稿申请仓储接口。
type WithdrawalRepository interface {
	Create(ctx context.Context, w *model.WithdrawalRequest) error
	Update(ctx context.Context, w *model.WithdrawalRequest) error
	FindByID(ctx context.Context, id uint) (*model.WithdrawalRequest, error)
	FindByIDForUpdate(ctx context.Context, id uint) (*model.WithdrawalRequest, error)
	FindPendingByPaper(ctx context.Context, paperID uint) (*model.WithdrawalRequest, error)
	ExistsPendingByPaper(ctx context.Context, paperID uint) (bool, error)
	List(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.WithdrawalRequest, int64, error)
	ListByPaper(ctx context.Context, paperID uint) ([]model.WithdrawalRequest, error)
}

type withdrawalRepository struct {
	db *gorm.DB
}

// NewWithdrawalRepository 构造撤稿申请仓储。
func NewWithdrawalRepository(db *gorm.DB) WithdrawalRepository {
	return &withdrawalRepository{db: db}
}

func (r *withdrawalRepository) Create(ctx context.Context, w *model.WithdrawalRequest) error {
	if err := r.db.WithContext(ctx).Create(w).Error; err != nil {
		return fmt.Errorf("create withdrawal: %w", err)
	}
	return nil
}

func (r *withdrawalRepository) Update(ctx context.Context, w *model.WithdrawalRequest) error {
	if err := r.db.WithContext(ctx).Omit("Paper", "Applicant", "ProcessedBy").Save(w).Error; err != nil {
		return fmt.Errorf("update withdrawal: %w", err)
	}
	return nil
}

func (r *withdrawalRepository) FindByID(ctx context.Context, id uint) (*model.WithdrawalRequest, error) {
	var w model.WithdrawalRequest
	if err := r.db.WithContext(ctx).
		Preload("Paper").Preload("Applicant").Preload("ProcessedBy").
		First(&w, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find withdrawal %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("find withdrawal %d: %w", id, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) FindByIDForUpdate(ctx context.Context, id uint) (*model.WithdrawalRequest, error) {
	var w model.WithdrawalRequest
	if err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&w, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("lock withdrawal %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("lock withdrawal %d: %w", id, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) FindPendingByPaper(ctx context.Context, paperID uint) (*model.WithdrawalRequest, error) {
	var w model.WithdrawalRequest
	if err := r.db.WithContext(ctx).
		Where("paper_id = ? AND status = ?", paperID, constants.WithdrawalStatusPending).
		First(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find pending withdrawal paper %d: %w", paperID, ErrNotFound)
		}
		return nil, fmt.Errorf("find pending withdrawal paper %d: %w", paperID, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) ExistsPendingByPaper(ctx context.Context, paperID uint) (bool, error) {
	var n int64
	if err := r.db.WithContext(ctx).Model(&model.WithdrawalRequest{}).
		Where("paper_id = ? AND status = ?", paperID, constants.WithdrawalStatusPending).
		Count(&n).Error; err != nil {
		return false, fmt.Errorf("count pending withdrawal paper %d: %w", paperID, err)
	}
	return n > 0, nil
}

func (r *withdrawalRepository) List(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.WithdrawalRequest, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.WithdrawalRequest{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.ApplicantID > 0 {
		q = q.Where("applicant_id = ?", filter.ApplicantID)
	}
	if filter.PaperID > 0 {
		q = q.Where("paper_id = ?", filter.PaperID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count withdrawals: %w", err)
	}
	var items []model.WithdrawalRequest
	if err := q.Preload("Paper").Preload("Applicant").Preload("ProcessedBy").
		Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list withdrawals: %w", err)
	}
	return items, total, nil
}

func (r *withdrawalRepository) ListByPaper(ctx context.Context, paperID uint) ([]model.WithdrawalRequest, error) {
	var items []model.WithdrawalRequest
	if err := r.db.WithContext(ctx).Preload("Applicant").Preload("ProcessedBy").
		Where("paper_id = ?", paperID).Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list withdrawals by paper %d: %w", paperID, err)
	}
	return items, nil
}
