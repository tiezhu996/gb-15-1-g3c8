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

// WithdrawalService 论文撤稿申请服务：作者发起申请、编辑批准/驳回、终态保护。
type WithdrawalService struct {
	store  repository.Store
	logger *slog.Logger
}

// NewWithdrawalService 构造撤稿服务。
func NewWithdrawalService(store repository.Store, logger *slog.Logger) *WithdrawalService {
	return &WithdrawalService{store: store, logger: logger}
}

// Apply 作者对本人未录用论文发起一次撤稿申请。
// 申请提交后论文流程冻结：审稿、修稿、查重等记录可查看但不能推进。
func (s *WithdrawalService) Apply(ctx context.Context, applicantID uint, paperID uint, req dto.WithdrawalApplyRequest) (*model.Withdrawal, error) {
	var created *model.Withdrawal
	err := s.store.Transaction(ctx, func(tx repository.Store) error {
		paper, err := tx.PaperRepository().FindByIDForUpdate(ctx, paperID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.ErrPaperNotFound,
					fmt.Sprintf("撤稿申请失败：论文 id=%d 不存在", paperID), nil)
			}
			return util.NewAppError(constants.ErrInternal, "撤稿申请失败：查询论文时系统内部错误", err)
		}
		if paper.SubmitterID != applicantID {
			return util.NewAppError(constants.ErrPermissionDenied,
				fmt.Sprintf("撤稿申请失败：论文 id=%d 不属于当前用户", paperID), nil)
		}
		if paper.Status == constants.PaperStatusAccepted {
			return util.NewAppError(constants.ErrPaperStatusNotAllowed,
				"撤稿申请失败：已录用论文不可撤稿", nil)
		}
		if paper.Status == constants.PaperStatusWithdrawn {
			return util.NewAppError(constants.ErrPaperStatusNotAllowed,
				fmt.Sprintf("撤稿申请失败：论文《%s》已撤稿，终态不可变更", paper.Title), nil)
		}
		if !containsStatus(constants.PaperStatusInProcess, paper.Status) {
			return util.NewAppError(constants.ErrPaperStatusNotAllowed,
				fmt.Sprintf("撤稿申请失败：论文《%s》当前状态 %s 不可发起撤稿",
					paper.Title, util.FormatPaperStatus(paper.Status)), nil)
		}
		// 重复/并发申请：仅保留一条待处理记录（唯一索引兜底 + 事务内行锁）。
		if existing, err := tx.WithdrawalRepository().FindPendingByPaper(ctx, paperID); err == nil {
			return util.NewAppError(constants.ErrWithdrawalExists,
				fmt.Sprintf("撤稿申请失败：论文《%s》已存在待处理撤稿申请（申请 id=%d），请勿重复提交",
					paper.Title, existing.ID), nil)
		} else if !errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(constants.ErrInternal, "撤稿申请失败：查询已有申请时系统内部错误", err)
		}
		if containsStatus(constants.WithdrawalAlternativeRequiredStatuses, paper.Status) && req.AlternativeNote == "" {
			return util.NewAppError(constants.ErrBadRequest,
				fmt.Sprintf("撤稿申请失败：论文当前处于%s，必须填写替代处理说明（如改投建议、后续安排）",
					util.FormatPaperStatus(paper.Status)), nil)
		}
		w := &model.Withdrawal{
			PaperID:         paperID,
			ApplicantID:     applicantID,
			Reason:          req.Reason,
			AlternativeNote: req.AlternativeNote,
			Status:          constants.WithdrawalStatusPending,
		}
		if err := tx.WithdrawalRepository().Create(ctx, w); err != nil {
			return util.NewAppError(constants.ErrConflict,
				fmt.Sprintf("撤稿申请失败：论文 id=%d 已存在待处理撤稿申请，请勿重复提交", paperID), err)
		}
		created = w
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

