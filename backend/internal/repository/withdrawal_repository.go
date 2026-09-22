package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/paperflow/paperflow/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// WithdrawalRepository 撤稿申请仓储接口。
type WithdrawalRepository interface {
	Create(ctx context.Context, w *model.Withdrawal) error
	Update(ctx context.Context, w *model.Withdrawal) error
	FindByID(ctx context.Context, id uint) (*model.Withdrawal, error)
	FindByIDForUpdate(ctx context.Context, id uint) (*model.Withdrawal, error)
	// FindPendingByPaper 查询论文当前待处理的撤稿申请（不存在返回 ErrNotFound）。
	FindPendingByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error)
	// FindLatestByPaper 论文最近一次撤稿申请（含已处理；不存在返回 ErrNotFound）。
	FindLatestByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error)
	List(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.Withdrawal, int64, error)
	// CloseOpenReviews 将论文未完成的审稿（invited/accepted）批量置为 closed。
	CloseOpenReviews(ctx context.Context, paperID uint, closedNote string) (int64, error)
}

type withdrawalRepository struct {
	db *gorm.DB
}

// NewWithdrawalRepository 构造撤稿申请仓储。
func NewWithdrawalRepository(db *gorm.DB) WithdrawalRepository {
	return &withdrawalRepository{db: db}
}

func (r *withdrawalRepository) Create(ctx context.Context, w *model.Withdrawal) error {
	if err := r.db.WithContext(ctx).Create(w).Error; err != nil {
		return fmt.Errorf("create withdrawal: %w", err)
	}
	return nil
}

func (r *withdrawalRepository) Update(ctx context.Context, w *model.Withdrawal) error {
	if err := r.db.WithContext(ctx).Omit("Paper", "Applicant", "ProcessedBy").Save(w).Error; err != nil {
		return fmt.Errorf("update withdrawal: %w", err)
	}
	return nil
}

func (r *withdrawalRepository) FindByID(ctx context.Context, id uint) (*model.Withdrawal, error) {
	var w model.Withdrawal
	if err := r.db.WithContext(ctx).First(&w, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find withdrawal %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("find withdrawal %d: %w", id, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) FindByIDForUpdate(ctx context.Context, id uint) (*model.Withdrawal, error) {
	var w model.Withdrawal
	if err := r.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&w, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("lock withdrawal %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("lock withdrawal %d: %w", id, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) FindPendingByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error) {
	var w model.Withdrawal
	if err := r.db.WithContext(ctx).
		Where("paper_id = ? AND status = ?", paperID, "pending").
		First(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find pending withdrawal paper %d: %w", paperID, ErrNotFound)
		}
		return nil, fmt.Errorf("find pending withdrawal paper %d: %w", paperID, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) FindLatestByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error) {
	var w model.Withdrawal
	if err := r.db.WithContext(ctx).
		Where("paper_id = ?", paperID).Order("created_at DESC").First(&w).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find latest withdrawal paper %d: %w", paperID, ErrNotFound)
		}
		return nil, fmt.Errorf("find latest withdrawal paper %d: %w", paperID, err)
	}
	return &w, nil
}

func (r *withdrawalRepository) List(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.Withdrawal, int64, error) {
	q := r.db.WithContext(ctx).Model(&model.Withdrawal{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.PaperID > 0 {
		q = q.Where("paper_id = ?", filter.PaperID)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count withdrawals: %w", err)
	}
	var items []model.Withdrawal
	if err := q.Preload("Paper").Preload("Applicant").
		Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list withdrawals: %w", err)
	}
	return items, total, nil
}

func (r *withdrawalRepository) CloseOpenReviews(ctx context.Context, paperID uint, closedNote string) (int64, error) {
	res := r.db.WithContext(ctx).Model(&model.Review{}).
		Where("paper_id = ? AND status IN ?", paperID, []string{"invited", "accepted"}).
		Updates(map[string]any{"status": "closed", "comments": gorm.Expr("COALESCE(NULLIF(comments, ''), ?)", closedNote)})
	if res.Error != nil {
		return 0, fmt.Errorf("close open reviews paper %d: %w", paperID, res.Error)
	}
	return res.RowsAffected, nil
}
