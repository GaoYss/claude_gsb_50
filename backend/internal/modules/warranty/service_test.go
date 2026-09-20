package warranty_test

import (
	"context"
	"fmt"
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

// harness 使用内存数据库装配真实模块, 验证质保登记与责任方判定的跨模块流程。
type harness struct {
	lamps    *lamp.Service
	faults   *fault.Service
	warranty *warranty.Service
	db       *gorm.DB
	supplier *warranty.Supplier
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
		&warranty.Supplier{}, &warranty.Warranty{}, &warranty.FaultAssignment{},
	))

	lampService := lamp.NewService(lamp.NewRepository(db))
	faultService := fault.NewService(fault.NewRepository(db), lampService)
	warrantyService := warranty.NewService(warranty.NewRepository(db), lampService)
	faultService.SetResponsibilityDecider(warrantyService)

	h := &harness{lamps: lampService, faults: faultService, warranty: warrantyService, db: db}

	supplier, err := warrantyService.CreateSupplier(context.Background(), warranty.SupplierCreateRequest{
		Name: "测试照明有限公司", ContactPerson: "测试联系人", ContactPhone: "13800000000",
	})
	require.NoError(t, err)
	h.supplier = supplier
	return h
}

func (h *harness) createLamp(t *testing.T, code string) *lamp.Lamp {
	t.Helper()
	entity, err := h.lamps.Create(context.Background(), lamp.CreateRequest{
		Code: code, RoadName: "测试路", LampType: lamp.LampTypeLED,
	})
	require.NoError(t, err)
	return entity
}

// registerWarranty 登记指定部件的质保, daysFromNow 为质保到期日相对今天的天数。
func (h *harness) registerWarranty(t *testing.T, lampID uint, component string, daysFromNow int) *warranty.Warranty {
	t.Helper()
	start := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	end := time.Now().AddDate(0, 0, daysFromNow).Format("2006-01-02")
	entity, err := h.warranty.CreateWarranty(context.Background(), warranty.WarrantyCreateRequest{
		LampID:     lampID,
		Component:  component,
		SupplierID: &h.supplier.ID,
		StartDate:  start,
		EndDate:    end,
	})
	require.NoError(t, err)
	return entity
}

func (h *harness) createFault(t *testing.T, lampID uint, faultType string) *fault.Fault {
	t.Helper()
	entity, err := h.faults.Create(context.Background(), fault.CreateRequest{
		LampID: lampID, FaultType: faultType, Description: "测试故障",
	})
	require.NoError(t, err)
	return entity
}

func (h *harness) assignmentOf(t *testing.T, faultID uint) *warranty.FaultAssignment {
	t.Helper()
	entity, err := h.warranty.GetAssignmentByFault(context.Background(), faultID)
	require.NoError(t, err)
	return entity
}

// requireConflict 断言错误是 409 业务冲突。
func requireConflict(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	businessErr, ok := apperr.As(err)
	require.True(t, ok, "期望业务错误, 实际: %v", err)
	require.Equal(t, http.StatusConflict, businessErr.Status, "错误信息: %s", businessErr.Message)
}

func TestDecideResponsibilityInWarranty(t *testing.T) {
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-001")
	h.registerWarranty(t, device.ID, warranty.ComponentLuminaire, 365)

	entity := h.createFault(t, device.ID, "灯不亮")

	// 登记故障时自动判定: 质保期内指派厂家
	require.NotNil(t, entity.Responsibility, "登记响应应携带责任方摘要")
	require.Equal(t, warranty.ResponsibleSupplier, entity.Responsibility.ResponsibleType)
	require.True(t, entity.Responsibility.InWarranty)
	require.Equal(t, warranty.ComponentLuminaire, entity.Responsibility.Component)

	assignment := h.assignmentOf(t, entity.ID)
	require.Equal(t, warranty.ResponsibleSupplier, assignment.ResponsibleType)
	require.Equal(t, warranty.AssignPending, assignment.Status)
	require.Equal(t, h.supplier.Name, assignment.SupplierName)
	require.Equal(t, h.supplier.ContactPhone, assignment.ContactPhone)
	require.True(t, assignment.Deadline.After(assignment.AssignedAt))
	require.Contains(t, assignment.Reason, "质保期内")
}

