package warranty

import (
	"context"
	"log/slog"
	"math"
	"strings"
	"time"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/pkg/pagination"
)

// sweepInterval 是厂家响应超时的后台巡检间隔。
const sweepInterval = 10 * time.Minute

// remindCooldown 是同一工单两次自动提醒之间的最小间隔。
const remindCooldown = 24 * time.Hour

// warrantySortSpec 质保登记列表排序白名单。
var warrantySortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"lamp_code":   "lamp_code",
		"road_name":   "road_name",
		"lamp_end_at": "lamp_end_at",
		"pole_end_at": "pole_end_at",
		"updated_at":  "updated_at",
		"created_at":  "created_at",
	},
	Default: "lamp_code",
}

// supplierSortSpec 供应商列表排序白名单。
var supplierSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"name":       "name",
		"short_name": "short_name",
		"created_at": "created_at",
		"updated_at": "updated_at",
	},
	Default: "id",
}

// claimSortSpec 责任工单列表排序白名单。
var claimSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"fault_no":    "fault_no",
		"lamp_code":   "lamp_code",
		"road_name":   "road_name",
		"component":   "component",
		"party_type":  "party_type",
		"status":      "status",
		"assigned_at": "assigned_at",
		"created_at":  "created_at",
	},
	Default: "assigned_at",
}

// Service 承载质保与责任方管理的业务规则。
type Service struct {
	repo   *Repository
	lamps  *lamp.Repository
	faults *fault.Repository

	stopSweep context.CancelFunc
}

// NewService 构造质保管理服务。
func NewService(repo *Repository, lamps *lamp.Repository, faults *fault.Repository) *Service {
	return &Service{repo: repo, lamps: lamps, faults: faults}
}

// Repository 暴露仓储, 供其它模块装配只读端口。
func (s *Service) Repository() *Repository { return s.repo }

