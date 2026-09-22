package constants

// 论文状态机枚举：已提交→初审中→外审中→修改中→已录用/已拒稿；作者撤稿批准后进入已撤稿终态。
const (
	PaperStatusSubmitted      = "submitted"
	PaperStatusInitialReview  = "initial_review"
	PaperStatusExternalReview = "external_review"
	PaperStatusRevision       = "revision"
	PaperStatusAccepted       = "accepted"
	PaperStatusRejected       = "rejected"
	PaperStatusWithdrawn      = "withdrawn"
)

// PaperStatusList 全部论文状态。
var PaperStatusList = []string{
	PaperStatusSubmitted,
	PaperStatusInitialReview,
	PaperStatusExternalReview,
	PaperStatusRevision,
	PaperStatusAccepted,
	PaperStatusRejected,
	PaperStatusWithdrawn,
}

// PaperStatusInProcess 审稿流程进行中（作者可发起撤稿）的状态集合：已录用不可撤稿，已拒稿/已撤稿为终态。
var PaperStatusInProcess = []string{
	PaperStatusSubmitted,
	PaperStatusInitialReview,
	PaperStatusExternalReview,
	PaperStatusRevision,
}
