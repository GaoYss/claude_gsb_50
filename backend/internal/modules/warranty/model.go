package warranty

import "time"

// 质保部件类型。
const (
	ComponentLamp = "lamp" // 灯具(含灯头/驱动等随灯质保部件)
	ComponentPole = "pole" // 灯杆
)

// 责任方类型。
const (
	PartyManufacturer = "manufacturer" // 厂家(质保期内)
	PartyInternal     = "internal"     // 自有班组(超期或非质保部件)
)

// 责任工单状态。
const (
	ClaimStatusPending    = "pending"    // 待厂家响应
	ClaimStatusProcessing = "processing" // 厂家处理中(已开工响应)
	ClaimStatusOverdue    = "overdue"    // 厂家响应超时
	ClaimStatusInternal   = "internal"   // 自有班组处理中(超期/非质保部件)
	ClaimStatusTakenOver  = "taken_over" // 厂家超时后转由自有班组接手
	ClaimStatusClosed     = "closed"     // 已闭环
)

// DefaultResponseDeadlineHours 是供应商未单独配置时采用的默认响应时限(小时)。
const DefaultResponseDeadlineHours = 24

// Components 返回全部质保部件取值。
func Components() []string {
	return []string{ComponentLamp, ComponentPole}
}

// Parties 返回全部责任方取值。
func Parties() []string {
	return []string{PartyManufacturer, PartyInternal}
}

// ClaimStatuses 返回全部责任工单状态取值。
func ClaimStatuses() []string {
	return []string{
		ClaimStatusPending,
		ClaimStatusProcessing,
		ClaimStatusOverdue,
		ClaimStatusInternal,
		ClaimStatusTakenOver,
		ClaimStatusClosed,
	}
}

// IsValidComponent 校验部件取值。
func IsValidComponent(value string) bool {
	for _, item := range Components() {
		if item == value {
			return true
		}
	}
	return false
}

// Supplier 质保供应商(灯具 / 灯杆厂家)档案与联系方式。
type Supplier struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Name                  string    `gorm:"size:128;uniqueIndex;not null" json:"name"`
	ShortName             string    `gorm:"size:64" json:"short_name"`
	ContactPerson         string    `gorm:"size:64" json:"contact_person"`
	ContactPhone          string    `gorm:"size:32" json:"contact_phone"`
	ServicePhone          string    `gorm:"size:32" json:"service_phone"`
	Address               string    `gorm:"size:255" json:"address"`
	ResponseDeadlineHours int       `gorm:"not null;default:24" json:"response_deadline_hours"` // 承诺到场/响应时限
	Remark                string    `gorm:"size:255" json:"remark"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Supplier) TableName() string { return "warranty_supplier" }

// Warranty 单盏路灯的质保登记, 灯具与灯杆可分别登记供应商与质保起止日期。
type Warranty struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	LampID   uint   `gorm:"uniqueIndex;not null" json:"lamp_id"`
	LampCode string `gorm:"size:64;index" json:"lamp_code"`
	RoadName string `gorm:"size:128;index" json:"road_name"`

	// 灯具质保。
	LampSupplierID     *uint      `gorm:"index" json:"lamp_supplier_id"`
	LampSupplierName   string     `gorm:"size:128" json:"lamp_supplier_name"`
	LampStartAt        *time.Time `gorm:"type:date" json:"lamp_start_at"`
	LampEndAt          *time.Time `gorm:"type:date;index" json:"lamp_end_at"`
	LampWarrantyMonths int        `gorm:"not null;default:0" json:"lamp_warranty_months"`

	// 灯杆质保。
	PoleSupplierID     *uint      `gorm:"index" json:"pole_supplier_id"`
	PoleSupplierName   string     `gorm:"size:128" json:"pole_supplier_name"`
	PoleStartAt        *time.Time `gorm:"type:date" json:"pole_start_at"`
	PoleEndAt          *time.Time `gorm:"type:date;index" json:"pole_end_at"`
	PoleWarrantyMonths int        `gorm:"not null;default:0" json:"pole_warranty_months"`

	Remark    string    `gorm:"size:255" json:"remark"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Warranty) TableName() string { return "warranty_registration" }

