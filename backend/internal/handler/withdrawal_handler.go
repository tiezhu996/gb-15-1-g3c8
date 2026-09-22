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

// WithdrawalHandler 撤稿申请处理器。
type WithdrawalHandler struct {
	withdrawalSvc *service.WithdrawalService
	auditSvc      *service.AuditLogService
	logger        *slog.Logger
}

// NewWithdrawalHandler 构造撤稿申请处理器。
func NewWithdrawalHandler(withdrawalSvc *service.WithdrawalService, auditSvc *service.AuditLogService, logger *slog.Logger) *WithdrawalHandler {
	return &WithdrawalHandler{withdrawalSvc: withdrawalSvc, auditSvc: auditSvc, logger: logger}
}

// Apply 作者发起撤稿申请。
func (h *WithdrawalHandler) Apply(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.CreateWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, "撤稿申请失败：请求参数不合法（reason 至少 5 字）")
		return
	}
	w, err := h.withdrawalSvc.Apply(c.Request.Context(), util.GetUserID(c), id, req)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	if e := h.auditSvc.Record(c.Request.Context(), util.GetUserID(c), util.GetUsername(c),
		constants.AuditActionApplyWithdrawal, "withdrawal", fmt.Sprint(w.ID),
		fmt.Sprintf("论文 id=%d 发起撤稿申请", id), c.ClientIP(), util.GetRequestID(c)); e != nil {
		h.logger.Error("audit apply withdrawal failed", "error", e)
	}
	util.OK(c, w)
}

// ListMine 我的撤稿申请列表（作者）。
func (h *WithdrawalHandler) ListMine(c *gin.Context) {
	var q dto.WithdrawalQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, "查询失败：分页参数不合法")
		return
	}
	page, size := pageSize(q.Page, q.Size)
	items, total, err := h.withdrawalSvc.ListMine(c.Request.Context(), util.GetUserID(c), page, size)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	util.OK(c, dto.PageResult{Total: total, Page: page, Size: size, Items: items})
}

// List 撤稿申请列表（编辑审批队列）。
func (h *WithdrawalHandler) List(c *gin.Context) {
	var q dto.WithdrawalQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, "查询失败：分页参数不合法")
		return
	}
	page, size := pageSize(q.Page, q.Size)
	items, total, err := h.withdrawalSvc.List(c.Request.Context(), q.Status, page, size)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	util.OK(c, dto.PageResult{Total: total, Page: page, Size: size, Items: items})
}

// ListByPaper 论文的撤稿申请记录。
func (h *WithdrawalHandler) ListByPaper(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	items, err := h.withdrawalSvc.ListByPaper(c.Request.Context(), id)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	util.OK(c, items)
}

// Process 编辑处理撤稿申请（批准/驳回）。
func (h *WithdrawalHandler) Process(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req dto.ProcessWithdrawalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		util.Fail(c, http.StatusBadRequest, constants.CodeValidation, "撤稿处理失败：请求参数不合法（process_result 必填）")
		return
	}
	w, err := h.withdrawalSvc.Process(c.Request.Context(), util.GetUserID(c), id, req)
	if err != nil {
		h.wrapError(c, err)
		return
	}
	if e := h.auditSvc.Record(c.Request.Context(), util.GetUserID(c), util.GetUsername(c),
		constants.AuditActionProcessWithdrawal, "withdrawal", fmt.Sprint(id),
		fmt.Sprintf("处理撤稿申请：approve=%v", req.Approve), c.ClientIP(), util.GetRequestID(c)); e != nil {
		h.logger.Error("audit process withdrawal failed", "error", e)
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
