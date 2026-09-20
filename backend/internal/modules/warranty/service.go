package warranty

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/fault"
	"streetlight/internal/modules/lamp"
	"streetlight/pkg/pagination"
)

// supplierSortSpec 定义供应商列表允许的排序字段白名单。
var supplierSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"name":       "name",
		"created_at": "created_at",
		"updated_at": "updated_at",
	},
	Default: "id",
}

// warrantySortSpec 定义质保登记列表允许的排序字段白名单。
var warrantySortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"lamp_code":  "lamp_code",
		"component":  "component",
		"start_date": "start_date",
		"end_date":   "end_date",
		"created_at": "created_at",
		"updated_at": "updated_at",
	},
	Default: "id",
}

// assignmentSortSpec 定义责任判定单列表允许的排序字段白名单。
var assignmentSortSpec = pagination.SortSpec{
	Allowed: map[string]string{
		"fault_no":    "fault_no",
		"assigned_at": "assigned_at",
		"deadline":    "deadline",
		"status":      "status",
		"created_at":  "created_at",
	},
	Default: "assigned_at",
}

// LampPort 由路灯台账模块实现, 质保模块通过它读取路灯档案。
type LampPort interface {
	Get(ctx context.Context, id uint) (*lamp.Lamp, error)
}

// Service 承载质保登记与责任方判定的业务规则。
type Service struct {
	repo  *Repository
	lamps LampPort
}

// NewService 构造质保与责任方服务。
func NewService(repo *Repository, lamps LampPort) *Service {
	return &Service{repo: repo, lamps: lamps}
}

// ---------- 供应商 ----------

// ListSuppliers 分页查询供应商。
func (s *Service) ListSuppliers(ctx context.Context, query SupplierListQuery) ([]Supplier, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, supplierSortSpec)
	items, total, err := s.repo.ListSuppliers(ctx, query.Keyword, page)
	if err != nil {
		return nil, 0, page, err
	}
	return items, total, page, nil
}

// SupplierOptions 返回全部供应商, 供质保登记表单下拉选择。
func (s *Service) SupplierOptions(ctx context.Context) ([]Supplier, error) {
	return s.repo.ListAllSuppliers(ctx)
}

