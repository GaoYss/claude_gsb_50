package warranty

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
)

// Module 质保与责任方管理模块: 供应商档案、灯具/灯杆质保登记、故障责任判定与厂家响应跟踪。
type Module struct {
	repository *Repository
	service    *Service
	handler    *Handler
}

// New 构造质保管理模块, 依赖路灯台账与故障登记的只读仓储。
func New(db *gorm.DB, lamps *lamp.Repository, faults *fault.Repository) *Module {
	repository := NewRepository(db)
	service := NewService(repository, lamps, faults)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
	}
}

// Service 暴露业务服务, 供故障/维修模块联动责任判定与响应跟踪。
func (m *Module) Service() *Service { return m.service }

// Repository 暴露只读仓储, 供状态查询模块聚合质保指标。
func (m *Module) Repository() *Repository { return m.repository }

// Start 启动厂家响应超时的定时巡检。
func (m *Module) Start(ctx context.Context) {
	m.service.StartScheduler(ctx)
}

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "质保与责任方管理" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any {
	return []any{&Supplier{}, &Warranty{}, &WarrantyClaim{}}
}

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	suppliers := api.Group("/warranty/suppliers")
	{
		suppliers.GET("", m.handler.ListSuppliers)
		suppliers.POST("", m.handler.CreateSupplier)
		suppliers.GET("/options", m.handler.SupplierOptions)
		suppliers.GET("/:id", m.handler.GetSupplier)
		suppliers.PUT("/:id", m.handler.UpdateSupplier)
		suppliers.DELETE("/:id", m.handler.DeleteSupplier)
	}

	warranties := api.Group("/warranty/registrations")
	{
		warranties.GET("", m.handler.ListWarranties)
		warranties.POST("", m.handler.CreateWarranty)
		warranties.GET("/:id", m.handler.GetWarranty)
		warranties.PUT("/:id", m.handler.UpdateWarranty)
		warranties.DELETE("/:id", m.handler.DeleteWarranty)
	}

	claims := api.Group("/warranty/claims")
	{
		claims.GET("", m.handler.ListClaims)
		claims.GET("/meta", m.handler.Metadata)
		claims.GET("/overview", m.handler.Overview)
		claims.GET("/index", m.handler.ClaimIndex)
		claims.GET("/by-fault/:faultId", m.handler.ClaimByFault)
		claims.GET("/:id", m.handler.GetClaim)
		claims.POST("/:id/remind", m.handler.Remind)
		claims.POST("/:id/respond", m.handler.MarkResponded)
		claims.POST("/:id/takeover", m.handler.Takeover)
	}
}
