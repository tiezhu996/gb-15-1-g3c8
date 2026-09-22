package constants

// 审稿任务状态枚举。
const (
	ReviewStatusInvited   = "invited"
	ReviewStatusAccepted  = "accepted"
	ReviewStatusDeclined  = "declined"
	ReviewStatusCompleted = "completed"
	// ReviewStatusClosed 审稿因论文撤稿批准而被系统关闭（不再可推进，但记录保留可查看）。
	ReviewStatusClosed = "closed"
)

// ReviewStatusList 全部审稿状态。
var ReviewStatusList = []string{
	ReviewStatusInvited,
	ReviewStatusAccepted,
	ReviewStatusDeclined,
	ReviewStatusCompleted,
	ReviewStatusClosed,
}

// ReviewStatusOpen 尚未完成、撤稿批准时需要一并关闭的审稿状态。
var ReviewStatusOpen = []string{
	ReviewStatusInvited,
	ReviewStatusAccepted,
}
