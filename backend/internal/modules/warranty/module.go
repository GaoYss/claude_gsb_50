package warranty

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Module 质保与责任方模块, 负责供应商档案、灯具/灯杆质保登记与故障责任方判定。
type Module struct {
	repository *Repository
	service    *Service
	handler    *Handler
}

// New 构造质保与责任方模块, lamps 为路灯台账模块提供的端口实现。
func New(db *gorm.DB, lamps LampPort) *Module {
	repository := NewRepository(db)
	service := NewService(repository, lamps)
	return &Module{
		repository: repository,
		service:    service,
		handler:    NewHandler(service),
	}
}

// Service 暴露业务服务, 供故障模块装配责任方判定端口。
func (m *Module) Service() *Service { return m.service }

// Name 实现 module.Module 接口。
func (m *Module) Name() string { return "质保与责任方" }

// Models 实现 module.Module 接口。
func (m *Module) Models() []any {
	return []any{&Supplier{}, &Warranty{}, &FaultAssignment{}}
}

// RegisterRoutes 实现 module.Module 接口。
func (m *Module) RegisterRoutes(api *gin.RouterGroup) {
	group := api.Group("/warranties")
	{
		group.GET("/suppliers", m.handler.ListSuppliers)
		group.POST("/suppliers", m.handler.CreateSupplier)
		group.GET("/suppliers/options", m.handler.SupplierOptions)
		group.PUT("/suppliers/:id", m.handler.UpdateSupplier)
		group.DELETE("/suppliers/:id", m.handler.DeleteSupplier)

		group.GET("/assignments", m.handler.ListAssignments)
		group.GET("/assignments/fault/:faultId", m.handler.GetAssignmentByFault)
		group.POST("/assignments/:id/transfer", m.handler.TransferAssignment)

		group.GET("/overview", m.handler.Overview)
		group.GET("/meta", m.handler.Metadata)

		group.GET("", m.handler.ListWarranties)
		group.POST("", m.handler.CreateWarranty)
		group.GET("/:id", m.handler.GetWarranty)
		group.PUT("/:id", m.handler.UpdateWarranty)
		group.DELETE("/:id", m.handler.DeleteWarranty)
	}
}
