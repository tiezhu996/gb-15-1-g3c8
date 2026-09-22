package router

import (
	"github.com/gin-gonic/gin"
	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/handler"
	"github.com/paperflow/paperflow/internal/middleware"
)

// RegisterWithdrawalRoutes 论文撤稿申请相关路由。
func RegisterWithdrawalRoutes(g *gin.RouterGroup, h *handler.WithdrawalHandler) {
	// 作者对本人论文发起撤稿申请（管理员代行）。
	g.POST("/papers/:id/withdrawal", middleware.RequireRoles(constants.RoleAuthor, constants.RoleAdmin), h.Apply)
	// 论文最近一次撤稿申请（作者/编辑刷新后查看原因、处理结果与状态）。
	g.GET("/papers/:id/withdrawal", h.GetByPaper)
	// 编辑撤稿处理队列。
	g.GET("/withdrawals", middleware.RequireRoles(constants.RoleEditor, constants.RoleAdmin), h.List)
	// 编辑批准/驳回撤稿申请。
	g.POST("/withdrawals/:id/decision", middleware.RequireRoles(constants.RoleEditor, constants.RoleAdmin), h.Decide)
}
