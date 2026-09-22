package dto

// WithdrawalApplyRequest 作者发起撤稿申请请求。
type WithdrawalApplyRequest struct {
	// Reason 撤稿原因（始终必填）。
	Reason string `json:"reason" binding:"required,min=10,max=2000"`
	// AlternativeNote 替代处理说明：论文处于外审中/修改中时必填。
	AlternativeNote string `json:"alternative_note" binding:"omitempty,min=10,max=2000"`
}

// WithdrawalDecisionRequest 编辑处理撤稿申请请求。
type WithdrawalDecisionRequest struct {
	// Approve true=批准撤稿，false=驳回并恢复原流程。
	Approve bool `json:"approve"`
	// Comment 处理意见；驳回时必填。
	Comment string `json:"comment" binding:"omitempty,max=2000"`
}

// WithdrawalQuery 撤稿申请列表查询参数。
type WithdrawalQuery struct {
	Status  string `form:"status" binding:"omitempty,oneof=pending approved rejected"`
	PaperID uint   `form:"paper_id" binding:"omitempty"`
	Page    int    `form:"page" binding:"omitempty,min=1"`
	Size    int    `form:"size" binding:"omitempty,min=1,max=100"`
}
