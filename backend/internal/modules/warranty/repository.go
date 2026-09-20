package warranty

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"streetlight/internal/apperr"
	"streetlight/internal/modules/repair"
	"streetlight/pkg/pagination"
)

// WarrantyFilter 是仓储层使用的质保查询条件, 状态边界已在服务层换算为日期。
type WarrantyFilter struct {
	Keyword   string
	Component string
	State     string
	Today     time.Time // 状态过滤使用的当天零点
}

// AssignmentFilter 是仓储层使用的判定单查询条件。
type AssignmentFilter struct {
	Keyword         string
	ResponsibleType string
	Status          string
	Component       string
}

// Repository 负责供应商、质保登记与责任判定单的数据访问。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造质保模块仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// ---------- 供应商 ----------

// CreateSupplier 新增供应商。
func (r *Repository) CreateSupplier(ctx context.Context, entity *Supplier) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		return fmt.Errorf("新增供应商失败: %w", err)
	}
	return nil
}

// UpdateSupplier 保存供应商全部字段。
func (r *Repository) UpdateSupplier(ctx context.Context, entity *Supplier) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		return fmt.Errorf("更新供应商失败: %w", err)
	}
	return nil
}

// DeleteSupplier 按主键删除供应商。
func (r *Repository) DeleteSupplier(ctx context.Context, id uint) error {
	if err := r.session(ctx).Delete(&Supplier{}, id).Error; err != nil {
		return fmt.Errorf("删除供应商失败: %w", err)
	}
	return nil
}

// GetSupplier 按主键查询供应商。
func (r *Repository) GetSupplier(ctx context.Context, id uint) (*Supplier, error) {
	var entity Supplier
	err := r.session(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("供应商不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询供应商失败: %w", err)
	}
	return &entity, nil
}

// ExistsSupplierByName 判断供应商名称是否已被占用, excludeID 用于更新场景排除自身。
func (r *Repository) ExistsSupplierByName(ctx context.Context, name string, excludeID uint) (bool, error) {
	query := r.session(ctx).Model(&Supplier{}).Where("name = ?", strings.TrimSpace(name))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("校验供应商名称失败: %w", err)
	}
	return count > 0, nil
}

// ListSuppliers 分页查询供应商。
func (r *Repository) ListSuppliers(ctx context.Context, keyword string, page pagination.Query) ([]Supplier, int64, error) {
	base := func() *gorm.DB {
		statement := r.session(ctx).Model(&Supplier{})
		if value := strings.TrimSpace(keyword); value != "" {
			like := "%" + value + "%"
			statement = statement.Where("name LIKE ? OR contact_person LIKE ? OR contact_phone LIKE ?", like, like, like)
		}
		return statement
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计供应商总数失败: %w", err)
	}

	entities := make([]Supplier, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询供应商列表失败: %w", err)
	}
	return entities, total, nil
}

// ListAllSuppliers 查询全部供应商, 供下拉选项使用。
func (r *Repository) ListAllSuppliers(ctx context.Context) ([]Supplier, error) {
	entities := make([]Supplier, 0)
	if err := r.session(ctx).Order("name ASC, id ASC").Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("查询供应商选项失败: %w", err)
	}
	return entities, nil
}

// CountSuppliers 统计供应商总数。
func (r *Repository) CountSuppliers(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&Supplier{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计供应商总数失败: %w", err)
	}
	return total, nil
}

// CountWarrantiesBySupplier 统计引用某供应商的质保记录数量。
func (r *Repository) CountWarrantiesBySupplier(ctx context.Context, supplierID uint) (int64, error) {
	var count int64
	err := r.session(ctx).Model(&Warranty{}).Where("supplier_id = ?", supplierID).Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计供应商质保记录失败: %w", err)
	}
	return count, nil
}

// ---------- 质保登记 ----------

// CreateWarranty 新增质保记录。
func (r *Repository) CreateWarranty(ctx context.Context, entity *Warranty) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		return fmt.Errorf("登记质保失败: %w", err)
	}
	return nil
}

// UpdateWarranty 保存质保全部字段。
func (r *Repository) UpdateWarranty(ctx context.Context, entity *Warranty) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		return fmt.Errorf("更新质保失败: %w", err)
	}
	return nil
}

// DeleteWarranty 按主键删除质保记录。
func (r *Repository) DeleteWarranty(ctx context.Context, id uint) error {
	if err := r.session(ctx).Delete(&Warranty{}, id).Error; err != nil {
		return fmt.Errorf("删除质保失败: %w", err)
	}
	return nil
}

