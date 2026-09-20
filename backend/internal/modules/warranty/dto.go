package warranty

import (
	"time"

	"streetlight/pkg/pagination"
)

// SupplierCreateRequest 新增供应商请求。
type SupplierCreateRequest struct {
	Name          string `json:"name" binding:"required,max=128"`
	ContactPerson string `json:"contact_person" binding:"max=64"`
	ContactPhone  string `json:"contact_phone" binding:"max=32"`
	Email         string `json:"email" binding:"omitempty,email,max=128"`
	Address       string `json:"address" binding:"max=255"`
	Remark        string `json:"remark" binding:"max=255"`
}

// SupplierUpdateRequest 更新供应商请求, 指针字段用于区分"未提交"与"置空"。
type SupplierUpdateRequest struct {
	Name          *string `json:"name" binding:"omitempty,max=128"`
	ContactPerson *string `json:"contact_person" binding:"omitempty,max=64"`
	ContactPhone  *string `json:"contact_phone" binding:"omitempty,max=32"`
	Email         *string `json:"email" binding:"omitempty,email,max=128"`
	Address       *string `json:"address" binding:"omitempty,max=255"`
	Remark        *string `json:"remark" binding:"omitempty,max=255"`
}

// SupplierListQuery 供应商列表查询条件。
type SupplierListQuery struct {
	pagination.Params
	Keyword string `form:"keyword"` // 名称 / 联系人 / 电话
}

// WarrantyCreateRequest 登记质保请求, 一盏路灯的一个部件仅允许一条质保。
type WarrantyCreateRequest struct {
	LampID        uint   `json:"lamp_id" binding:"required"`
	Component     string `json:"component" binding:"required,oneof=luminaire pole"`
	SupplierID    *uint  `json:"supplier_id"`
	ContactPerson string `json:"contact_person" binding:"max=64"`
	ContactPhone  string `json:"contact_phone" binding:"max=32"`
	StartDate     string `json:"start_date" binding:"required,datetime=2006-01-02"`
	EndDate       string `json:"end_date" binding:"required,datetime=2006-01-02"`
	Remark        string `json:"remark" binding:"max=255"`
}

// WarrantyUpdateRequest 更新质保请求。
type WarrantyUpdateRequest struct {
	Component     *string `json:"component" binding:"omitempty,oneof=luminaire pole"`
	SupplierID    *uint   `json:"supplier_id"`
	ContactPerson *string `json:"contact_person" binding:"omitempty,max=64"`
	ContactPhone  *string `json:"contact_phone" binding:"omitempty,max=32"`
	StartDate     *string `json:"start_date" binding:"omitempty,datetime=2006-01-02"`
	EndDate       *string `json:"end_date" binding:"omitempty,datetime=2006-01-02"`
	Remark        *string `json:"remark" binding:"omitempty,max=255"`
}

// WarrantyListQuery 质保登记列表查询条件。
type WarrantyListQuery struct {
	pagination.Params
	Keyword   string `form:"keyword"`   // 路灯编号 / 供应商名称
	Component string `form:"component"` // 部件: luminaire / pole
	State     string `form:"state"`     // 质保状态: active / expiring / expired
}

// AssignmentListQuery 责任判定单列表查询条件。
type AssignmentListQuery struct {
	pagination.Params
	Keyword         string `form:"keyword"` // 故障单号 / 路灯编号 / 供应商
	ResponsibleType string `form:"responsible_type"`
	Status          string `form:"status"`
	Component       string `form:"component"`
}

// TransferRequest 厂家超时后转自有班组接手请求。
type TransferRequest struct {
	Remark string `json:"remark" binding:"max=255"`
}

// Meta 质保模块字典, 供前端渲染下拉框。
type Meta struct {
	Components         []string `json:"components"`
	ResponsibleTypes   []string `json:"responsible_types"`
	AssignmentStatuses []string `json:"assignment_statuses"`
	ResponseTimeoutHr  float64  `json:"response_timeout_hours"`
}

// Overview 质保与责任方概览。
type Overview struct {
	SupplierTotal      int64             `json:"supplier_total"`
	WarrantyTotal      int64             `json:"warranty_total"`
	WarrantyByState    map[string]int64  `json:"warranty_by_state"`
	AssignmentTotal    int64             `json:"assignment_total"`
	SupplierAssigned   int64             `json:"supplier_assigned"` // 质保期内指派厂家
	OwnTeamAssigned    int64             `json:"own_team_assigned"` // 无质保或超期转自有班组
	OverdueTotal       int64             `json:"overdue_total"`     // 厂家响应超时(已自动提醒)
	TransferredTotal   int64             `json:"transferred_total"` // 超时后转自有班组
	RepairTotal        int64             `json:"repair_total"`
	InWarrantyRepair   int64             `json:"in_warranty_repair_total"`
	InWarrantyRatio    float64           `json:"in_warranty_repair_ratio"` // 质保内维修占比(%)
	ResponseTimeoutHr  float64           `json:"response_timeout_hours"`
	OverdueAssignments []FaultAssignment `json:"overdue_assignments"` // 厂家超时清单
	GeneratedAt        time.Time         `json:"generated_at"`
}