// CreateSupplier 新增供应商, 名称全局唯一。
func (s *Service) CreateSupplier(ctx context.Context, req SupplierCreateRequest) (*Supplier, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, apperr.BadRequest("供应商名称不能为空")
	}
	exists, err := s.repo.ExistsSupplierByName(ctx, name, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperr.Conflict("供应商名称已存在: %s", name)
	}

	entity := &Supplier{
		Name:          name,
		ContactPerson: strings.TrimSpace(req.ContactPerson),
		ContactPhone:  strings.TrimSpace(req.ContactPhone),
		Email:         strings.TrimSpace(req.Email),
		Address:       strings.TrimSpace(req.Address),
		Remark:        strings.TrimSpace(req.Remark),
	}
	if err := s.repo.CreateSupplier(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// UpdateSupplier 更新供应商档案, 同步刷新引用它的质保记录上的厂家快照。
func (s *Service) UpdateSupplier(ctx context.Context, id uint, req SupplierUpdateRequest) (*Supplier, error) {
	entity, err := s.repo.GetSupplier(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, apperr.BadRequest("供应商名称不能为空")
		}
		if name != entity.Name {
			exists, err := s.repo.ExistsSupplierByName(ctx, name, id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, apperr.Conflict("供应商名称已存在: %s", name)
			}
			entity.Name = name
		}
	}
	if req.ContactPerson != nil {
		entity.ContactPerson = strings.TrimSpace(*req.ContactPerson)
	}
	if req.ContactPhone != nil {
		entity.ContactPhone = strings.TrimSpace(*req.ContactPhone)
	}
	if req.Email != nil {
		entity.Email = strings.TrimSpace(*req.Email)
	}
	if req.Address != nil {
		entity.Address = strings.TrimSpace(*req.Address)
	}
	if req.Remark != nil {
		entity.Remark = strings.TrimSpace(*req.Remark)
	}

	if err := s.repo.UpdateSupplier(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// DeleteSupplier 删除供应商, 存在引用它的质保记录时拒绝删除。
func (s *Service) DeleteSupplier(ctx context.Context, id uint) error {
	entity, err := s.repo.GetSupplier(ctx, id)
	if err != nil {
		return err
	}
	count, err := s.repo.CountWarrantiesBySupplier(ctx, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return apperr.Conflict("供应商 %s 已被 %d 条质保记录引用, 请先调整质保登记后再删除", entity.Name, count)
	}
	return s.repo.DeleteSupplier(ctx, id)
}

// ---------- 质保登记 ----------

// ListWarranties 分页查询质保登记。
func (s *Service) ListWarranties(ctx context.Context, query WarrantyListQuery) ([]Warranty, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, warrantySortSpec)
	filter, err := buildWarrantyFilter(query)
	if err != nil {
		return nil, 0, page, err
	}
	items, total, err := s.repo.ListWarranties(ctx, filter, page)
	if err != nil {
		return nil, 0, page, err
	}
	now := time.Now()
	for index := range items {
		items[index].FillState(now)
	}
	return items, total, page, nil
}

// GetWarranty 查询质保详情。
func (s *Service) GetWarranty(ctx context.Context, id uint) (*Warranty, error) {
	entity, err := s.repo.GetWarranty(ctx, id)
	if err != nil {
		return nil, err
	}
	entity.FillState(time.Now())
	return entity, nil
}

// CreateWarranty 为路灯的灯具或灯杆登记质保, 同一部件仅允许一条质保记录。
func (s *Service) CreateWarranty(ctx context.Context, req WarrantyCreateRequest) (*Warranty, error) {
	device, err := s.lamps.Get(ctx, req.LampID)
	if err != nil {
		return nil, err
	}

	component := strings.TrimSpace(req.Component)
	if !IsValidComponent(component) {
		return nil, apperr.BadRequest("非法的质保部件: %s", component)
	}

	exists, err := s.repo.ExistsWarranty(ctx, device.ID, component, 0)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperr.Conflict("路灯 %s 的%s已登记质保, 请直接编辑原记录", device.Code, ComponentLabel(component))
	}

	startDate, endDate, err := parseWarrantyPeriod(req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	entity := &Warranty{
		LampID:        device.ID,
		LampCode:      device.Code,
		Component:     component,
		ContactPerson: strings.TrimSpace(req.ContactPerson),
		ContactPhone:  strings.TrimSpace(req.ContactPhone),
		StartDate:     startDate,
		EndDate:       endDate,
		Remark:        strings.TrimSpace(req.Remark),
	}
	if err := s.applySupplier(ctx, entity, req.SupplierID); err != nil {
		return nil, err
	}

	if err := s.repo.CreateWarranty(ctx, entity); err != nil {
		return nil, err
	}
	entity.FillState(time.Now())
	return entity, nil
}

// UpdateWarranty 更新质保登记。
func (s *Service) UpdateWarranty(ctx context.Context, id uint, req WarrantyUpdateRequest) (*Warranty, error) {
	entity, err := s.repo.GetWarranty(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Component != nil {
		component := strings.TrimSpace(*req.Component)
		if !IsValidComponent(component) {
			return nil, apperr.BadRequest("非法的质保部件: %s", component)
		}
		if component != entity.Component {
			exists, err := s.repo.ExistsWarranty(ctx, entity.LampID, component, id)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, apperr.Conflict("路灯 %s 的%s已登记质保, 不允许重复", entity.LampCode, ComponentLabel(component))
			}
			entity.Component = component
		}
	}
	if req.StartDate != nil || req.EndDate != nil {
		startValue, endValue := entity.StartDate, entity.EndDate
		if req.StartDate != nil {
			parsed, err := parseDate(*req.StartDate)
			if err != nil {
				return nil, err
			}
			startValue = parsed
		}
		if req.EndDate != nil {
			parsed, err := parseDate(*req.EndDate)
			if err != nil {
				return nil, err
			}
			endValue = parsed
		}
		if startValue == nil || endValue == nil || endValue.Before(*startValue) {
			return nil, apperr.BadRequest("质保结束日期不能早于开始日期")
		}
		entity.StartDate = startValue
		entity.EndDate = endValue
	}
	if req.SupplierID != nil {
		if err := s.applySupplier(ctx, entity, req.SupplierID); err != nil {
			return nil, err
		}
	}
	if req.ContactPerson != nil {
		entity.ContactPerson = strings.TrimSpace(*req.ContactPerson)
	}
	if req.ContactPhone != nil {
		entity.ContactPhone = strings.TrimSpace(*req.ContactPhone)
	}
	if req.Remark != nil {
		entity.Remark = strings.TrimSpace(*req.Remark)
	}

	if err := s.repo.UpdateWarranty(ctx, entity); err != nil {
		return nil, err
	}
	entity.FillState(time.Now())
	return entity, nil
}

// DeleteWarranty 删除质保记录, 已生成的责任判定单保留厂家快照不受影响。
func (s *Service) DeleteWarranty(ctx context.Context, id uint) error {
	if _, err := s.repo.GetWarranty(ctx, id); err != nil {
		return err
	}
	return s.repo.DeleteWarranty(ctx, id)
}

// applySupplier 校验并回填供应商快照; 联系方式未显式填写时默认沿用供应商档案。
func (s *Service) applySupplier(ctx context.Context, entity *Warranty, supplierID *uint) error {
	if supplierID == nil || *supplierID == 0 {
		entity.SupplierID = nil
		entity.SupplierName = ""
		return nil
	}
	supplier, err := s.repo.GetSupplier(ctx, *supplierID)
	if err != nil {
		return err
	}
	entity.SupplierID = &supplier.ID
	entity.SupplierName = supplier.Name
	if entity.ContactPerson == "" {
		entity.ContactPerson = supplier.ContactPerson
	}
	if entity.ContactPhone == "" {
		entity.ContactPhone = supplier.ContactPhone
	}
	return nil
}

// ---------- 责任方判定 ----------

// DecideForFault 实现故障模块的 ResponsibilityDecider 端口:
// 登记故障时按部件质保自动判定责任方, 质保期内指派厂家, 无质保或超期转自有班组。
func (s *Service) DecideForFault(ctx context.Context, entity *fault.Fault) (*fault.ResponsibilityInfo, error) {
	component := ComponentForFaultType(entity.FaultType)
	now := time.Now()

	assignment := &FaultAssignment{
		FaultID:    entity.ID,
		FaultNo:    entity.FaultNo,
		LampID:     entity.LampID,
		LampCode:   entity.LampCode,
		Component:  component,
		AssignedAt: now,
		Deadline:   now.Add(ResponseTimeout),
		Status:     AssignPending,
	}

	warranty, err := s.repo.GetWarrantyByLampComponent(ctx, entity.LampID, component)
	if err != nil {
		return nil, err
	}

	switch {
	case warranty == nil:
		assignment.ResponsibleType = ResponsibleOwnTeam
		assignment.Reason = fmt.Sprintf("路灯 %s 未登记%s质保, 由自有班组处理", entity.LampCode, ComponentLabel(component))
	case !warranty.Covers(entity.ReportedAt):
		assignment.ResponsibleType = ResponsibleOwnTeam
		assignment.WarrantyID = &warranty.ID
		assignment.SupplierID = warranty.SupplierID
		assignment.SupplierName = warranty.SupplierName
		assignment.Reason = fmt.Sprintf("%s质保已于 %s 到期, 由自有班组处理",
			ComponentLabel(component), warranty.EndDate.Format("2006-01-02"))
	default:
		assignment.ResponsibleType = ResponsibleSupplier
		assignment.InWarranty = true
		assignment.WarrantyID = &warranty.ID
		assignment.SupplierID = warranty.SupplierID
		assignment.SupplierName = warranty.SupplierName
		assignment.ContactPerson = warranty.ContactPerson
		assignment.ContactPhone = warranty.ContactPhone
		assignment.Reason = fmt.Sprintf("%s质保期内(%s ~ %s), 指派厂家处理",
			ComponentLabel(component),
			warranty.StartDate.Format("2006-01-02"),
			warranty.EndDate.Format("2006-01-02"))
	}

	if err := s.repo.CreateAssignment(ctx, assignment); err != nil {
		return nil, err
	}
	return responsibilityOf(assignment), nil
}

// responsibilityOf 将判定单转换为故障模块使用的摘要结构。
func responsibilityOf(assignment *FaultAssignment) *fault.ResponsibilityInfo {
	name := assignment.SupplierName
	if assignment.ResponsibleType == ResponsibleOwnTeam || name == "" {
		name = "自有班组"
	}
	return &fault.ResponsibilityInfo{
		ResponsibleType: assignment.ResponsibleType,
		ResponsibleName: name,
		Component:       assignment.Component,
		InWarranty:      assignment.InWarranty,
	}
}

// ListAssignments 分页查询责任判定单, 查询前自动清扫厂家超时单并提醒。
func (s *Service) ListAssignments(ctx context.Context, query AssignmentListQuery) ([]FaultAssignment, int64, pagination.Query, error) {
	page := pagination.Parse(query.Params, assignmentSortSpec)
	filter, err := buildAssignmentFilter(query)
	if err != nil {
		return nil, 0, page, err
	}
	s.sweepOverdue(ctx)
	items, total, err := s.repo.ListAssignments(ctx, filter, page)
	if err != nil {
		return nil, 0, page, err
	}
	return items, total, page, nil
}

// GetAssignmentByFault 查询指定故障的责任判定单。
func (s *Service) GetAssignmentByFault(ctx context.Context, faultID uint) (*FaultAssignment, error) {
	s.sweepOverdue(ctx)
	entity, err := s.repo.GetAssignmentByFault(ctx, faultID)
	if err != nil {
		return nil, err
	}
	if entity == nil {
		return nil, apperr.NotFound("故障 id=%d 尚未生成责任判定", faultID)
	}
	return entity, nil
}

// TransferAssignment 厂家处理超时(或厂家负责中)的故障转由自有班组接手。
func (s *Service) TransferAssignment(ctx context.Context, id uint, req TransferRequest) (*FaultAssignment, error) {
	entity, err := s.repo.GetAssignment(ctx, id)
	if err != nil {
		return nil, err
	}
	if entity.ResponsibleType != ResponsibleSupplier {
		return nil, apperr.Conflict("故障 %s 已由自有班组负责, 无需转派", entity.FaultNo)
	}
	if entity.Status == AssignTransferred {
		return nil, apperr.Conflict("故障 %s 已转自有班组接手, 请勿重复操作", entity.FaultNo)
	}

	now := time.Now()
	entity.ResponsibleType = ResponsibleOwnTeam
	entity.Status = AssignTransferred
	entity.TransferredAt = &now
	entity.TransferRemark = strings.TrimSpace(req.Remark)

	if err := s.repo.UpdateAssignment(ctx, entity); err != nil {
		return nil, err
	}
	return entity, nil
}

// sweepOverdue 厂家处理超时自动提醒: 将超过响应时限仍未处理的厂家单置为超时提醒。
func (s *Service) sweepOverdue(ctx context.Context) {
	affected, err := s.repo.SweepOverdueAssignments(ctx, time.Now())
	if err != nil {
		slog.Warn("清扫厂家超时判定单失败", "error", err)
		return
	}
	if affected > 0 {
		slog.Info("厂家处理超时, 已自动提醒", "count", affected)
	}
}

// ---------- 概览与字典 ----------

// Overview 汇总质保与责任方概览: 质保内维修占比、厂家响应超时数量等。
func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	now := time.Now()
	s.sweepOverdue(ctx)

	supplierTotal, err := s.repo.CountSuppliers(ctx)
	if err != nil {
		return nil, err
	}
	warrantyTotal, err := s.repo.CountWarranties(ctx)
	if err != nil {
		return nil, err
	}
	warrantyByState, err := s.repo.CountWarrantiesByState(ctx, now)
	if err != nil {
		return nil, err
	}
	assignmentTotal, err := s.repo.CountAssignments(ctx)
	if err != nil {
		return nil, err
	}
	byResponsible, err := s.repo.CountAssignmentsByColumn(ctx, "responsible_type")
	if err != nil {
		return nil, err
	}
	byStatus, err := s.repo.CountAssignmentsByColumn(ctx, "status")
	if err != nil {
		return nil, err
	}
	repairTotal, err := s.repo.CountRepairs(ctx)
	if err != nil {
		return nil, err
	}
	inWarrantyRepair, err := s.repo.CountInWarrantyRepairs(ctx)
	if err != nil {
		return nil, err
	}
	overdueAssignments, err := s.repo.ListOverdueAssignments(ctx, 8)
	if err != nil {
		return nil, err
	}

	var ratio float64
	if repairTotal > 0 {
		ratio = float64(inWarrantyRepair) / float64(repairTotal) * 100
	}

	return &Overview{
		SupplierTotal:      supplierTotal,
		WarrantyTotal:      warrantyTotal,
		WarrantyByState:    warrantyByState,
		AssignmentTotal:    assignmentTotal,
		SupplierAssigned:   byResponsible[ResponsibleSupplier],
		OwnTeamAssigned:    byResponsible[ResponsibleOwnTeam],
		OverdueTotal:       byStatus[AssignOverdue],
		TransferredTotal:   byStatus[AssignTransferred],
		RepairTotal:        repairTotal,
		InWarrantyRepair:   inWarrantyRepair,
		InWarrantyRatio:    round1(ratio),
		ResponseTimeoutHr:  ResponseTimeout.Hours(),
		OverdueAssignments: overdueAssignments,
		GeneratedAt:        now,
	}, nil
}

