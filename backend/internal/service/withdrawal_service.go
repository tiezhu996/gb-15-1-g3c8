package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/dto"
	"github.com/paperflow/paperflow/internal/model"
	"github.com/paperflow/paperflow/internal/repository"
	"github.com/paperflow/paperflow/internal/util"
)

// WithdrawalService 撤稿申请服务：作者申请、编辑审批、终态保护。
type WithdrawalService struct {
	store  repository.Store
	logger *slog.Logger
}

// NewWithdrawalService 构造撤稿申请服务。
func NewWithdrawalService(store repository.Store, logger *slog.Logger) *WithdrawalService {
	return &WithdrawalService{store: store, logger: logger}
}

// ensureNoPendingWithdrawal 待处理撤稿申请冻结校验：存在时审稿/修稿/查重等流程不得推进。
// 被 PaperService / ReviewService / PlagiarismService 多个 service 复用。
func ensureNoPendingWithdrawal(ctx context.Context, store repository.Store, paperID uint) error {
	exists, err := store.WithdrawalRepository().ExistsPendingByPaper(ctx, paperID)
	if err != nil {
		return util.NewAppError(constants.ErrInternal, "撤稿冻结校验失败：系统内部错误", err)
	}
	if exists {
		return util.NewAppError(constants.ErrWithdrawalPending,
			fmt.Sprintf("操作失败：论文 id=%d 存在待处理撤稿申请，审稿、修稿与查重流程已暂停，待编辑处理后方可继续", paperID), nil)
	}
	return nil
}

// Apply 作者发起撤稿申请：事务内锁论文行，重复或并发申请只保留一条。
func (s *WithdrawalService) Apply(ctx context.Context, applicantID, paperID uint, req dto.CreateWithdrawalRequest) (*model.WithdrawalRequest, error) {
	var created *model.WithdrawalRequest
	err := s.store.Transaction(ctx, func(tx repository.Store) error {
		paper, err := tx.PaperRepository().FindByIDForUpdate(ctx, paperID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.ErrPaperNotFound,
					fmt.Sprintf("撤稿申请失败：论文 id=%d 不存在", paperID), nil)
			}
			return util.NewAppError(constants.ErrInternal, "撤稿申请失败：锁定论文时系统内部错误", err)
		}
		if paper.SubmitterID != applicantID {
			return util.NewAppError(constants.ErrPermissionDenied,
				fmt.Sprintf("撤稿申请失败：论文 id=%d 不属于当前作者 id=%d，仅本人可发起撤稿", paperID, applicantID), nil)
		}
		switch paper.Status {
		case constants.PaperStatusSubmitted, constants.PaperStatusInitialReview,
			constants.PaperStatusExternalReview, constants.PaperStatusRevision:
			// 未录用在审论文允许撤稿
		default:
			return util.NewAppError(constants.ErrWithdrawalNotAllowed,
				fmt.Sprintf("撤稿申请失败：论文 %s 当前状态 %s 为终态，不允许撤稿",
					paper.Title, util.FormatPaperStatus(paper.Status)), nil)
		}
		if (paper.Status == constants.PaperStatusExternalReview || paper.Status == constants.PaperStatusRevision) &&
			req.AltHandlingNote == "" {
			return util.NewAppError(constants.ErrWithdrawalNotAllowed,
				fmt.Sprintf("撤稿申请失败：论文 %s 当前状态 %s，须填写替代处理说明",
					paper.Title, util.FormatPaperStatus(paper.Status)), nil)
		}
		if _, err := tx.WithdrawalRepository().FindPendingByPaper(ctx, paperID); err == nil {
			return util.NewAppError(constants.ErrWithdrawalNotAllowed,
				fmt.Sprintf("撤稿申请失败：论文 id=%d 已存在待处理撤稿申请，重复或并发申请只保留一条", paperID), nil)
		} else if !errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(constants.ErrInternal, "撤稿申请失败：查询已有申请时系统内部错误", err)
		}
		created = &model.WithdrawalRequest{
			PaperID:         paperID,
			ApplicantID:     applicantID,
			Reason:          req.Reason,
			AltHandlingNote: req.AltHandlingNote,
			Status:          constants.WithdrawalStatusPending,
		}
		if err := tx.WithdrawalRepository().Create(ctx, created); err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿申请失败：保存申请时系统内部错误", err)
		}
		return nil
	})
	if err != nil {
		var appErr *util.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, util.NewAppError(constants.ErrInternal, "撤稿申请失败：系统内部错误", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogWithdrawalApply, created.ID, paperID, applicantID))
	return s.store.WithdrawalRepository().FindByID(ctx, created.ID)
}