// GetWarranty 按主键查询质保记录。
func (r *Repository) GetWarranty(ctx context.Context, id uint) (*Warranty, error) {
	var entity Warranty
	err := r.session(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("质保记录不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询质保失败: %w", err)
	}
	return &entity, nil
}

// GetWarrantyByLampComponent 查询某盏路灯指定部件的质保记录, 不存在时返回 nil。
func (r *Repository) GetWarrantyByLampComponent(ctx context.Context, lampID uint, component string) (*Warranty, error) {
	var entity Warranty
	err := r.session(ctx).
		Where("lamp_id = ? AND component = ?", lampID, component).
		First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询路灯部件质保失败: %w", err)
	}
	return &entity, nil
}

// ExistsWarranty 判断某盏路灯指定部件是否已登记质保, excludeID 用于更新场景排除自身。
func (r *Repository) ExistsWarranty(ctx context.Context, lampID uint, component string, excludeID uint) (bool, error) {
	query := r.session(ctx).Model(&Warranty{}).Where("lamp_id = ? AND component = ?", lampID, component)
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("校验质保记录失败: %w", err)
	}
	return count > 0, nil
}

// ListWarranties 分页查询质保记录。
func (r *Repository) ListWarranties(ctx context.Context, filter WarrantyFilter, page pagination.Query) ([]Warranty, int64, error) {
	base := func() *gorm.DB {
		return applyWarrantyFilter(r.session(ctx).Model(&Warranty{}), filter)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计质保总数失败: %w", err)
	}

	entities := make([]Warranty, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询质保列表失败: %w", err)
	}
	return entities, total, nil
}

// CountWarranties 统计质保记录总数。
func (r *Repository) CountWarranties(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&Warranty{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计质保总数失败: %w", err)
	}
	return total, nil
}

// CountWarrantiesByState 按质保状态(在保/临期/已到期)分组统计。
func (r *Repository) CountWarrantiesByState(ctx context.Context, now time.Time) (map[string]int64, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	expiringBound := today.Add(ExpiringWindow)

	countWhere := func(where string, args ...any) (int64, error) {
		var count int64
		if err := r.session(ctx).Model(&Warranty{}).Where(where, args...).Count(&count).Error; err != nil {
			return 0, fmt.Errorf("统计质保状态失败: %w", err)
		}
		return count, nil
	}

	expired, err := countWhere("end_date < ?", today)
	if err != nil {
		return nil, err
	}
	expiring, err := countWhere("end_date >= ? AND end_date < ?", today, expiringBound)
	if err != nil {
		return nil, err
	}
	active, err := countWhere("end_date >= ?", expiringBound)
	if err != nil {
		return nil, err
	}
	return map[string]int64{
		StateActive:   active,
		StateExpiring: expiring,
		StateExpired:  expired,
	}, nil
}

// applyWarrantyFilter 统一拼装质保列表查询条件。
func applyWarrantyFilter(statement *gorm.DB, filter WarrantyFilter) *gorm.DB {
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		statement = statement.Where("lamp_code LIKE ? OR supplier_name LIKE ?", like, like)
	}
	if filter.Component != "" {
		statement = statement.Where("component = ?", filter.Component)
	}
	today := filter.Today
	switch filter.State {
	case StateExpired:
		statement = statement.Where("end_date < ?", today)
	case StateExpiring:
		statement = statement.Where("end_date >= ? AND end_date < ?", today, today.Add(ExpiringWindow))
	case StateActive:
		statement = statement.Where("end_date >= ?", today.Add(ExpiringWindow))
	}
	return statement
}

// ---------- 责任判定单 ----------

// CreateAssignment 新增责任判定单。
func (r *Repository) CreateAssignment(ctx context.Context, entity *FaultAssignment) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		return fmt.Errorf("生成责任判定失败: %w", err)
	}
	return nil
}

// UpdateAssignment 保存判定单全部字段。
func (r *Repository) UpdateAssignment(ctx context.Context, entity *FaultAssignment) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		return fmt.Errorf("更新责任判定失败: %w", err)
	}
	return nil
}

// GetAssignment 按主键查询判定单。
func (r *Repository) GetAssignment(ctx context.Context, id uint) (*FaultAssignment, error) {
	var entity FaultAssignment
	err := r.session(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("责任判定单不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询责任判定失败: %w", err)
	}
	return &entity, nil
}