// Metadata 返回质保模块字典。
func (s *Service) Metadata() *Meta {
	return &Meta{
		Components:         Components(),
		ResponsibleTypes:   ResponsibleTypes(),
		AssignmentStatuses: AssignmentStatuses(),
		ResponseTimeoutHr:  ResponseTimeout.Hours(),
	}
}

// ComponentLabel 返回部件中文名称。
func ComponentLabel(component string) string {
	switch component {
	case ComponentPole:
		return "灯杆"
	default:
		return "灯具"
	}
}

// buildWarrantyFilter 将查询参数转换为仓储条件。
func buildWarrantyFilter(query WarrantyListQuery) (WarrantyFilter, error) {
	filter := WarrantyFilter{
		Keyword:   strings.TrimSpace(query.Keyword),
		Component: strings.TrimSpace(query.Component),
		State:     strings.TrimSpace(query.State),
	}
	if filter.Component != "" && !IsValidComponent(filter.Component) {
		return filter, apperr.BadRequest("非法的质保部件: %s", filter.Component)
	}
	switch filter.State {
	case "", StateActive, StateExpiring, StateExpired:
	default:
		return filter, apperr.BadRequest("非法的质保状态: %s", filter.State)
	}
	now := time.Now()
	filter.Today = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return filter, nil
}