func TestDecideResponsibilityExpiredOrMissing(t *testing.T) {
	h := newHarness(t)

	// 质保已到期 -> 自有班组
	expiredLamp := h.createLamp(t, "LD-W-002")
	h.registerWarranty(t, expiredLamp.ID, warranty.ComponentLuminaire, -10)
	expiredFault := h.createFault(t, expiredLamp.ID, "灯不亮")
	require.NotNil(t, expiredFault.Responsibility)
	require.Equal(t, warranty.ResponsibleOwnTeam, expiredFault.Responsibility.ResponsibleType)
	require.False(t, expiredFault.Responsibility.InWarranty)

	expiredAssignment := h.assignmentOf(t, expiredFault.ID)
	require.Equal(t, warranty.ResponsibleOwnTeam, expiredAssignment.ResponsibleType)
	require.Contains(t, expiredAssignment.Reason, "到期")

	// 未登记质保 -> 自有班组
	noWarrantyLamp := h.createLamp(t, "LD-W-003")
	noWarrantyFault := h.createFault(t, noWarrantyLamp.ID, "灯不亮")
	require.Equal(t, warranty.ResponsibleOwnTeam, noWarrantyFault.Responsibility.ResponsibleType)

	missingAssignment := h.assignmentOf(t, noWarrantyFault.ID)
	require.Contains(t, missingAssignment.Reason, "未登记")
}

func TestDecideResponsibilityPoleComponent(t *testing.T) {
	h := newHarness(t)

	// 只登记灯杆质保, 灯具未登记
	poleLamp := h.createLamp(t, "LD-W-004")
	h.registerWarranty(t, poleLamp.ID, warranty.ComponentPole, 365)

	// 灯杆倾斜 -> 归灯杆部件, 命中灯杆质保
	poleFault := h.createFault(t, poleLamp.ID, "灯杆倾斜")
	require.Equal(t, warranty.ComponentPole, poleFault.Responsibility.Component)
	require.Equal(t, warranty.ResponsibleSupplier, poleFault.Responsibility.ResponsibleType)

	// 灯不亮 -> 归灯具部件, 灯具无质保 -> 自有班组
	luminaireLamp := h.createLamp(t, "LD-W-005")
	h.registerWarranty(t, luminaireLamp.ID, warranty.ComponentPole, 365)
	luminaireFault := h.createFault(t, luminaireLamp.ID, "灯不亮")
	require.Equal(t, warranty.ComponentLuminaire, luminaireFault.Responsibility.Component)
	require.Equal(t, warranty.ResponsibleOwnTeam, luminaireFault.Responsibility.ResponsibleType)
}

func TestOverdueSweepAndTransfer(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-005")
	h.registerWarranty(t, device.ID, warranty.ComponentLuminaire, 365)
	entity := h.createFault(t, device.ID, "灯不亮")

	assignment := h.assignmentOf(t, entity.ID)
	require.Equal(t, warranty.AssignPending, assignment.Status)

	// 将响应时限改到过去, 模拟厂家处理超时
	pastDeadline := time.Now().Add(-time.Hour)
	require.NoError(t, h.db.Model(&warranty.FaultAssignment{}).
		Where("id = ?", assignment.ID).
		Update("deadline", pastDeadline).Error)

	// 查询列表时自动清扫: 厂家超时单被置为超时提醒
	_, _, _, err := h.warranty.ListAssignments(ctx, warranty.AssignmentListQuery{})
	require.NoError(t, err)

	reminded := h.assignmentOf(t, entity.ID)
	require.Equal(t, warranty.AssignOverdue, reminded.Status, "超时厂家单应被自动提醒")
	require.NotNil(t, reminded.RemindedAt, "应记录提醒时间")

	// 厂家处理超时后转自有班组接手
	transferred, err := h.warranty.TransferAssignment(ctx, reminded.ID, warranty.TransferRequest{Remark: "厂家超时, 一班接手"})
	require.NoError(t, err)
	require.Equal(t, warranty.ResponsibleOwnTeam, transferred.ResponsibleType)
	require.Equal(t, warranty.AssignTransferred, transferred.Status)
	require.NotNil(t, transferred.TransferredAt)
	require.True(t, transferred.InWarranty, "转班组后仍属于质保内案件")

	// 重复转派被拒绝
	_, err = h.warranty.TransferAssignment(ctx, reminded.ID, warranty.TransferRequest{})
	requireConflict(t, err)
}

