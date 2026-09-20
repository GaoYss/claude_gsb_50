package warranty_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/internal/modules/repair"
	"streetlight/internal/modules/warranty"
)

// harness 使用内存数据库装配真实模块, 验证质保责任判定与厂家响应流转。
type harness struct {
	db        *gorm.DB
	lamps     *lamp.Service
	faults    *fault.Service
	repairs   *repair.Service
	warrantys *warranty.Service
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{
		Logger:         logger.Default.LogMode(logger.Silent),
		NamingStrategy: schema.NamingStrategy{SingularTable: true},
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	require.NoError(t, db.AutoMigrate(
		&lamp.Lamp{}, &fault.Fault{}, &repair.Repair{},
		&warranty.Supplier{}, &warranty.Warranty{}, &warranty.WarrantyClaim{},
	))

	lampRepository := lamp.NewRepository(db)
	lampService := lamp.NewService(lampRepository)

	faultRepository := fault.NewRepository(db)
	faultService := fault.NewService(faultRepository, lampService)
	lampService.SetOpenFaultCounter(faultRepository)

	repairRepository := repair.NewRepository(db)
	repairService := repair.NewService(repairRepository, faultService)

	warrantyRepository := warranty.NewRepository(db)
	warrantyService := warranty.NewService(warrantyRepository, lampRepository, faultRepository)
	faultService.SetWarrantyHook(warrantyService)

	return &harness{
		db:        db,
		lamps:     lampService,
		faults:    faultService,
		repairs:   repairService,
		warrantys: warrantyService,
	}
}

func (h *harness) createLamp(t *testing.T, code string) *lamp.Lamp {
	t.Helper()
	entity, err := h.lamps.Create(context.Background(), lamp.CreateRequest{
		Code: code, Name: "测试灯杆", RoadName: "测试路", LampType: lamp.LampTypeLED,
	})
	require.NoError(t, err)
	return entity
}

func (h *harness) createSupplier(t *testing.T, name string, deadlineHours int) *warranty.Supplier {
	t.Helper()
	hours := deadlineHours
	entity, err := h.warrantys.CreateSupplier(context.Background(), warranty.SupplierUpsertRequest{
		Name:                  name,
		ContactPerson:         "厂家联系人",
		ContactPhone:          "13700000000",
		ServicePhone:          "400-000-0000",
		ResponseDeadlineHours: &hours,
	})
	require.NoError(t, err)
	return entity
}

// registerWarranty 为路灯登记灯具质保, start/end 为相对当前时刻的天数偏移。
func (h *harness) registerWarranty(t *testing.T, lampID, supplierID uint, startOffset, endOffset int, withPole bool) {
	t.Helper()
	ctx := context.Background()
	now := time.Now()
	req := warranty.WarrantyUpsertRequest{
		LampID:             lampID,
		LampSupplierID:     &supplierID,
		LampStartAt:        now.AddDate(0, 0, startOffset).Format("2006-01-02"),
		LampEndAt:          now.AddDate(0, 0, endOffset).Format("2006-01-02"),
		LampWarrantyMonths: intPtr(24),
	}
	if withPole {
		req.PoleSupplierID = &supplierID
		req.PoleStartAt = now.AddDate(0, 0, startOffset).Format("2006-01-02")
		req.PoleEndAt = now.AddDate(0, 0, endOffset+365).Format("2006-01-02")
	}
	_, err := h.warrantys.UpsertWarranty(ctx, req)
	require.NoError(t, err)
}

func (h *harness) createFault(t *testing.T, lampID uint, faultType string) *fault.Fault {
	t.Helper()
	entity, err := h.faults.Create(context.Background(), fault.CreateRequest{
		LampID: lampID, FaultType: faultType, FaultLevel: fault.LevelHigh,
		Source: fault.SourceInspection, Description: "质保模块测试", Reporter: "巡检员",
	})
	require.NoError(t, err)
	return entity
}

func intPtr(value int) *int    { return &value }
func uintPtr(value uint) *uint { return &value }

func TestDecidePartyWithinWarrantyAssignsManufacturer(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-001")
	supplier := h.createSupplier(t, "在保灯具厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -100, 265, false)

	entity := h.createFault(t, device.ID, "灯不亮")

	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.NotNil(t, claim)
	require.Equal(t, warranty.ComponentLamp, claim.Component)
	require.True(t, claim.InWarranty)
	require.Equal(t, warranty.PartyManufacturer, claim.PartyType)
	require.Equal(t, warranty.ClaimStatusPending, claim.Status)
	require.Equal(t, supplier.ID, *claim.SupplierID)
	require.Equal(t, "13700000000", claim.ContactPhone)
	require.Equal(t, 24, claim.ResponseDeadlineHours)
}

func TestDecidePartyExpiredWarrantyGoesInternal(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-002")
	supplier := h.createSupplier(t, "超期灯具厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -800, -70, false)

	entity := h.createFault(t, device.ID, "灯光闪烁")

	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.NotNil(t, claim)
	require.False(t, claim.InWarranty)
	require.Equal(t, warranty.PartyInternal, claim.PartyType)
	require.Nil(t, claim.SupplierID)
	require.Equal(t, warranty.ClaimStatusInternal, claim.Status, "超期故障应直接转自有班组处理中")
}

func TestNonWarrantyComponentAlwaysInternal(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-003")
	supplier := h.createSupplier(t, "在保厂家三号", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -10, 700, false)

	// 线路故障不属于灯具/灯杆质保部件, 即使灯具在保也由自有班组负责。
	entity := h.createFault(t, device.ID, "线路故障")

	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.NotNil(t, claim)
	require.Equal(t, "", claim.Component)
	require.Equal(t, warranty.PartyInternal, claim.PartyType)
	require.False(t, claim.InWarranty)
	require.Equal(t, warranty.ClaimStatusInternal, claim.Status)
}

func TestPoleFaultUsesPoleWarranty(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-004")
	supplier := h.createSupplier(t, "灯杆厂家", 48)

	now := time.Now()
	_, err := h.warrantys.UpsertWarranty(ctx, warranty.WarrantyUpsertRequest{
		LampID: device.ID,
		// 灯具已超期, 灯杆仍在保。
		LampSupplierID: uintPtr(supplier.ID),
		LampStartAt:    now.AddDate(0, 0, -800).Format("2006-01-02"),
		LampEndAt:      now.AddDate(0, 0, -70).Format("2006-01-02"),
		PoleSupplierID: uintPtr(supplier.ID),
		PoleStartAt:    now.AddDate(0, 0, -200).Format("2006-01-02"),
		PoleEndAt:      now.AddDate(0, 0, 1500).Format("2006-01-02"),
	})
	require.NoError(t, err)

	entity := h.createFault(t, device.ID, "灯杆倾斜")
	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.ComponentPole, claim.Component)
	require.True(t, claim.InWarranty, "灯杆在保时灯杆故障应由厂家承担")
	require.Equal(t, warranty.PartyManufacturer, claim.PartyType)
	require.Equal(t, 48, claim.ResponseDeadlineHours)
}

func TestRepairStartMarksManufacturerResponded(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-005")
	supplier := h.createSupplier(t, "响应厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -50, 300, false)

	entity := h.createFault(t, device.ID, "灯具破损")
	claimBefore, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.ClaimStatusPending, claimBefore.Status)
	require.Nil(t, claimBefore.RespondedAt)

	// 厂家(维修人员)开工, 责任工单自动进入厂家处理中。
	_, err = h.repairs.Create(ctx, repair.CreateRequest{
		FaultID: entity.ID, Repairman: "厂家维修工", RepairTeam: supplier.Name,
	})
	require.NoError(t, err)

	claimAfter, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.ClaimStatusProcessing, claimAfter.Status)
	require.NotNil(t, claimAfter.RespondedAt)
}

func TestSweepOverdueMarksOverdueAndReminds(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-006")
	supplier := h.createSupplier(t, "超时厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -100, 265, false)

	entity := h.createFault(t, device.ID, "灯不亮")
	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.ClaimStatusPending, claim.Status)
	require.False(t, claim.Overdue, "刚登记不应超时")

	// 概览中超时数量为 0。
	overview, err := h.warrantys.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), overview.ResponseOverdue)

	// 将指派时间回拨到 25 小时前, 超过 24 小时承诺时限。
	require.NoError(t, h.db.Model(&warranty.WarrantyClaim{}).
		Where("id = ?", claim.ID).
		Update("assigned_at", time.Now().Add(-25*time.Hour)).Error)

	overdue, reminded, err := h.warrantys.SweepOverdue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, overdue)
	require.Equal(t, 1, reminded, "首次超时应自动提醒一次")

	updated, err := h.warrantys.GetClaim(ctx, claim.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.ClaimStatusOverdue, updated.Status)
	require.Equal(t, 1, updated.RemindCount)
	require.True(t, updated.Overdue)

	// 再次巡检, 冷却期内不重复提醒。
	overdue2, reminded2, err := h.warrantys.SweepOverdue(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, overdue2)
	require.Equal(t, 0, reminded2)

	overview, err = h.warrantys.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), overview.ResponseOverdue, "概览应统计厂家响应超时数量")
}