// buildAssignmentFilter 将查询参数转换为仓储条件。
func buildAssignmentFilter(query AssignmentListQuery) (AssignmentFilter, error) {
	filter := AssignmentFilter{
		Keyword:         strings.TrimSpace(query.Keyword),
		ResponsibleType: strings.TrimSpace(query.ResponsibleType),
		Status:          strings.TrimSpace(query.Status),
		Component:       strings.TrimSpace(query.Component),
	}
	if filter.ResponsibleType != "" && !IsValidResponsibleType(filter.ResponsibleType) {
		return filter, apperr.BadRequest("非法的责任方类型: %s", filter.ResponsibleType)
	}
	if filter.Status != "" && !IsValidAssignmentStatus(filter.Status) {
		return filter, apperr.BadRequest("非法的判定单状态: %s", filter.Status)
	}
	if filter.Component != "" && !IsValidComponent(filter.Component) {
		return filter, apperr.BadRequest("非法的质保部件: %s", filter.Component)
	}
	return filter, nil
}

// parseWarrantyPeriod 解析质保起止日期并校验先后关系。
func parseWarrantyPeriod(startValue, endValue string) (*time.Time, *time.Time, error) {
	startDate, err := parseDate(startValue)
	if err != nil {
		return nil, nil, err
	}
	endDate, err := parseDate(endValue)
	if err != nil {
		return nil, nil, err
	}
	if startDate == nil || endDate == nil {
		return nil, nil, apperr.BadRequest("质保起止日期不能为空")
	}
	if endDate.Before(*startDate) {
		return nil, nil, apperr.BadRequest("质保结束日期不能早于开始日期")
	}
	return startDate, endDate, nil
}

// parseDate 解析 YYYY-MM-DD 格式的日期。
func parseDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	date, err := time.ParseInLocation("2006-01-02", value, time.Local)
	if err != nil {
		return nil, apperr.BadRequest("日期格式应为 YYYY-MM-DD, 当前值: %s", value)
	}
	return &date, nil
}

// round1 保留一位小数。
func round1(value float64) float64 {
	return float64(int(value*10+0.5)) / 10
}
