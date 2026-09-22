package model

import "time"

// WithdrawalRequest 论文撤稿申请：作者发起、编辑审批的终态保护流程。
type WithdrawalRequest struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	PaperID         uint       `gorm:"not null;index" json:"paper_id"`
	Paper           Paper      `gorm:"foreignKey:PaperID" json:"paper,omitempty"`
	ApplicantID     uint       `gorm:"not null;index" json:"applicant_id"`
	Applicant       User       `gorm:"foreignKey:ApplicantID" json:"applicant,omitempty"`
	Reason          string     `gorm:"type:text;not null" json:"reason"`
	AltHandlingNote string     `gorm:"type:text" json:"alt_handling_note"`
	Status          string     `gorm:"size:32;not null;default:pending;index" json:"status"`
	ProcessedByID   *uint      `json:"processed_by_id"`
	ProcessedBy     *User      `gorm:"foreignKey:ProcessedByID" json:"processed_by,omitempty"`
	ProcessResult   string     `gorm:"type:text" json:"process_result"`
	ProcessedAt     *time.Time `json:"processed_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// WithdrawalFilter 撤稿申请查询过滤条件。
type WithdrawalFilter struct {
	Status      string
	ApplicantID uint
	PaperID     uint
}
