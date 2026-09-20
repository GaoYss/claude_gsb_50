package bootstrap

import (
	"gorm.io/gorm"

	"streetlight/internal/module"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
	"streetlight/internal/modules/status"
	"streetlight/internal/modules/warranty"
)

// buildModules 按依赖方向装配业务模块。
//
// 依赖关系: 路灯台账 <- 故障登记 <- 维修记录, 维修状态查询依赖三者的只读仓储。
// 其中 "删除路灯前校验未闭环故障" 需要路灯模块反向调用故障模块,
// 因此通过 SetOpenFaultCounter 在构造完成后回填, 避免循环构造依赖。
//
// 质保与责任方管理模块读取路灯/故障仓储完成责任判定, 故障模块通过
// SetWarrantyHook 反向接收责任判定与厂家响应联动事件, 同样避免构造循环。
func buildModules(db *gorm.DB) []module.Module {
	lampModule := lamp.New(db)

	faultModule := fault.New(db, lampModule.Service())
	lampModule.Service().SetOpenFaultCounter(faultModule.Repository())

	repairModule := repair.New(db, faultModule.Service())

	warrantyModule := warranty.New(db, lampModule.Repository(), faultModule.Repository())
	faultModule.Service().SetWarrantyHook(warrantyModule.Service())

	statusModule := status.New(
		db,
		lampModule.Repository(),
		faultModule.Repository(),
		repairModule.Repository(),
		warrantyModule.Repository(),
	)

	return []module.Module{
		lampModule,
		faultModule,
		repairModule,
		warrantyModule,
		statusModule,
	}
}
