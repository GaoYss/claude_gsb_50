package warranty

import (
	"time"

	"streetlight/pkg/pagination"
)

// SupplierUpsertRequest 新增 / 修改供应商请求。
type SupplierUpsertRequest struct {
	Name                  string `json:"name" binding:"required,max=128"`
	ShortName             string `json:"short_name" binding:"max=64"`
	ContactPerson         string `json:"contact_person" binding:"max=64"`
	ContactPhone          string `json:"contact_phone" binding:"max=32"`
	ServicePhone          string `json:"service_phone" binding:"max=32"`
	Address               string `json:"address" binding:"max=255"`
	ResponseDeadlineHours *int   `json:"response_deadline_hours" binding:"omitempty,min=1,max=720"`
	Remark                string `json:"remark" binding:"max=255"`
}

// SupplierListQuery 供应商查询条件。
type SupplierListQuery struct {
	pagination.Params
	Keyword string `form:"keyword"` // 名称 / 简称 / 联系人 / 电话
}

// WarrantyUpsertRequest 质保登记请求, 灯具与灯杆信息均选填, 至少登记一项。
type WarrantyUpsertRequest struct {
	LampID uint `json:"lamp_id" binding:"required"`

	LampSupplierID     *uint  `json:"lamp_supplier_id"`
	LampStartAt        string `json:"lamp_start_at" binding:"omitempty,max=16"`
	LampEndAt          string `json:"lamp_end_at" binding:"omitempty,max=16"`
	LampWarrantyMonths *int   `json:"lamp_warranty_months" binding:"omitempty,min=0,max=240"`

	PoleSupplierID     *uint  `json:"pole_supplier_id"`
	PoleStartAt        string `json:"pole_start_at" binding:"omitempty,max=16"`
	PoleEndAt          string `json:"pole_end_at" binding:"omitempty,max=16"`
	PoleWarrantyMonths *int   `json:"pole_warranty_months" binding:"omitempty,min=0,max=240"`

	Remark string `json:"remark" binding:"max=255"`
}

// WarrantyListQuery 质保登记查询条件。
type WarrantyListQuery struct {
	pagination.Params
	Keyword       string `form:"keyword"` // 路灯编号 / 道路 / 供应商
	SupplierID    uint   `form:"supplier_id"`
	Component     string `form:"component"`      // lamp / pole
	WarrantyState string `form:"warranty_state"` // active 质保中 / expired 已超期 / expiring 30 天内到期 / unregistered 未登记
	RoadName      string `form:"road_name"`
}

// ClaimListQuery 责任工单查询条件。
type ClaimListQuery struct {
	pagination.Params
	Keyword    string `form:"keyword"` // 故障单号 / 路灯编号 / 道路 / 供应商
	Status     string `form:"status"`
	PartyType  string `form:"party_type"`
	Component  string `form:"component"`
	SupplierID uint   `form:"supplier_id"`
	InWarranty *bool  `form:"in_warranty"`
	OnlyOpen   bool   `form:"only_open"`
}

// TakeoverRequest 将厂家工单转由自有班组接手的请求。
type TakeoverRequest struct {
	Team   string `json:"team" binding:"max=64"`
	By     string `json:"by" binding:"max=64"`
	Reason string `json:"reason" binding:"omitempty,max=255"`
}

// RemindRequest 手工催办厂家请求。
type RemindRequest struct {
	Remark string `json:"remark" binding:"max=255"`
}

// ComponentMeta 质保模块字典。
type Meta struct {
	Components        []string `json:"components"`
	Parties           []string `json:"parties"`
	ClaimStatuses     []string `json:"claim_statuses"`
	DefaultDeadlineHr int      `json:"default_response_deadline_hours"`
}

// SupplierOption 供应商下拉选项。
type SupplierOption struct {
	ID                    uint   `json:"id"`
	Name                  string `json:"name"`
	ShortName             string `json:"short_name"`
	ContactPerson         string `json:"contact_person"`
	ContactPhone          string `json:"contact_phone"`
	ServicePhone          string `json:"service_phone"`
	ResponseDeadlineHours int    `json:"response_deadline_hours"`
}

// WarrantyState 是列表中用于前端展示的质保状态派生值, 不落库。
type WarrantyState struct {
	Lamp string `json:"lamp"` // active / expired / unregistered
	Pole string `json:"pole"`
}

// WarrantyRow 质保登记列表行, 附带供应商联系方式与派生状态。
type WarrantyRow struct {
	Warranty
	LampName          string          `json:"lamp_name"`
	LampSupplier      *SupplierOption `json:"lamp_supplier,omitempty"`
	PoleSupplier      *SupplierOption `json:"pole_supplier,omitempty"`
	State             WarrantyState   `json:"state"`
	LampRemainingDays int             `json:"lamp_remaining_days"`
	PoleRemainingDays int             `json:"pole_remaining_days"`
}

// ClaimRow 责任工单列表行, 附带派生的超时与等待信息。
type ClaimRow struct {
	WarrantyClaim
	Supplier          *SupplierOption `json:"supplier,omitempty"`
	WaitingHours      float64         `json:"waiting_hours"`
	Overdue           bool            `json:"overdue"`
	ResponseUsedHours float64         `json:"response_used_hours"` // 厂家实际响应耗时; 未响应为已等待时长
	CurrentParty      string          `json:"current_party"`       // manufacturer / internal
	FaultType         string          `json:"fault_type"`
	FaultLevel        string          `json:"fault_level"`
	FaultStatus       string          `json:"fault_status"`
}

// Overview 质保责任概览。
type Overview struct {
	// 登记覆盖。
	RegisteredLamps int64 `json:"registered_lamps"` // 已登记质保的路灯数
	WarrantyActive  int64 `json:"warranty_active"`  // 灯具或灯杆至少一项仍在质保期内的路灯数
	LampActive      int64 `json:"lamp_active"`
	PoleActive      int64 `json:"pole_active"`
	ExpiringSoon    int64 `json:"expiring_soon"` // 30 天内到期
	SupplierTotal   int64 `json:"supplier_total"`

	// 责任工单。
	ClaimTotal       int64            `json:"claim_total"`
	InWarrantyTotal  int64            `json:"in_warranty_total"`  // 质保内责任工单(厂家)
	OutWarrantyTotal int64            `json:"out_warranty_total"` // 超期/非质保(自有班组)
	InWarrantyRate   float64          `json:"in_warranty_rate"`   // 质保内维修占比
	ManufacturerOpen int64            `json:"manufacturer_open"`  // 厂家未闭环
	ResponseOverdue  int64            `json:"response_overdue"`   // 厂家响应超时数量
	TakenOverTotal   int64            `json:"taken_over_total"`   // 累计转自有班组
	ByClaimStatus    map[string]int64 `json:"by_claim_status"`
	BySupplier       []LabelCount     `json:"by_supplier"`
	OverdueClaims    []ClaimRow       `json:"overdue_claims"`
	GeneratedAt      time.Time        `json:"generated_at"`
}

// LabelCount 通用分组统计项。
type LabelCount struct {
	Label string `json:"label"`
	Count int64  `json:"count"`
}

// ExpiringDays "即将到期" 的窗口(天)。
const ExpiringDays = 30