// ComponentForFaultType 依据故障类型推断质保部件:
// 灯头/光源类故障归属灯具, 灯杆倾斜归属灯杆, 线路与控制箱等不属质保部件。
func ComponentForFaultType(faultType string) string {
	switch strings.TrimSpace(faultType) {
	case "灯杆倾斜":
		return ComponentPole
	case "灯不亮", "灯光闪烁", "灯具常亮", "灯具破损":
		return ComponentLamp
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// 供应商
// ---------------------------------------------------------------------------

// CreateSupplier 新增供应商。
func (s *Service) CreateSupplier(ctx context.Context, req SupplierUpsertRequest) (*Supplier, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperr.BadRequest("供应商名称不能为空")
	}
	exists, err := s.repo.ExistsSupplierName(ctx, name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperr.Conflict("供应商名称已存在: %s", name)
	}

	entity := &Supplier{
		Name:                  name,
		ShortName:             strings.TrimSpace(req.ShortName),
		ContactPerson:         strings.TrimSpace(req.ContactPerson),
		ContactPhone:          strings.TrimSpace(req.ContactPhone),
		ServicePhone:          strings.TrimSpace(req.ServicePhone),
		Address:               strings.TrimSpace(req.Address),
		ResponseDeadlineHours: deadlineOrDefault(req.ResponseDeadlineHours),
		Remark:                strings.TrimSpace(req.Remark),
	}
	if err := s.repo.CreateSupplier(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// UpdateSupplier 修改供应商。
func (s *Service) UpdateSupplier(ctx context.Context, id uint, req SupplierUpsertRequest) (*Supplier, error) {
	entity, err := s.repo.GetSupplier(ctx, id)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperr.BadRequest("供应商名称不能为空")
	}
	if name != entity.Name {
		exists, err := s.repo.ExistsSupplierName(ctx, name, id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, apperr.Conflict("供应商名称已存在: %s", name)
		}
	}

	entity.Name = name
	entity.ShortName = strings.TrimSpace(req.ShortName)
	entity.ContactPerson = strings.TrimSpace(req.ContactPerson)
	entity.ContactPhone = strings.TrimSpace(req.ContactPhone)
	entity.ServicePhone = strings.TrimSpace(req.ServicePhone)
	entity.Address = strings.TrimSpace(req.Address)
	entity.ResponseDeadlineHours = deadlineOrDefault(req.ResponseDeadlineHours)
	entity.Remark = strings.TrimSpace(req.Remark)
	if err := s.repo.UpdateSupplier(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// DeleteSupplier 删除供应商, 仍被质保登记或未闭环工单引用时拒绝。
func (s *Service) DeleteSupplier(ctx context.Context, id uint) error {
	if _, err := s.repo.GetSupplier(ctx, id); err != nil {
		return err
	}
	warrantyRefs, err := s.repo.CountWarrantyRefsBySupplier(ctx, id)
	if err != nil {
		return err
	}
	if warrantyRefs > 0 {
		return apperr.Conflict("该供应商仍被 %d 条质保登记引用, 请先调整质保归属", warrantyRefs)
	}
	claimRefs, err := s.repo.CountOpenClaimRefsBySupplier(ctx, id)
	if err != nil {
		return err
	}
	if claimRefs > 0 {
		return apperr.Conflict("该供应商仍有 %d 条未闭环责任工单, 暂不允许删除", claimRefs)
	}
	return s.repo.DeleteSupplier(ctx, id)
}

// GetSupplier 查询供应商详情。
func (s *Service) GetSupplier(ctx context.Context, id uint) (*Supplier, error) {
	return s.repo.GetSupplier(ctx, id)
}

// ListSuppliers 分页查询供应商。
func (s *Service) ListSuppliers(ctx context.Context, query SupplierListQuery) ([]Supplier, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, supplierSortSpec)
	items, total, err := s.repo.ListSuppliers(ctx, query.Keyword, page)
	if err != nil {
		return nil, 0, page, err
	}
	return items, total, page, nil
}

// SupplierOptions 返回供应商下拉选项。
func (s *Service) SupplierOptions(ctx context.Context) ([]SupplierOption, error) {
	entities, err := s.repo.ListAllSuppliers(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]SupplierOption, 0, len(entities))
	for _, item := range entities {
		options = append(options, toSupplierOption(&item))
	}
	return options, nil
}

// Metadata 返回质保模块字典。
func (s *Service) Metadata() *Meta {
	return &Meta{
		Components:        Components(),
		Parties:           Parties(),
		ClaimStatuses:     ClaimStatuses(),
		DefaultDeadlineHr: DefaultResponseDeadlineHours,
	}
}

func deadlineOrDefault(value *int) int {
	if value == nil || *value <= 0 {
		return DefaultResponseDeadlineHours
	}
	return *value
}

func toSupplierOption(entity *Supplier) SupplierOption {
	return SupplierOption{
		ID:                    entity.ID,
		Name:                  entity.Name,
		ShortName:             entity.ShortName,
		ContactPerson:         entity.ContactPerson,
		ContactPhone:          entity.ContactPhone,
		ServicePhone:          entity.ServicePhone,
		ResponseDeadlineHours: entity.ResponseDeadlineHours,
	}
}

// ---------------------------------------------------------------------------
// 质保登记
// ---------------------------------------------------------------------------

// warrantySide 是单侧(灯具/灯杆)质保登记的解析结果。
type warrantySide struct {
	supplierID *uint
	supplier   *Supplier
	startAt    *time.Time
	endAt      *time.Time
	months     int
}

// UpsertWarranty 为路灯登记灯具与灯杆质保, 已存在时整体更新。
func (s *Service) UpsertWarranty(ctx context.Context, req WarrantyUpsertRequest) (*Warranty, error) {
	device, err := s.lamps.GetByID(ctx, req.LampID)
	if err != nil {
		return nil, err
	}

	lampSide, err := s.parseSide(ctx, req.LampSupplierID, req.LampStartAt, req.LampEndAt, req.LampWarrantyMonths, "灯具")
	if err != nil {
		return nil, err
	}
	poleSide, err := s.parseSide(ctx, req.PoleSupplierID, req.PoleStartAt, req.PoleEndAt, req.PoleWarrantyMonths, "灯杆")
	if err != nil {
		return nil, err
	}
	if lampSide == nil && poleSide == nil {
		return nil, apperr.BadRequest("请至少登记灯具或灯杆其中一项的供应商与质保起止日期")
	}

	entity, err := s.repo.GetWarrantyByLamp(ctx, device.ID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		entity = &Warranty{LampID: device.ID}
	}

	entity.LampCode = device.Code
	entity.RoadName = device.RoadName
	applySide := func(side *warrantySide, target *Warranty, isLamp bool) {
		clear := func() {
			if isLamp {
				entity.LampSupplierID, entity.LampSupplierName = nil, ""
				entity.LampStartAt, entity.LampEndAt = nil, nil
				entity.LampWarrantyMonths = 0
			} else {
				entity.PoleSupplierID, entity.PoleSupplierName = nil, ""
				entity.PoleStartAt, entity.PoleEndAt = nil, nil
				entity.PoleWarrantyMonths = 0
			}
		}
		if side == nil {
			clear()
			return
		}
		if isLamp {
			target.LampSupplierID = side.supplierID
			target.LampSupplierName = side.supplier.Name
			target.LampStartAt = side.startAt
			target.LampEndAt = side.endAt
			target.LampWarrantyMonths = side.months
		} else {
			target.PoleSupplierID = side.supplierID
			target.PoleSupplierName = side.supplier.Name
			target.PoleStartAt = side.startAt
			target.PoleEndAt = side.endAt
			target.PoleWarrantyMonths = side.months
		}
	}
	applySide(lampSide, entity, true)
	applySide(poleSide, entity, false)
	entity.Remark = strings.TrimSpace(req.Remark)

	if entity.ID == 0 {
		err = s.repo.CreateWarranty(ctx, entity)
	} else {
		err = s.repo.UpdateWarranty(ctx, entity)
	}
	if err != nil {
		return nil, err
	}
	return entity, nil
}

// parseSide 解析单侧质保入参: 未提供任何字段视为不登记; 提供则供应商与起止日期必填且结束不早于开始。
func (s *Service) parseSide(ctx context.Context, supplierID *uint, startRaw, endRaw string, months *int, label string) (*warrantySide, error) {
	supplierID = normalizeID(supplierID)
	startRaw, endRaw = strings.TrimSpace(startRaw), strings.TrimSpace(endRaw)
	if supplierID == nil && startRaw == "" && endRaw == "" {
		return nil, nil
	}
	if supplierID == nil {
		return nil, apperr.BadRequest("%s质保请选择供应商", label)
	}
	supplier, err := s.repo.GetSupplier(ctx, *supplierID)
	if err != nil {
		return nil, err
	}
	if startRaw == "" || endRaw == "" {
		return nil, apperr.BadRequest("%s质保的起止日期均需填写", label)
	}
	startAt, err := parseDate(startRaw)
	if err != nil {
		return nil, apperr.BadRequest("%s质保开始日期格式应为 YYYY-MM-DD", label)
	}
	endAt, err := parseDate(endRaw)
	if err != nil {
		return nil, apperr.BadRequest("%s质保结束日期格式应为 YYYY-MM-DD", label)
	}
	if endAt.Before(startAt) {
		return nil, apperr.BadRequest("%s质保结束日期不能早于开始日期", label)
	}

	monthCount := 0
	if months != nil && *months > 0 {
		monthCount = *months
	} else {
		monthCount = monthsBetween(startAt, endAt)
	}
	return &warrantySide{
		supplierID: supplierID,
		supplier:   supplier,
		startAt:    &startAt,
		endAt:      &endAt,
		months:     monthCount,
	}, nil
}

// DeleteWarranty 删除质保登记, 该路灯存在质保内未闭环工单时拒绝。
func (s *Service) DeleteWarranty(ctx context.Context, id uint) error {
	entity, err := s.repo.GetWarranty(ctx, id)
	if err != nil {
		return err
	}
	open, err := s.faults.GetOpenByLamp(ctx, entity.LampID)
	if err != nil {
		return err
	}
	if open != nil {
		claim, err := s.repo.GetClaimByFault(ctx, open.ID)
		if err != nil {
			return err
		}
		if claim != nil && claim.PartyType == PartyManufacturer {
			return apperr.Conflict("路灯 %s 的故障 %s 仍由厂家处理中, 请先闭环或转办后再删除质保登记", entity.LampCode, open.FaultNo)
		}
	}
	return s.repo.DeleteWarranty(ctx, id)
}

// GetWarranty 查询质保登记详情。
func (s *Service) GetWarranty(ctx context.Context, id uint) (*WarrantyRow, error) {
	entity, err := s.repo.GetWarranty(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := s.enrichWarranties(ctx, []Warranty{*entity})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// GetWarrantyByLamp 查询某盏路灯的质保登记(供故障详情展示), 未登记时返回 nil。
func (s *Service) GetWarrantyByLamp(ctx context.Context, lampID uint) (*Warranty, error) {
	return s.repo.GetWarrantyByLamp(ctx, lampID)
}

// ListWarranties 分页查询质保登记并附加派生状态。
func (s *Service) ListWarranties(ctx context.Context, query WarrantyListQuery) ([]WarrantyRow, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, warrantySortSpec)
	items, total, err := s.repo.ListWarranties(ctx, warrantyFilter{
		Keyword:       strings.TrimSpace(query.Keyword),
		SupplierID:    query.SupplierID,
		Component:     strings.TrimSpace(query.Component),
		WarrantyState: strings.TrimSpace(query.WarrantyState),
		RoadName:      strings.TrimSpace(query.RoadName),
	}, page)
	if err != nil {
		return nil, 0, page, err
	}
	rows, err := s.enrichWarranties(ctx, items)
	if err != nil {
		return nil, 0, page, err
	}
	return rows, total, page, nil
}

// enrichWarranties 批量补充路灯名称、供应商选项与派生的质保状态。
func (s *Service) enrichWarranties(ctx context.Context, items []Warranty) ([]WarrantyRow, error) {
	rows := make([]WarrantyRow, 0, len(items))
	if len(items) == 0 {
		return rows, nil
	}

	ids := make([]uint, 0, len(items))
	supplierIDs := make([]uint, 0, len(items)*2)
	for _, item := range items {
		ids = append(ids, item.LampID)
		if item.LampSupplierID != nil {
			supplierIDs = append(supplierIDs, *item.LampSupplierID)
		}
		if item.PoleSupplierID != nil {
			supplierIDs = append(supplierIDs, *item.PoleSupplierID)
		}
	}

	lamps, err := s.lamps.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	lampNames := make(map[uint]string, len(lamps))
	for _, item := range lamps {
		lampNames[item.ID] = item.Name
	}
	suppliers, err := s.repo.GetSuppliersByIDs(ctx, supplierIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	for _, item := range items {
		row := WarrantyRow{Warranty: item, LampName: lampNames[item.LampID]}
		if item.LampSupplierID != nil {
			if supplier, ok := suppliers[*item.LampSupplierID]; ok {
				option := toSupplierOption(supplier)
				row.LampSupplier = &option
			}
		}
		if item.PoleSupplierID != nil {
			if supplier, ok := suppliers[*item.PoleSupplierID]; ok {
				option := toSupplierOption(supplier)
				row.PoleSupplier = &option
			}
		}
		row.State.Lamp = componentState(&item, true, now)
		row.State.Pole = componentState(&item, false, now)
		row.LampRemainingDays = remainingDays(item.LampEndAt, now)
		row.PoleRemainingDays = remainingDays(item.PoleEndAt, now)
		rows = append(rows, row)
	}
	return rows, nil
}

// componentState 计算单侧部件的展示状态。
func componentState(entity *Warranty, isLamp bool, now time.Time) string {
	var (
		supplierID *uint
		endAt      *time.Time
	)
	if isLamp {
		supplierID, endAt = entity.LampSupplierID, entity.LampEndAt
	} else {
		supplierID, endAt = entity.PoleSupplierID, entity.PoleEndAt
	}
	if supplierID == nil {
		return "unregistered"
	}
	if endAt == nil || !truncateDay(*endAt).Before(truncateDay(now)) {
		return "active"
	}
	return "expired"
}

// remainingDays 返回距质保截止日的天数, 已超期为负数, 未设置返回 0。
func remainingDays(endAt *time.Time, now time.Time) int {
	if endAt == nil {
		return 0
	}
	end := truncateDay(*endAt)
	today := truncateDay(now)
	return int(end.Sub(today).Hours() / 24)
}

// monthsBetween 计算两个日期之间的整月数(按对日计算, 不足一月按一月计)。
func monthsBetween(start, end time.Time) int {
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month())
	if end.Day() > start.Day() {
		months++
	}
	if months < 1 {
		months = 1
	}
	return months
}

// ---------------------------------------------------------------------------
// 责任工单: 故障联动
// ---------------------------------------------------------------------------

// OnFaultCreated 故障登记后自动判定责任方并生成责任工单, 幂等。
// 质保期内的灯具/灯杆故障指派厂家, 超期或非质保部件转自有班组。
func (s *Service) OnFaultCreated(ctx context.Context, entity *fault.Fault) error {
	exists, err := s.repo.GetClaimByFault(ctx, entity.ID)
	if err != nil {
		return err
	}
	if exists != nil {
		return nil
	}

	component := ComponentForFaultType(entity.FaultType)
	registration, err := s.repo.GetWarrantyByLamp(ctx, entity.LampID)
	if err != nil {
		return err
	}
	decision := registration.Decide(component, entity.ReportedAt)

	claim := &WarrantyClaim{
		FaultID:    entity.ID,
		FaultNo:    entity.FaultNo,
		LampID:     entity.LampID,
		LampCode:   entity.LampCode,
		RoadName:   entity.RoadName,
		Component:  component,
		PartyType:  decision.PartyType,
		InWarranty: decision.InWarranty,
		AssignedAt: entity.ReportedAt,
	}
	if decision.EndAt != nil {
		end := *decision.EndAt
		claim.WarrantyEndAt = &end
	}

	if decision.PartyType == PartyManufacturer && decision.SupplierID != nil {
		supplier, err := s.repo.GetSupplier(ctx, *decision.SupplierID)
		if err != nil {
			return err
		}
		claim.SupplierID = &supplier.ID
		claim.SupplierName = supplier.Name
		claim.ContactPerson = supplier.ContactPerson
		claim.ContactPhone = supplier.ContactPhone
		claim.ServicePhone = supplier.ServicePhone
		claim.ResponseDeadlineHours = supplier.ResponseDeadlineHours
		claim.Status = ClaimStatusPending
		notifiedAt := entity.ReportedAt
		claim.NotifiedAt = &notifiedAt
	} else {
		// 非质保部件或已超期: 直接由自有班组承接, 跟踪至故障闭环。
		claim.Status = ClaimStatusInternal
	}

	if err := s.repo.CreateClaim(ctx, claim); err != nil {
		return err
	}
	slog.Info("故障责任方已自动判定",
		"fault_no", entity.FaultNo, "lamp_code", entity.LampCode,
		"component", ComponentLabel(component), "party", PartyLabel(claim.PartyType),
		"supplier", claim.SupplierName, "in_warranty", claim.InWarranty,
	)
	return nil
}

// OnFaultClosed 故障闭环时同步关闭仍处于处理中的责任工单。
func (s *Service) OnFaultClosed(ctx context.Context, faultID uint) error {
	claim, err := s.repo.GetClaimByFault(ctx, faultID)
	if err != nil || claim == nil {
		return err
	}
	if claim.Status == ClaimStatusClosed {
		return nil
	}
	now := time.Now()
	return s.repo.UpdateClaimColumns(ctx, claim.ID, map[string]any{
		"status":    ClaimStatusClosed,
		"closed_at": now,
	})
}

// OnFaultDeleted 删除故障时清理其责任工单。
func (s *Service) OnFaultDeleted(ctx context.Context, faultID uint) error {
	return s.repo.DeleteClaimByFault(ctx, faultID)
}

// OnRepairStarted 维修开工联动: 厂家责任工单在厂家首次开工时标记为已响应、处理中。
func (s *Service) OnRepairStarted(ctx context.Context, faultID uint) error {
	claim, err := s.repo.GetClaimByFault(ctx, faultID)
	if err != nil || claim == nil {
		return err
	}
	if claim.PartyType != PartyManufacturer || claim.RespondedAt != nil {
		return nil
	}
	if claim.Status == ClaimStatusClosed || claim.Status == ClaimStatusTakenOver {
		return nil
	}
	now := time.Now()
	return s.repo.UpdateClaimColumns(ctx, claim.ID, map[string]any{
		"status":       ClaimStatusProcessing,
		"responded_at": now,
	})
}

// Remind 人工催办厂家, 记录催办时间与次数。
func (s *Service) Remind(ctx context.Context, id uint, req RemindRequest) (*WarrantyClaim, error) {
	claim, err := s.getOpenManufacturerClaim(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	columns := map[string]any{
		"last_reminded_at": now,
		"remind_count":     claim.RemindCount + 1,
	}
	if claim.NotifiedAt == nil {
		columns["notified_at"] = now
	}
	if isResponseOverdue(claim, now) {
		columns["status"] = ClaimStatusOverdue
	}
	if err := s.repo.UpdateClaimColumns(ctx, claim.ID, columns); err != nil {
		return nil, err
	}
	slog.Warn("已向质保厂家发送催办提醒",
		"fault_no", claim.FaultNo, "supplier", claim.SupplierName,
		"contact", claim.ContactPhone, "service_phone", claim.ServicePhone,
		"deadline_hours", claim.ResponseDeadlineHours, "remark", strings.TrimSpace(req.Remark))
	return s.repo.GetClaim(ctx, claim.ID)
}

// MarkResponded 手工登记厂家已响应(到场), 工单进入厂家处理中。
func (s *Service) MarkResponded(ctx context.Context, id uint) (*WarrantyClaim, error) {
	claim, err := s.getOpenManufacturerClaim(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	columns := map[string]any{"status": ClaimStatusProcessing}
	if claim.RespondedAt == nil {
		columns["responded_at"] = now
	}
	if err := s.repo.UpdateClaimColumns(ctx, claim.ID, columns); err != nil {
		return nil, err
	}
	return s.repo.GetClaim(ctx, claim.ID)
}

// Takeover 将厂家工单转由自有班组接手。
func (s *Service) Takeover(ctx context.Context, id uint, req TakeoverRequest) (*WarrantyClaim, error) {
	claim, err := s.getOpenManufacturerClaim(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.repo.UpdateClaimColumns(ctx, claim.ID, map[string]any{
		"status":          ClaimStatusTakenOver,
		"takeover_at":     now,
		"takeover_team":   strings.TrimSpace(req.Team),
		"takeover_by":     strings.TrimSpace(req.By),
		"takeover_reason": strings.TrimSpace(req.Reason),
	}); err != nil {
		return nil, err
	}
	slog.Warn("厂家责任工单已转由自有班组接手",
		"fault_no", claim.FaultNo, "supplier", claim.SupplierName,
		"team", strings.TrimSpace(req.Team), "by", strings.TrimSpace(req.By))
	return s.repo.GetClaim(ctx, claim.ID)
}

// getOpenManufacturerClaim 取出仍处开放状态的厂家责任工单。
func (s *Service) getOpenManufacturerClaim(ctx context.Context, id uint) (*WarrantyClaim, error) {
	claim, err := s.repo.GetClaim(ctx, id)
	if err != nil {
		return nil, err
	}
	if claim.PartyType != PartyManufacturer {
		return nil, apperr.Conflict("故障 %s 责任方为自有班组, 无需厂家催办/转办", claim.FaultNo)
	}
	if claim.Status == ClaimStatusClosed {
		return nil, apperr.Conflict("故障 %s 的责任工单已闭环", claim.FaultNo)
	}
	if claim.Status == ClaimStatusTakenOver {
		return nil, apperr.Conflict("故障 %s 已转由自有班组接手", claim.FaultNo)
	}
	return claim, nil
}

// GetClaim 查询责任工单详情。
func (s *Service) GetClaim(ctx context.Context, id uint) (*ClaimRow, error) {
	claim, err := s.repo.GetClaim(ctx, id)
	if err != nil {
		return nil, err
	}
	rows, err := s.enrichClaims(ctx, []WarrantyClaim{*claim})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// GetClaimByFault 查询某条故障的责任工单, 未生成时返回 nil。
func (s *Service) GetClaimByFault(ctx context.Context, faultID uint) (*ClaimRow, error) {
	claim, err := s.repo.GetClaimByFault(ctx, faultID)
	if err != nil || claim == nil {
		return nil, err
	}
	rows, err := s.enrichClaims(ctx, []WarrantyClaim{*claim})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// ClaimsByFaults 按故障 ID 批量返回责任工单, 供故障列表/详情展示责任方。
func (s *Service) ClaimsByFaults(ctx context.Context, faultIDs []uint) (map[uint]ClaimRow, error) {
	result := make(map[uint]ClaimRow)
	if len(faultIDs) == 0 {
		return result, nil
	}
	claims, err := s.repo.GetClaimsByFaults(ctx, faultIDs)
	if err != nil {
		return nil, err
	}
	rows, err := s.enrichClaims(ctx, claims)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.FaultID] = row
	}
	return result, nil
}

// ListClaims 分页查询责任工单。
func (s *Service) ListClaims(ctx context.Context, query ClaimListQuery) ([]ClaimRow, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, claimSortSpec)

	status := strings.TrimSpace(query.Status)
	if status != "" && !isValidClaimStatus(status) {
		return nil, 0, page, apperr.BadRequest("非法的责任工单状态: %s", status)
	}
	partyType := strings.TrimSpace(query.PartyType)
	if partyType != "" && partyType != PartyManufacturer && partyType != PartyInternal {
		return nil, 0, page, apperr.BadRequest("非法的责任方类型: %s", partyType)
	}
	component := strings.TrimSpace(query.Component)
	if component != "" && !IsValidComponent(component) {
		return nil, 0, page, apperr.BadRequest("非法的质保部件: %s", component)
	}

	items, total, err := s.repo.ListClaims(ctx, claimFilter{
		Keyword:    strings.TrimSpace(query.Keyword),
		Status:     status,
		PartyType:  partyType,
		Component:  component,
		SupplierID: query.SupplierID,
		InWarranty: query.InWarranty,
		OnlyOpen:   query.OnlyOpen,
	}, page)
	if err != nil {
		return nil, 0, page, err
	}
	rows, err := s.enrichClaims(ctx, items)
	if err != nil {
		return nil, 0, page, err
	}
	return rows, total, page, nil
}

// enrichClaims 批量补充供应商选项、故障摘要与超时派生字段。
func (s *Service) enrichClaims(ctx context.Context, items []WarrantyClaim) ([]ClaimRow, error) {
	rows := make([]ClaimRow, 0, len(items))
	if len(items) == 0 {
		return rows, nil
	}

	supplierIDs := make([]uint, 0, len(items))
	faultIDs := make([]uint, 0, len(items))
	for _, item := range items {
		if item.SupplierID != nil {
			supplierIDs = append(supplierIDs, *item.SupplierID)
		}
		faultIDs = append(faultIDs, item.FaultID)
	}
	suppliers, err := s.repo.GetSuppliersByIDs(ctx, supplierIDs)
	if err != nil {
		return nil, err
	}
	faults, err := s.faults.ListByIDs(ctx, faultIDs)
	if err != nil {
		return nil, err
	}
	faultMap := make(map[uint]fault.Fault, len(faults))
	for _, item := range faults {
		faultMap[item.ID] = item
	}

	now := time.Now()
	for _, item := range items {
		row := ClaimRow{WarrantyClaim: item}
		if item.SupplierID != nil {
			if supplier, ok := suppliers[*item.SupplierID]; ok {
				option := toSupplierOption(supplier)
				row.Supplier = &option
			}
		}
		if target, ok := faultMap[item.FaultID]; ok {
			row.FaultType = target.FaultType
			row.FaultLevel = target.FaultLevel
			row.FaultStatus = target.Status
		}
		row.Overdue = isResponseOverdue(&item, now)
		row.WaitingHours = round2(now.Sub(item.AssignedAt).Hours())
		if item.RespondedAt != nil {
			row.ResponseUsedHours = round2(item.RespondedAt.Sub(item.AssignedAt).Hours())
		} else {
			row.ResponseUsedHours = row.WaitingHours
		}
		if item.Status == ClaimStatusTakenOver {
			row.CurrentParty = PartyInternal
		} else {
			row.CurrentParty = item.PartyType
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// SweepOverdue 巡检厂家响应情况: 超过承诺时限仍未响应的工单自动置为超时并发送提醒。
func (s *Service) SweepOverdue(ctx context.Context) (overdue int, reminded int, err error) {
	pending, err := s.repo.ListPendingResponse(ctx)
	if err != nil {
		return 0, 0, err
	}
	now := time.Now()
	for _, item := range pending {
		if !isResponseOverdue(&item, now) {
			continue
		}
		overdue++

		columns := map[string]any{}
		if item.Status != ClaimStatusOverdue {
			columns["status"] = ClaimStatusOverdue
		}
		shouldRemind := item.LastRemindedAt == nil ||
			now.Sub(*item.LastRemindedAt) >= remindCooldown
		if shouldRemind {
			columns["last_reminded_at"] = now
			columns["remind_count"] = item.RemindCount + 1
			if item.NotifiedAt == nil {
				columns["notified_at"] = item.AssignedAt
			}
			reminded++
		}
		if len(columns) > 0 {
			if updateErr := s.repo.UpdateClaimColumns(ctx, item.ID, columns); updateErr != nil {
				return overdue, reminded, updateErr
			}
		}
		if shouldRemind {
			slog.Warn("厂家响应超时, 已自动提醒",
				"fault_no", item.FaultNo, "lamp_code", item.LampCode,
				"supplier", item.SupplierName, "contact_person", item.ContactPerson,
				"contact_phone", item.ContactPhone, "service_phone", item.ServicePhone,
				"deadline_hours", item.ResponseDeadlineHours,
				"waiting_hours", round2(now.Sub(item.AssignedAt).Hours()),
			)
		}
	}
	return overdue, reminded, nil
}

// StartScheduler 启动厂家响应超时的定时巡检, 进程退出时随 ctx 取消。
func (s *Service) StartScheduler(ctx context.Context) {
	if s.stopSweep != nil {
		return
	}
	ctx, cancel := context.WithCancel(ctx)
	s.stopSweep = cancel

	run := func(stage string) {
		overdue, reminded, err := s.SweepOverdue(ctx)
		if err != nil {
			slog.Error("厂家响应超时巡检失败", "stage", stage, "error", err)
			return
		}
		if overdue > 0 {
			slog.Info("厂家响应超时巡检完成", "stage", stage, "超时工单", overdue, "本次提醒", reminded)
		}
	}

	go func() {
		run("startup")
		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run("ticker")
			}
		}
	}()
}

// StopScheduler 停止定时巡检。
func (s *Service) StopScheduler() {
	if s.stopSweep != nil {
		s.stopSweep()
		s.stopSweep = nil
	}
}

// Overview 汇总质保与责任方概览。
func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	now := time.Now()

	coverage, err := s.repo.CountWarrantyCoverage(ctx, now)
	if err != nil {
		return nil, err
	}
	supplierTotal, err := s.repo.CountSuppliers(ctx)
	if err != nil {
		return nil, err
	}

	claimTotal, err := s.repo.CountClaims(ctx)
	if err != nil {
		return nil, err
	}
	inWarranty, err := s.repo.CountInWarrantyClaims(ctx)
	if err != nil {
		return nil, err
	}
	takenOver, err := s.repo.CountTakenOverClaims(ctx)
	if err != nil {
		return nil, err
	}
	manufacturerOpen, err := s.repo.CountManufacturerOpen(ctx)
	if err != nil {
		return nil, err
	}
	responseOverdue, err := s.repo.CountResponseOverdue(ctx, now)
	if err != nil {
		return nil, err
	}
	byStatus, err := s.repo.CountClaimsByColumn(ctx, "status")
	if err != nil {
		return nil, err
	}
	bySupplier, err := s.repo.CountClaimsBySupplier(ctx)
	if err != nil {
		return nil, err
	}
	overdueEntities, err := s.repo.ListResponseOverdue(ctx, now, 8)
	if err != nil {
		return nil, err
	}
	overdueRows, err := s.enrichClaims(ctx, overdueEntities)
	if err != nil {
		return nil, err
	}

	outWarranty := claimTotal - inWarranty
	result := &Overview{
		RegisteredLamps:  coverage.registered,
		WarrantyActive:   coverage.active,
		LampActive:       coverage.lampActive,
		PoleActive:       coverage.poleActive,
		ExpiringSoon:     coverage.expiring,
		SupplierTotal:    supplierTotal,
		ClaimTotal:       claimTotal,
		InWarrantyTotal:  inWarranty,
		OutWarrantyTotal: outWarranty,
		ManufacturerOpen: manufacturerOpen,
		ResponseOverdue:  responseOverdue,
		TakenOverTotal:   takenOver,
		ByClaimStatus:    byStatus,
		BySupplier:       bySupplier,
		OverdueClaims:    overdueRows,
		GeneratedAt:      now,
	}
	if claimTotal > 0 {
		result.InWarrantyRate = math.Round(float64(inWarranty)/float64(claimTotal)*1000) / 1000
	}
	return result, nil
}

// isValidClaimStatus 校验责任工单状态。
func isValidClaimStatus(status string) bool {
	for _, item := range ClaimStatuses() {
		if item == status {
			return true
		}
	}
	return false
}

// normalizeID 将空指针或 0 值统一为 nil。
func normalizeID(id *uint) *uint {
	if id == nil || *id == 0 {
		return nil
	}
	return id
}

// parseDate 解析 YYYY-MM-DD 日期。
func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", value, time.Local)
}

// round2 保留两位小数, 负值归零。
func round2(value float64) float64 {
	if value < 0 {
		value = 0
	}
	return math.Round(value*100) / 100
}