func TestTakeoverTransfersToInternalTeam(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-007")
	supplier := h.createSupplier(t, "待转办厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -100, 265, false)

	entity := h.createFault(t, device.ID, "灯不亮")
	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)

	taken, err := h.warrantys.Takeover(ctx, claim.ID, warranty.TakeoverRequest{
		Team: "市政照明一班", By: "调度员", Reason: "厂家超时未到场, 应急接管",
	})
	require.NoError(t, err)
	require.Equal(t, warranty.ClaimStatusTakenOver, taken.Status)
	require.Equal(t, "市政照明一班", taken.TakeoverTeam)
	require.NotNil(t, taken.TakeoverAt)

	// 已转办后不能再次催办或转办。
	_, err = h.warrantys.Remind(ctx, claim.ID, warranty.RemindRequest{})
	requireClaimConflict(t, err)
	_, err = h.warrantys.Takeover(ctx, claim.ID, warranty.TakeoverRequest{})
	requireClaimConflict(t, err)

	overview, err := h.warrantys.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), overview.TakenOverTotal)
	require.Equal(t, int64(0), overview.ManufacturerOpen, "转办后不再计入厂家未闭环")
}

func TestRemindRejectedForInternalClaim(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-008")
	supplier := h.createSupplier(t, "无关厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -800, -70, false)

	entity := h.createFault(t, device.ID, "灯不亮")
	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.PartyInternal, claim.PartyType)

	_, err = h.warrantys.Remind(ctx, claim.ID, warranty.RemindRequest{})
	requireClaimConflict(t, err)
}