// WarrantyClaim 故障责任工单: 每条故障登记时生成一条, 记录责任判定与厂家响应过程。
type WarrantyClaim struct {
	ID       uint   `gorm:"primaryKey" json:"id"`
	FaultID  uint   `gorm:"uniqueIndex;not null" json:"fault_id"`
	FaultNo  string `gorm:"size:64;index" json:"fault_no"`
	LampID   uint   `gorm:"index;not null" json:"lamp_id"`
	LampCode string `gorm:"size:64;index" json:"lamp_code"`
	RoadName string `gorm:"size:128;index" json:"road_name"`

	// 责任判定结果(登记时确定, 改派/转办后 CurrentStatus 表达当前负责方)。
	Component             string     `gorm:"size:16;index;not null" json:"component"` // lamp / pole, 非质保部件故障为空
	PartyType             string     `gorm:"size:16;index;not null;default:internal" json:"party_type"`
	InWarranty            bool       `gorm:"not null;default:false" json:"in_warranty"`
	SupplierID            *uint      `gorm:"index" json:"supplier_id"`
	SupplierName          string     `gorm:"size:128" json:"supplier_name"`
	ContactPerson         string     `gorm:"size:64" json:"contact_person"`
	ContactPhone          string     `gorm:"size:32" json:"contact_phone"`
	ServicePhone          string     `gorm:"size:32" json:"service_phone"`
	ResponseDeadlineHours int        `gorm:"not null;default:24" json:"response_deadline_hours"`
	WarrantyEndAt         *time.Time `gorm:"type:date" json:"warranty_end_at"`

	// 处置过程。
	Status         string     `gorm:"size:16;index;not null;default:pending" json:"status"`
	AssignedAt     time.Time  `gorm:"index;not null" json:"assigned_at"`
	NotifiedAt     *time.Time `json:"notified_at"`
	RespondedAt    *time.Time `json:"responded_at"` // 厂家首次开工响应时间
	LastRemindedAt *time.Time `json:"last_reminded_at"`
	RemindCount    int        `gorm:"not null;default:0" json:"remind_count"`
	TakeoverAt     *time.Time `json:"takeover_at"`
	TakeoverTeam   string     `gorm:"size:64" json:"takeover_team"`
	TakeoverBy     string     `gorm:"size:64" json:"takeover_by"`
	TakeoverReason string     `gorm:"size:255" json:"takeover_reason"`
	ClosedAt       *time.Time `json:"closed_at"`
	Result         string     `gorm:"size:32" json:"result"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (WarrantyClaim) TableName() string { return "warranty_claim" }

// LampActiveAt 判断灯具质保在指定时刻是否有效(截止日期当天仍计入质保)。
func (w *Warranty) LampActiveAt(at time.Time) bool {
	return dateRangeActive(w.LampStartAt, w.LampEndAt, at)
}

// PoleActiveAt 判断灯杆质保在指定时刻是否有效。
func (w *Warranty) PoleActiveAt(at time.Time) bool {
	return dateRangeActive(w.PoleStartAt, w.PoleEndAt, at)
}

// PartyDecision 是一次责任方判定的结果。
type PartyDecision struct {
	PartyType  string
	Component  string
	SupplierID *uint
	InWarranty bool
	EndAt      *time.Time
}

// Decide 依据部件类型与质保区间判定责任方: 质保期内指派厂家, 超期或非质保部件转自有班组。
// component 为空(线路 / 控制箱等非灯具灯杆故障)时一律由自有班组负责。
func (w *Warranty) Decide(component string, at time.Time) PartyDecision {
	decision := PartyDecision{PartyType: PartyInternal, Component: component}
	if w == nil {
		return decision
	}

	var (
		active     bool
		supplierID *uint
		endAt      *time.Time
	)
	switch component {
	case ComponentLamp:
		active = w.LampActiveAt(at)
		supplierID = w.LampSupplierID
		endAt = w.LampEndAt
	case ComponentPole:
		active = w.PoleActiveAt(at)
		supplierID = w.PoleSupplierID
		endAt = w.PoleEndAt
	default:
		return decision
	}

	decision.InWarranty = active
	decision.EndAt = endAt
	if active && supplierID != nil {
		decision.PartyType = PartyManufacturer
		decision.SupplierID = supplierID
	}
	return decision
}

// dateRangeActive 判断 [start, end] 日期区间是否覆盖 at, 起止为空时按单边判断, 截止日当天有效。
func dateRangeActive(start, end *time.Time, at time.Time) bool {
	if end == nil {
		return false
	}
	day := truncateDay(at)
	if day.After(truncateDay(*end)) {
		return false
	}
	if start != nil && day.Before(truncateDay(*start)) {
		return false
	}
	return true
}

// truncateDay 归整到当地零点, 仅按日期比较。
func truncateDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
