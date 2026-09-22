package dto

// CreateWithdrawalRequest 发起撤稿申请请求。
type CreateWithdrawalRequest struct {
	Reason          string `json:"reason" binding:"required,min=5,max=2000"`
	AltHandlingNote string `json:"alt_handling_note" binding:"omitempty,max=2000"`
}

// ProcessWithdrawalRequest 编辑处理撤稿申请请求。
type ProcessWithdrawalRequest struct {
	Approve       bool   `json:"approve"`
	ProcessResult string `json:"process_result" binding:"required,min=2,max=2000"`
}

// WithdrawalQuery 撤稿申请列表查询参数。
type WithdrawalQuery struct {
	Status string `form:"status" binding:"omitempty,oneof=pending approved rejected"`
	Page   int    `form:"page" binding:"omitempty,min=1"`
	Size   int    `form:"size" binding:"omitempty,min=1,max=100"`
}
