package model

import "time"

// Withdrawal 论文撤稿申请：作者对本人未录用论文发起一次撤稿，编辑批准/驳回。
type Withdrawal struct {
	ID uint `gorm:"primaryKey" json:"id"`
	// 部分唯一索引：每篇论文至多一条 pending 申请（驳回/批准后该申请不再占位，允许重新申请）。
	PaperID     uint  `gorm:"not null;index;uniqueIndex:uniq_withdrawal_pending,where:status = 'pending'" json:"paper_id"`
	Paper       Paper `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
	ApplicantID uint  `gorm:"not null;index" json:"applicant_id"`
	Applicant   User  `gorm:"foreignKey:ApplicantID" json:"applicant,omitempty"`
	// Reason 撤稿原因（始终必填）。
	Reason string `gorm:"type:text;not null" json:"reason"`
	// AlternativeNote 替代处理说明：论文处于外审中/修改中时必填。
	AlternativeNote string `gorm:"type:text" json:"alternative_note"`
	// Status pending / approved / rejected。
	Status string `gorm:"size:32;not null;default:pending;index" json:"status"`
	// DecisionComment 编辑处理意见（驳回时必填）。
	DecisionComment string     `gorm:"type:text" json:"decision_comment"`
	ProcessedByID   *uint      `gorm:"index" json:"processed_by_id,omitempty"`
	ProcessedBy     *User      `gorm:"foreignKey:ProcessedByID" json:"processed_by,omitempty"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// WithdrawalFilter 撤稿申请查询过滤条件。
type WithdrawalFilter struct {
	Status  string
	PaperID uint
}
