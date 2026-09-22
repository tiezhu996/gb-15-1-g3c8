package constants

// 撤稿申请状态机枚举：待处理→已批准/已驳回。
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
