package constants

// 撤稿申请状态枚举：待处理→编辑批准/驳回（终态）。
const (
	WithdrawalStatusPending  = "pending"
	WithdrawalStatusApproved = "approved"
	WithdrawalStatusRejected = "rejected"
)

// WithdrawalStatusList 全部撤稿申请状态。
var WithdrawalStatusList = []string{
	WithdrawalStatusPending,
	WithdrawalStatusApproved,
	WithdrawalStatusRejected,
}

// 撤稿申请发起时要求填写「替代处理说明」的论文状态：外审中、修改中。
// 已提交/初审中仅需填写撤稿原因。
var WithdrawalAlternativeRequiredStatuses = []string{
	PaperStatusExternalReview,
	PaperStatusRevision,
}