// GetAssignmentByFault 按故障 ID 查询判定单, 不存在时返回 nil。
func (r *Repository) GetAssignmentByFault(ctx context.Context, faultID uint) (*FaultAssignment, error) {
	var entity FaultAssignment
	err := r.session(ctx).Where("fault_id = ?", faultID).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询故障责任判定失败: %w", err)
	}
	return &entity, nil
}

// ListAssignments 分页查询责任判定单。
func (r *Repository) ListAssignments(ctx context.Context, filter AssignmentFilter, page pagination.Query) ([]FaultAssignment, int64, error) {
	base := func() *gorm.DB {
		return applyAssignmentFilter(r.session(ctx).Model(&FaultAssignment{}), filter)
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计责任判定总数失败: %w", err)
	}

	entities := make([]FaultAssignment, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询责任判定列表失败: %w", err)
	}
	return entities, total, nil
}

// ListOverdueAssignments 查询当前处于超时提醒状态的判定单, 按超时时刻正序。
func (r *Repository) ListOverdueAssignments(ctx context.Context, limit int) ([]FaultAssignment, error) {
	entities := make([]FaultAssignment, 0)
	err := r.session(ctx).
		Where("status = ?", AssignOverdue).
		Order("deadline ASC, id ASC").
		Limit(limit).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("查询厂家超时清单失败: %w", err)
	}
	return entities, nil
}

// SweepOverdueAssignments 将超时的厂家判定单批量置为"已超时提醒", 返回受影响条数。
// 这是"厂家处理超时自动提醒"的落点: 列表与概览查询前触发, reminded_at 即提醒时间。
func (r *Repository) SweepOverdueAssignments(ctx context.Context, now time.Time) (int64, error) {
	result := r.session(ctx).Model(&FaultAssignment{}).
		Where("responsible_type = ? AND status = ? AND deadline < ?", ResponsibleSupplier, AssignPending, now).
		Updates(map[string]any{
			"status":      AssignOverdue,
			"reminded_at": now,
		})
	if result.Error != nil {
		return 0, fmt.Errorf("清扫厂家超时判定单失败: %w", result.Error)
	}
	return result.RowsAffected, nil
}

// CountAssignments 统计判定单总数。
func (r *Repository) CountAssignments(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&FaultAssignment{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计责任判定总数失败: %w", err)
	}
	return total, nil
}

// CountAssignmentsByColumn 按列分组统计判定单, column 仅允许来自内部常量。
func (r *Repository) CountAssignmentsByColumn(ctx context.Context, column string) (map[string]int64, error) {
	type row struct {
		Label string
		Total int64
	}
	rows := make([]row, 0)
	err := r.session(ctx).Model(&FaultAssignment{}).
		Select(column + " AS label, COUNT(*) AS total").
		Group(column).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("分组统计责任判定 %s 失败: %w", column, err)
	}
	result := make(map[string]int64, len(rows))
	for _, item := range rows {
		result[item.Label] = item.Total
	}
	return result, nil
}

// CountRepairs 统计维修记录总数(跨模块只读, 用于质保内维修占比)。
func (r *Repository) CountRepairs(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&repair.Repair{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计维修记录总数失败: %w", err)
	}
	return total, nil
}

// CountInWarrantyRepairs 统计质保内故障产生的维修记录数量。
func (r *Repository) CountInWarrantyRepairs(ctx context.Context) (int64, error) {
	var total int64
	err := r.session(ctx).Model(&repair.Repair{}).
		Joins("JOIN fault_assignment ON fault_assignment.fault_id = repair.fault_id AND fault_assignment.in_warranty = ?", true).
		Count(&total).Error
	if err != nil {
		return 0, fmt.Errorf("统计质保内维修记录失败: %w", err)
	}
	return total, nil
}

// applyAssignmentFilter 统一拼装判定单列表查询条件。
func applyAssignmentFilter(statement *gorm.DB, filter AssignmentFilter) *gorm.DB {
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		statement = statement.Where(
			"fault_no LIKE ? OR lamp_code LIKE ? OR supplier_name LIKE ?",
			like, like, like,
		)
	}
	if filter.ResponsibleType != "" {
		statement = statement.Where("responsible_type = ?", filter.ResponsibleType)
	}
	if filter.Status != "" {
		statement = statement.Where("status = ?", filter.Status)
	}
	if filter.Component != "" {
		statement = statement.Where("component = ?", filter.Component)
	}
	return statement
}