func TestOverviewComputesInWarrantyRate(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	inWarrantyLamp := h.createLamp(t, "LD-W-101")
	expiredLamp := h.createLamp(t, "LD-W-102")
	noWarrantyLamp := h.createLamp(t, "LD-W-103")
	supplier := h.createSupplier(t, "概览厂家", 24)

	h.registerWarranty(t, inWarrantyLamp.ID, supplier.ID, -100, 265, false)
	h.registerWarranty(t, expiredLamp.ID, supplier.ID, -800, -70, false)

	h.createFault(t, inWarrantyLamp.ID, "灯不亮")  // 质保内 -> 厂家
	h.createFault(t, expiredLamp.ID, "灯不亮")     // 超期 -> 自有班组
	h.createFault(t, noWarrantyLamp.ID, "线路故障") // 未登记/非部件 -> 自有班组

	overview, err := h.warrantys.Overview(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), overview.ClaimTotal)
	require.Equal(t, int64(1), overview.InWarrantyTotal)
	require.Equal(t, int64(2), overview.OutWarrantyTotal)
	require.InDelta(t, 0.333, overview.InWarrantyRate, 0.001, "质保内维修占比应为 1/3")
	require.Equal(t, int64(2), overview.RegisteredLamps)
	require.Equal(t, int64(1), overview.LampActive)
}

func TestFaultCloseClosesClaim(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-201")
	supplier := h.createSupplier(t, "闭环厂家", 24)
	h.registerWarranty(t, device.ID, supplier.ID, -100, 265, false)

	entity := h.createFault(t, device.ID, "灯不亮")

	// 厂家开工并修复。
	record, err := h.repairs.Create(ctx, repair.CreateRequest{FaultID: entity.ID, Repairman: "厂家维修工"})
	require.NoError(t, err)
	_, err = h.repairs.Finish(ctx, record.ID, repair.FinishRequest{Result: repair.ResultFixed})
	require.NoError(t, err)
	_, err = h.faults.Close(ctx, entity.ID, fault.CloseRequest{Remark: "闭环"})
	require.NoError(t, err)

	claim, err := h.warrantys.GetClaimByFault(ctx, entity.ID)
	require.NoError(t, err)
	require.Equal(t, warranty.ClaimStatusClosed, claim.Status)
	require.NotNil(t, claim.ClosedAt)
}

func TestWarrantyEndDateInclusive(t *testing.T) {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	start := today.AddDate(0, 0, -30)
	end := today // 截止日当天应仍在质保期内

	registration := &warranty.Warranty{}
	supplierID := uint(1)
	registration.LampSupplierID = &supplierID
	registration.LampStartAt = &start
	registration.LampEndAt = &end

	decision := registration.Decide(warranty.ComponentLamp, now.Add(-2*time.Hour))
	require.True(t, decision.InWarranty, "质保截止日当天应计入质保")
	require.Equal(t, warranty.PartyManufacturer, decision.PartyType)

	// 次日超期。
	nextDay := registration.Decide(warranty.ComponentLamp, now.Add(26*time.Hour))
	require.False(t, nextDay.InWarranty)
	require.Equal(t, warranty.PartyInternal, nextDay.PartyType)
}

func TestSupplierLifecycleValidation(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	supplier, err := h.warrantys.CreateSupplier(ctx, warranty.SupplierUpsertRequest{Name: "唯一供应商"})
	require.NoError(t, err)
	require.Equal(t, warranty.DefaultResponseDeadlineHours, supplier.ResponseDeadlineHours, "未填时限应取默认值")

	// 名称重复返回 409。
	_, err = h.warrantys.CreateSupplier(ctx, warranty.SupplierUpsertRequest{Name: "唯一供应商"})
	requireClaimConflict(t, err)

	// 被质保登记引用时拒绝删除。
	device := h.createLamp(t, "LD-W-301")
	h.registerWarranty(t, device.ID, supplier.ID, -10, 300, false)
	err = h.warrantys.DeleteSupplier(ctx, supplier.ID)
	requireClaimConflict(t, err)
}

// requireClaimConflict 断言错误是 409 业务冲突。
func requireClaimConflict(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	businessErr, ok := apperr.As(err)
	require.True(t, ok, "期望业务错误, 实际: %v", err)
	require.Equal(t, http.StatusConflict, businessErr.Status, "错误信息: %s", businessErr.Message)
}