func TestTransferRejectedForOwnTeam(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-006")
	// 未登记质保 -> 自有班组
	entity := h.createFault(t, device.ID, "灯不亮")

	assignment := h.assignmentOf(t, entity.ID)
	_, err := h.warranty.TransferAssignment(ctx, assignment.ID, warranty.TransferRequest{})
	requireConflict(t, err)
}

func TestOverviewStatistics(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)

	// 一盏在保(厂家), 一盏超期(自有班组)
	inWarrantyLamp := h.createLamp(t, "LD-W-007")
	h.registerWarranty(t, inWarrantyLamp.ID, warranty.ComponentLuminaire, 365)
	expiredLamp := h.createLamp(t, "LD-W-008")
	h.registerWarranty(t, expiredLamp.ID, warranty.ComponentLuminaire, -1)

	inWarrantyFault := h.createFault(t, inWarrantyLamp.ID, "灯不亮")
	h.createFault(t, expiredLamp.ID, "灯不亮")

	// 在保故障产生一条维修记录, 用于质保内维修占比
	require.NoError(t, h.db.Create(&repair.Repair{
		RepairNo:  "WX209901010001",
		FaultID:   inWarrantyFault.ID,
		FaultNo:   inWarrantyFault.FaultNo,
		LampID:    inWarrantyLamp.ID,
		LampCode:  inWarrantyLamp.Code,
		Repairman: "测试维修工",
		StartedAt: time.Now(),
		Status:    repair.StatusOngoing,
	}).Error)

	overview, err := h.warranty.Overview(ctx)
	require.NoError(t, err)
	require.EqualValues(t, 1, overview.SupplierTotal)
	require.EqualValues(t, 2, overview.WarrantyTotal)
	require.EqualValues(t, 1, overview.WarrantyByState[warranty.StateActive])
	require.EqualValues(t, 1, overview.WarrantyByState[warranty.StateExpired])
	require.EqualValues(t, 2, overview.AssignmentTotal)
	require.EqualValues(t, 1, overview.SupplierAssigned)
	require.EqualValues(t, 1, overview.OwnTeamAssigned)
	require.EqualValues(t, 1, overview.RepairTotal)
	require.EqualValues(t, 1, overview.InWarrantyRepair)
	require.InDelta(t, 100.0, overview.InWarrantyRatio, 0.01)
	require.EqualValues(t, 0, overview.OverdueTotal)
}

func TestWarrantyValidation(t *testing.T) {
	ctx := context.Background()
	h := newHarness(t)
	device := h.createLamp(t, "LD-W-009")
	h.registerWarranty(t, device.ID, warranty.ComponentLuminaire, 365)

	// 同一路灯同一部件重复登记 -> 409
	_, err := h.warranty.CreateWarranty(ctx, warranty.WarrantyCreateRequest{
		LampID: device.ID, Component: warranty.ComponentLuminaire,
		StartDate: "2026-01-01", EndDate: "2027-01-01",
	})
	requireConflict(t, err)

	// 结束日期早于开始日期 -> 400
	_, err = h.warranty.CreateWarranty(ctx, warranty.WarrantyCreateRequest{
		LampID: device.ID, Component: warranty.ComponentPole,
		StartDate: "2027-01-01", EndDate: "2026-01-01",
	})
	businessErr, ok := apperr.As(err)
	require.True(t, ok)
	require.Equal(t, http.StatusBadRequest, businessErr.Status)

	// 供应商被质保引用时不允许删除 -> 409
	requireConflict(t, h.warranty.DeleteSupplier(ctx, h.supplier.ID))

	// 供应商重名 -> 409
	_, err = h.warranty.CreateSupplier(ctx, warranty.SupplierCreateRequest{Name: h.supplier.Name})
	requireConflict(t, err)
}

func TestComponentRule(t *testing.T) {
	require.Equal(t, warranty.ComponentPole, warranty.ComponentForFaultType("灯杆倾斜"))
	for _, faultType := range []string{"灯不亮", "灯光闪烁", "灯具常亮", "线路故障", "控制箱故障", "灯具破损", "其他"} {
		require.Equal(t, warranty.ComponentLuminaire, warranty.ComponentForFaultType(faultType),
			fmt.Sprintf("故障类型 %s 应归灯具部件", faultType))
	}
}
