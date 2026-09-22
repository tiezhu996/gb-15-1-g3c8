package router

import (
	"github.com/gin-gonic/gin"
	"github.com/paperflow/paperflow/internal/constants"
	"github.com/paperflow/paperflow/internal/handler"
	"github.com/paperflow/paperflow/internal/middleware"
)

// RegisterWithdrawalRoutes 撤稿申请相关路由。
func RegisterWithdrawalRoutes(g *gin.RouterGroup, h *handler.WithdrawalHandler) {
	g.POST("/papers/:id/withdrawals", middleware.RequireRoles(constants.RoleAuthor, constants.RoleAdmin), h.Apply)
	g.GET("/papers/:id/withdrawals", h.ListByPaper)
	g.GET("/withdrawals/mine", middleware.RequireRoles(constants.RoleAuthor, constants.RoleAdmin), h.ListMine)
	g.GET("/withdrawals", middleware.RequireRoles(constants.RoleEditor, constants.RoleAdmin), h.List)
	g.POST("/withdrawals/:id/process", middleware.RequireRoles(constants.RoleEditor, constants.RoleAdmin), h.Process)
}