// list 复用同一仓储 List 方法的统一列表查询（被 ListMine/List 复用）。
func (s *WithdrawalService) list(ctx context.Context, filter model.WithdrawalFilter, page, size int) ([]model.WithdrawalRequest, int64, error) {
	items, total, err := s.store.WithdrawalRepository().List(ctx, filter, page, size)
	if err != nil {
		return nil, 0, util.NewAppError(constants.ErrInternal, "查询撤稿申请列表失败：系统内部错误", err)
	}
	return items, total, nil
}

// ListMine 我的撤稿申请列表（作者）。
func (s *WithdrawalService) ListMine(ctx context.Context, applicantID uint, page, size int) ([]model.WithdrawalRequest, int64, error) {
	return s.list(ctx, model.WithdrawalFilter{ApplicantID: applicantID}, page, size)
}

// List 撤稿申请列表（编辑审批队列；status 为空则列出全部）。
func (s *WithdrawalService) List(ctx context.Context, status string, page, size int) ([]model.WithdrawalRequest, int64, error) {
	items, total, err := s.list(ctx, model.WithdrawalFilter{Status: status}, page, size)
	if err != nil {
		return nil, 0, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogWithdrawalList, status, page))
	return items, total, nil
}

// ListByPaper 论文的撤稿申请记录（详情页展示原因、处理结果与状态）。
func (s *WithdrawalService) ListByPaper(ctx context.Context, paperID uint) ([]model.WithdrawalRequest, error) {
	items, err := s.store.WithdrawalRepository().ListByPaper(ctx, paperID)
	if err != nil {
		return nil, util.NewAppError(constants.ErrInternal, "撤稿申请记录获取失败：系统内部错误", err)
	}
	return items, nil
}

// Process 编辑处理撤稿申请：批准则论文进入已撤稿并关闭未完成审稿，驳回则恢复原流程。
func (s *WithdrawalService) Process(ctx context.Context, editorID, withdrawalID uint, req dto.ProcessWithdrawalRequest) (*model.WithdrawalRequest, error) {
	var closedReviews int64
	err := s.store.Transaction(ctx, func(tx repository.Store) error {
		w, err := tx.WithdrawalRepository().FindByIDForUpdate(ctx, withdrawalID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.ErrWithdrawalNotFound,
					fmt.Sprintf("撤稿处理失败：撤稿申请 id=%d 不存在", withdrawalID), nil)
			}
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：锁定申请时系统内部错误", err)
		}
		if w.Status != constants.WithdrawalStatusPending {
			return util.NewAppError(constants.ErrWithdrawalNotAllowed,
				fmt.Sprintf("撤稿处理失败：撤稿申请 id=%d 当前状态 %s 不可重复处理",
					withdrawalID, util.FormatWithdrawalStatus(w.Status)), nil)
		}
		now := timeNow()
		w.ProcessedByID = &editorID
		w.ProcessResult = req.ProcessResult
		w.ProcessedAt = &now
		if !req.Approve {
			w.Status = constants.WithdrawalStatusRejected
			if err := tx.WithdrawalRepository().Update(ctx, w); err != nil {
				return util.NewAppError(constants.ErrInternal, "撤稿处理失败：保存处理结果时系统内部错误", err)
			}
			s.logger.Info(fmt.Sprintf(constants.LogWithdrawalReject, w.ID, w.PaperID))
			return nil
		}
		paper, err := tx.PaperRepository().FindByIDForUpdate(ctx, w.PaperID)
		if err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：锁定论文时系统内部错误", err)
		}
		switch paper.Status {
		case constants.PaperStatusSubmitted, constants.PaperStatusInitialReview,
			constants.PaperStatusExternalReview, constants.PaperStatusRevision:
			// 仅在审论文可转入已撤稿
		default:
			return util.NewAppError(constants.ErrWithdrawalNotAllowed,
				fmt.Sprintf("撤稿处理失败：论文 %s 当前状态 %s 为终态，不可批准撤稿",
					paper.Title, util.FormatPaperStatus(paper.Status)), nil)
		}
		paper.Status = constants.PaperStatusWithdrawn
		if err := tx.PaperRepository().Update(ctx, paper); err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：更新论文状态时系统内部错误", err)
		}
		closedReviews, err = tx.ReviewRepository().CloseUnfinishedByPaper(ctx, paper.ID)
		if err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：关闭未完成审稿时系统内部错误", err)
		}
		w.Status = constants.WithdrawalStatusApproved
		if err := tx.WithdrawalRepository().Update(ctx, w); err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：保存处理结果时系统内部错误", err)
		}
		s.logger.Info(fmt.Sprintf(constants.LogWithdrawalApprove, w.ID, w.PaperID, closedReviews))
		return nil
	})
	if err != nil {
		var appErr *util.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, util.NewAppError(constants.ErrInternal, "撤稿处理失败：系统内部错误", err)
	}
	return s.store.WithdrawalRepository().FindByID(ctx, withdrawalID)
}
