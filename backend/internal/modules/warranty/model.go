package warranty

import "time"

// 质保部件: 一盏路灯分别登记灯具与灯杆两类质保。
const (
	ComponentLuminaire = "luminaire" // 灯具
	ComponentPole      = "pole"      // 灯杆
)

// 责任方类型。
const (
	ResponsibleSupplier = "supplier" // 厂家(质保期内)
	ResponsibleOwnTeam  = "own_team" // 自有班组(无质保或质保超期)
)

// 责任判定单状态。
const (
	AssignPending     = "pending"     // 处理中
	AssignOverdue     = "overdue"     // 厂家处理超时, 已自动提醒
	AssignTransferred = "transferred" // 已转自有班组接手
)

// 质保状态(按当前日期推算, 不落库)。
const (
	StateActive   = "active"   // 在保
	StateExpiring = "expiring" // 临期(30 天内到期)
	StateExpired  = "expired"  // 已到期
)

// ResponseTimeout 是厂家处理响应的超时阈值, 超过后自动提醒并允许转自有班组。
const ResponseTimeout = 24 * time.Hour

// ExpiringWindow 是质保临期的判定窗口。
const ExpiringWindow = 30 * 24 * time.Hour

// Components 返回全部质保部件取值。
func Components() []string {
	return []string{ComponentLuminaire, ComponentPole}
}

// ResponsibleTypes 返回全部责任方类型取值。
func ResponsibleTypes() []string {
	return []string{ResponsibleSupplier, ResponsibleOwnTeam}
}

// AssignmentStatuses 返回全部判定单状态取值。
func AssignmentStatuses() []string {
	return []string{AssignPending, AssignOverdue, AssignTransferred}
}

// IsValidComponent 校验质保部件取值。
func IsValidComponent(component string) bool {
	for _, item := range Components() {
		if item == component {
			return true
		}
	}
	return false
}

// IsValidResponsibleType 校验责任方类型取值。
func IsValidResponsibleType(value string) bool {
	for _, item := range ResponsibleTypes() {
		if item == value {
			return true
		}
	}
	return false
}

// IsValidAssignmentStatus 校验判定单状态取值。
func IsValidAssignmentStatus(status string) bool {
	for _, item := range AssignmentStatuses() {
		if item == status {
			return true
		}
	}
	return false
}

// ComponentForFaultType 依据故障类型判定责任部件。
// 灯杆类故障(灯杆倾斜)归灯杆质保, 其余电气与灯具故障默认归灯具质保。
func ComponentForFaultType(faultType string) string {
	switch faultType {
	case "灯杆倾斜":
		return ComponentPole
	default:
		return ComponentLuminaire
	}
}

// Supplier 供应商档案, 记录厂家联系方式, 供质保登记引用。
type Supplier struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	Name          string    `gorm:"size:128;uniqueIndex;not null" json:"name"`
	ContactPerson string    `gorm:"size:64" json:"contact_person"`
	ContactPhone  string    `gorm:"size:32" json:"contact_phone"`
	Email         string    `gorm:"size:128" json:"email"`
	Address       string    `gorm:"size:255" json:"address"`
	Remark        string    `gorm:"size:255" json:"remark"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Supplier) TableName() string { return "supplier" }

// Warranty 质保登记, 一盏路灯的一个部件对应一条质保记录。
type Warranty struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	LampID        uint       `gorm:"uniqueIndex:idx_warranty_lamp_component;not null" json:"lamp_id"`
	LampCode      string     `gorm:"size:64;index" json:"lamp_code"`
	Component     string     `gorm:"size:32;uniqueIndex:idx_warranty_lamp_component;not null" json:"component"`
	SupplierID    *uint      `gorm:"index" json:"supplier_id"`
	SupplierName  string     `gorm:"size:128" json:"supplier_name"`
	ContactPerson string     `gorm:"size:64" json:"contact_person"`
	ContactPhone  string     `gorm:"size:32" json:"contact_phone"`
	StartDate     *time.Time `gorm:"type:date;not null" json:"start_date"`
	EndDate       *time.Time `gorm:"type:date;not null" json:"end_date"`
	Remark        string     `gorm:"size:255" json:"remark"`

	// State 质保状态(在保/临期/已到期), 仅用于响应展示, 不落库。
	State string `gorm:"-" json:"state,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName 指定表名。
func (Warranty) TableName() string { return "warranty" }

// FillState 依据当前日期推算质保状态。
func (w *Warranty) FillState(now time.Time) {
	w.State = StateOf(w.EndDate, now)
}

// StateOf 计算指定到期日期对应的质保状态。
func StateOf(endDate *time.Time, now time.Time) string {
	if endDate == nil {
		return StateExpired
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if endDate.Before(today) {
		return StateExpired
	}
	if endDate.Before(today.Add(ExpiringWindow)) {
		return StateExpiring
	}
	return StateActive
}

// Covers 判断质保是否覆盖指定时刻(按日期区间含端点)。
func (w *Warranty) Covers(moment time.Time) bool {
	if w.StartDate == nil || w.EndDate == nil {
		return false
	}
	dayEnd := w.EndDate.AddDate(0, 0, 1)
	return !moment.Before(*w.StartDate) && moment.Before(dayEnd)
}

// FaultAssignment 故障责任判定单, 登记故障时自动生成, 跟踪厂家处理时效。
type FaultAssignment struct {
	ID              uint       `gorm:"primaryKey" json:"id"`
	FaultID         uint       `gorm:"uniqueIndex;not null" json:"fault_id"`
	FaultNo         string     `gorm:"size:64;index" json:"fault_no"`
	LampID          uint       `gorm:"index;not null" json:"lamp_id"`
	LampCode        string     `gorm:"size:64;index" json:"lamp_code"`
	Component       string     `gorm:"size:32;index" json:"component"`
	InWarranty      bool       `gorm:"index" json:"in_warranty"`
	ResponsibleType string     `gorm:"size:32;index;not null" json:"responsible_type"`
	Status          string     `gorm:"size:32;index;not null;default:pending" json:"status"`
	WarrantyID      *uint      `json:"warranty_id"`
	SupplierID      *uint      `json:"supplier_id"`
	SupplierName    string     `gorm:"size:128" json:"supplier_name"`
	ContactPerson   string     `gorm:"size:64" json:"contact_person"`
	ContactPhone    string     `gorm:"size:32" json:"contact_phone"`
	Reason          string     `gorm:"size:255" json:"reason"`
	AssignedAt      time.Time  `gorm:"index;not null" json:"assigned_at"`
	Deadline        time.Time  `gorm:"index;not null" json:"deadline"`
	RemindedAt      *time.Time `json:"reminded_at"`
	TransferredAt   *time.Time `json:"transferred_at"`
	TransferRemark  string     `gorm:"size:255" json:"transfer_remark"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (FaultAssignment) TableName() string { return "fault_assignment" }
