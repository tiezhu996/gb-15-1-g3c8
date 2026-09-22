package handler

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/dto"
	"github.com/paperflow/paperflow/internal/service"
	"github.com/paperflow/paperflow/internal/util"
)

// WithdrawalHandler 论文撤稿申请处理器。
type WithdrawalHandler struct {
	withdrawalSvc *service.WithdrawalService
	paperSvc      *service.PaperService
	auditSvc      *service.AuditLogService
	logger        *slog.Logger
}

// NewWithdrawalHandler 构造撤稿处理器。
func NewWithdrawalHandler(withdrawalSvc *service.WithdrawalService, paperSvc *service.PaperService, auditSvc *service.AuditLogService, logger *slog.Logger) *WithdrawalHandler {
	return &WithdrawalHandler{withdrawalSvc: withdrawalSvc, paperSvc: paperSvc, auditSvc: auditSvc, logger: logger}
}

// Apply 作者发起撤稿申请。
func (h *WithdrawalHandler) Apply(c *gin.Context) {
	paperID, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.WithdrawalApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation,
			"撤稿申请失败：请求参数不合法（撤稿原因至少 10 字；外审/修稿阶段必须填写替代处理说明）")
		return
	}
	// 仅论文作者本人可发起撤稿。
	paper, err := h.paperSvc.Detail(c.Request.Context(), paperID)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	if paper.SubmitterID != util.GetUserID(c) && util.GetRole(c) != constants.RoleAdmin {
		util.Fail(c, http.StatusForbidden, constants.ErrPermissionDenied,
			fmt.Sprintf("撤稿申请失败：论文 id=%d 仅作者本人可申请撤稿", paperID))
		return
	}
	w, err := h.withdrawalSvc.Apply(c.Request.Context(), util.GetUserID(c), paperID, req)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	if e := h.auditSvc.Record(c.Request.Context(), util.GetUserID(c), util.GetUsername(c),
		constants.AuditActionApplyWithdrawal, "paper", fmt.Sprint(paperID),
		"发起撤稿申请："+req.Reason, c.ClientIP(), util.GetRequestID(c)); e != nil {
		h.logger.Error("audit apply withdrawal failed", "error", e)
	}
	util.OK(c, w)
}

// List 撤稿申请队列（编辑/管理员）。
func (h *WithdrawalHandler) List(c *gin.Context) {
	var q dto.WithdrawalQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, "查询失败：分页/状态参数不合法")
		return
	}
	page, size := pageSize(q.Page, q.Size)
	items, total, err := h.withdrawalSvc.List(c.Request.Context(), q.Status, q.PaperID, page, size)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	util.OK(c, dto.PageResult{Total: total, Page: page, Size: size, Items: items})
}

// GetByPaper 论文最近一次撤稿申请（列表/详情刷新后展示原因、处理结果与状态）。
func (h *WithdrawalHandler) GetByPaper(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	w, err := h.withdrawalSvc.GetByPaper(c.Request.Context(), id)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	util.OK(c, w)
}

// Decide 编辑批准/驳回撤稿申请。
func (h *WithdrawalHandler) Decide(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.WithdrawalDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, "撤稿处理失败：请求参数不合法")
		return
	}
	w, err := h.withdrawalSvc.Decide(c.Request.Context(), util.GetUserID(c), id, req)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	result := "批准撤稿"
	if !req.Approve {
		result = "驳回撤稿（恢复原流程）"
	}
	if e := h.auditSvc.Record(c.Request.Context(), util.GetUserID(c), util.GetUsername(c),
		constants.AuditActionDecideWithdrawal, "withdrawal", fmt.Sprint(id),
		fmt.Sprintf("%s：%s", result, req.Comment), c.ClientIP(), util.GetRequestID(c)); e != nil {
		h.logger.Error("audit decide withdrawal failed", "error", e)
	}
	util.OK(c, w)
}

// wrapError 撤稿处理器错误包装。
func (h *WithdrawalHandler) wrapError(c *gin.Context, err error) {
	var appErr *util.AppError
	if errors.As(err, &appErr) {
		util.Fail(c, httpStatusForCode(appErr.Code), appErr.Code, appErr.Message)
		return
	}
	util.Fail(c, http.StatusInternalServerError, constants.ErrInternal, "系统内部错误："+err.Error())
}