// Decide 编辑处理撤稿申请：驳回恢复原流程，批准后论文进入已撤稿终态。
func (s *WithdrawalService) Decide(ctx context.Context, editorID uint, withdrawalID uint, req dto.WithdrawalDecisionRequest) (*model.Withdrawal, error) {
	err := s.store.Transaction(ctx, func(tx repository.Store) error {
		w, err := tx.WithdrawalRepository().FindByIDForUpdate(ctx, withdrawalID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(constants.ErrWithdrawalNotFound,
					fmt.Sprintf("撤稿处理失败：撤稿申请 id=%d 不存在", withdrawalID), nil)
			}
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：查询申请时系统内部错误", err)
		}
		if w.Status != constants.WithdrawalStatusPending {
			return util.NewAppError(constants.ErrPaperStatusNotAllowed,
				fmt.Sprintf("撤稿处理失败：申请 id=%d 已处理（%s），不可重复处理",
					withdrawalID, util.FormatWithdrawalStatus(w.Status)), nil)
		}
		if !req.Approve && req.Comment == "" {
			return util.NewAppError(constants.ErrBadRequest,
				"撤稿处理失败：驳回撤稿申请必须填写处理意见（恢复原流程的说明）", nil)
		}
		paper, err := tx.PaperRepository().FindByIDForUpdate(ctx, w.PaperID)
		if err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：查询论文时系统内部错误", err)
		}
		now := timeNow()
		w.ProcessedAt = &now
		w.ProcessedByID = &editorID
		w.DecisionComment = req.Comment
		if req.Approve {
			// 批准：论文进入已撤稿终态，未完成审稿一并关闭；撤回前的审稿/修稿/查重记录原样保留。
			w.Status = constants.WithdrawalStatusApproved
			paper.Status = constants.PaperStatusWithdrawn
			if _, err := tx.WithdrawalRepository().CloseOpenReviews(ctx, paper.ID,
				"论文已由作者申请撤稿并经编辑部批准，审稿任务关闭。"); err != nil {
				return util.NewAppError(constants.ErrInternal, "撤稿处理失败：关闭未完成审稿时系统内部错误", err)
			}
		} else {
			// 驳回：申请终态化，论文状态未被改动，自动恢复原流程。
			w.Status = constants.WithdrawalStatusRejected
		}
		if err := tx.WithdrawalRepository().Update(ctx, w); err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：保存申请时系统内部错误", err)
		}
		if err := tx.PaperRepository().Update(ctx, paper); err != nil {
			return util.NewAppError(constants.ErrInternal, "撤稿处理失败：保存论文时系统内部错误", err)
		}
		return nil
	})
	if err != nil {
		var appErr *util.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		return nil, util.NewAppError(constants.ErrInternal, "撤稿处理失败：系统内部错误", err)
	}
	w, err := s.store.WithdrawalRepository().FindByID(ctx, withdrawalID)
	if err != nil {
		return nil, util.NewAppError(constants.ErrInternal, "撤稿处理失败：查询处理结果时系统内部错误", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogWithdrawalDecide, withdrawalID, w.PaperID, req.Approve, editorID))
	return w, nil
}

// List 撤稿申请列表（编辑处理队列；按状态过滤）。
func (s *WithdrawalService) List(ctx context.Context, status string, paperID uint, page, size int) ([]model.Withdrawal, int64, error) {
	items, total, err := s.store.WithdrawalRepository().List(ctx,
		model.WithdrawalFilter{Status: status, PaperID: paperID}, page, size)
	if err != nil {
		return nil, 0, util.NewAppError(constants.ErrInternal, "撤稿申请列表获取失败：系统内部错误", err)
	}
	return items, total, nil
}

// GetByPaper 论文最近一次撤稿申请（供详情/状态展示）。
func (s *WithdrawalService) GetByPaper(ctx context.Context, paperID uint) (*model.Withdrawal, error) {
	w, err := s.store.WithdrawalRepository().FindLatestByPaper(ctx, paperID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.ErrWithdrawalNotFound,
				fmt.Sprintf("撤稿记录获取失败：论文 id=%d 暂无撤稿申请", paperID), nil)
		}
		return nil, util.NewAppError(constants.ErrInternal, "撤稿记录获取失败：系统内部错误", err)
	}
	return w, nil
}

// EnsureFlowAllowed 流程守卫：待处理撤稿冻结流程推进，已撤稿为终态。
// 依赖调用方已预加载 paper.Withdrawal（如 PaperRepository.FindByIDWithDetail/List）。
// action 为尝试推进的业务动作名称，用于错误提示。
func EnsureFlowAllowed(paper *model.Paper, action string) error {
	if paper.Status == constants.PaperStatusWithdrawn {
		return util.NewAppError(constants.ErrPaperStatusNotAllowed,
			fmt.Sprintf("%s失败：论文《%s》已撤稿，终态不可变更", action, paper.Title), nil)
	}
	if paper.Withdrawal != nil && paper.Withdrawal.Status == constants.WithdrawalStatusPending {
		return util.NewAppError(constants.ErrPaperStatusNotAllowed,
			fmt.Sprintf("%s失败：论文《%s》存在待处理撤稿申请，流程已暂停，待编辑部处理后再试",
				action, paper.Title), nil)
	}
	return nil
}

// GuardPaperFlow 事务内守卫：基于论文状态与待处理撤稿申请，阻断终态/冻结期的流程推进。
// action 为尝试推进的业务动作名称，用于错误提示。
func GuardPaperFlow(ctx context.Context, tx repository.Store, paper *model.Paper, action string) error {
	if err := EnsureFlowAllowed(paper, action); err != nil {
		return err
	}
	if _, err := tx.WithdrawalRepository().FindPendingByPaper(ctx, paper.ID); err == nil {
		return util.NewAppError(constants.ErrPaperStatusNotAllowed,
			fmt.Sprintf("%s失败：论文《%s》存在待处理撤稿申请，流程已暂停，待编辑部处理后再试",
				action, paper.Title), nil)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return util.NewAppError(constants.ErrInternal, "撤稿状态校验失败：系统内部错误", err)
	}
	return nil
}

func containsStatus(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}
