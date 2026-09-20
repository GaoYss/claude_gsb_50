package warranty

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"streetlight/internal/apperr"
	"streetlight/pkg/pagination"
)

// Repository 负责质保供应商、质保登记与责任工单的数据访问。
type Repository struct {
	db *gorm.DB
}

// NewRepository 构造质保管理仓储。
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) session(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

// isUniqueViolation 兼容 sqlite 与 postgres 的唯一约束冲突判断。
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "duplicate key") ||
		strings.Contains(message, "unique violation")
}

// ---------------------------------------------------------------------------
// 供应商
// ---------------------------------------------------------------------------

// CreateSupplier 新增供应商。
func (r *Repository) CreateSupplier(ctx context.Context, entity *Supplier) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("供应商名称已存在: %s", entity.Name)
		}
		return fmt.Errorf("新增供应商失败: %w", err)
	}
	return nil
}

// UpdateSupplier 保存供应商全部字段。
func (r *Repository) UpdateSupplier(ctx context.Context, entity *Supplier) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("供应商名称已存在: %s", entity.Name)
		}
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

// GetSuppliersByIDs 批量查询供应商。
func (r *Repository) GetSuppliersByIDs(ctx context.Context, ids []uint) (map[uint]*Supplier, error) {
	result := make(map[uint]*Supplier, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	entities := make([]Supplier, 0, len(ids))
	if err := r.session(ctx).Where("id IN ?", ids).Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("批量查询供应商失败: %w", err)
	}
	for index := range entities {
		result[entities[index].ID] = &entities[index]
	}
	return result, nil
}

// ListSuppliers 分页查询供应商。
func (r *Repository) ListSuppliers(ctx context.Context, keyword string, page pagination.Query) ([]Supplier, int64, error) {
	base := func() *gorm.DB {
		statement := r.session(ctx).Model(&Supplier{})
		if value := strings.TrimSpace(keyword); value != "" {
			like := "%" + value + "%"
			statement = statement.Where(
				"name LIKE ? OR short_name LIKE ? OR contact_person LIKE ? OR contact_phone LIKE ? OR service_phone LIKE ?",
				like, like, like, like, like,
			)
		}
		return statement
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计供应商数量失败: %w", err)
	}

	entities := make([]Supplier, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询供应商列表失败: %w", err)
	}
	return entities, total, nil
}

// ListAllSuppliers 查询全部供应商(下拉选项使用)。
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
		return 0, fmt.Errorf("统计供应商数量失败: %w", err)
	}
	return total, nil
}

// ExistsSupplierName 判断供应商名称是否被占用。
func (r *Repository) ExistsSupplierName(ctx context.Context, name string, excludeID uint) (bool, error) {
	var count int64
	query := r.session(ctx).Model(&Supplier{}).Where("name = ?", strings.TrimSpace(name))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, fmt.Errorf("校验供应商名称失败: %w", err)
	}
	return count > 0, nil
}

// CountWarrantyRefsBySupplier 统计供应商被质保登记引用的数量。
func (r *Repository) CountWarrantyRefsBySupplier(ctx context.Context, supplierID uint) (int64, error) {
	var count int64
	err := r.session(ctx).Model(&Warranty{}).
		Where("lamp_supplier_id = ? OR pole_supplier_id = ?", supplierID, supplierID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计供应商质保引用失败: %w", err)
	}
	return count, nil
}

// CountOpenClaimRefsBySupplier 统计供应商仍未闭环的责任工单数量。
func (r *Repository) CountOpenClaimRefsBySupplier(ctx context.Context, supplierID uint) (int64, error) {
	var count int64
	err := r.session(ctx).Model(&WarrantyClaim{}).
		Where("supplier_id = ? AND status <> ?", supplierID, ClaimStatusClosed).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("统计供应商工单引用失败: %w", err)
	}
	return count, nil
}

// ---------------------------------------------------------------------------
// 质保登记
// ---------------------------------------------------------------------------

// CreateWarranty 新增质保登记。
func (r *Repository) CreateWarranty(ctx context.Context, entity *Warranty) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("该路灯已登记质保信息, 请直接修改")
		}
		return fmt.Errorf("新增质保登记失败: %w", err)
	}
	return nil
}

// UpdateWarranty 保存质保登记全部字段。
func (r *Repository) UpdateWarranty(ctx context.Context, entity *Warranty) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		return fmt.Errorf("更新质保登记失败: %w", err)
	}
	return nil
}

// DeleteWarranty 按主键删除质保登记。
func (r *Repository) DeleteWarranty(ctx context.Context, id uint) error {
	if err := r.session(ctx).Delete(&Warranty{}, id).Error; err != nil {
		return fmt.Errorf("删除质保登记失败: %w", err)
	}
	return nil
}

// GetWarranty 按主键查询质保登记。
func (r *Repository) GetWarranty(ctx context.Context, id uint) (*Warranty, error) {
	var entity Warranty
	err := r.session(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("质保登记不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询质保登记失败: %w", err)
	}
	return &entity, nil
}

// GetWarrantyByLamp 按路灯查询质保登记, 不存在时返回 nil。
func (r *Repository) GetWarrantyByLamp(ctx context.Context, lampID uint) (*Warranty, error) {
	var entity Warranty
	err := r.session(ctx).Where("lamp_id = ?", lampID).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询路灯质保登记失败: %w", err)
	}
	return &entity, nil
}

// warrantyFilter 是仓储层的质保登记查询条件。
type warrantyFilter struct {
	Keyword       string
	SupplierID    uint
	Component     string
	WarrantyState string
	RoadName      string
}

// ListWarranties 分页查询质保登记。
func (r *Repository) ListWarranties(ctx context.Context, filter warrantyFilter, page pagination.Query) ([]Warranty, int64, error) {
	base := func() *gorm.DB {
		statement := r.session(ctx).Model(&Warranty{})
		if value := strings.TrimSpace(filter.Keyword); value != "" {
			like := "%" + value + "%"
			statement = statement.Where(
				"lamp_code LIKE ? OR road_name LIKE ? OR lamp_supplier_name LIKE ? OR pole_supplier_name LIKE ?",
				like, like, like, like,
			)
		}
		if filter.SupplierID > 0 {
			statement = statement.Where("lamp_supplier_id = ? OR pole_supplier_id = ?", filter.SupplierID, filter.SupplierID)
		}
		if value := strings.TrimSpace(filter.RoadName); value != "" {
			statement = statement.Where("road_name = ?", value)
		}
		if cond, args := warrantyStateCondition(filter.Component, filter.WarrantyState); cond != "" {
			statement = statement.Where(cond, args...)
		}
		return statement
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计质保登记数量失败: %w", err)
	}

	entities := make([]Warranty, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询质保登记列表失败: %w", err)
	}
	return entities, total, nil
}

// CountWarranties 统计已登记质保的路灯数量。
func (r *Repository) CountWarranties(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&Warranty{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计质保登记数量失败: %w", err)
	}
	return total, nil
}

// warrantyCoverage 是质保覆盖统计的中间结果。
type warrantyCoverage struct {
	registered int64
	active     int64 // 灯具或灯杆至少一项在质保期内
	lampActive int64
	poleActive int64
	expiring   int64 // 30 天内有部件到期且当前在保
}

// CountWarrantyCoverage 统计质保覆盖情况, 日期按天比较, 截止日当天仍计入在保。
func (r *Repository) CountWarrantyCoverage(ctx context.Context, now time.Time) (warrantyCoverage, error) {
	today := truncateDay(now)
	soon := today.AddDate(0, 0, ExpiringDays)
	todayText := today.Format("2006-01-02")
	soonText := soon.Format("2006-01-02")

	lampActiveCond := activeCondition(ComponentLamp)
	poleActiveCond := activeCondition(ComponentPole)
	lampExpiringCond := expiringCondition(ComponentLamp)
	poleExpiringCond := expiringCondition(ComponentPole)

	type counts struct {
		Registered int64
		Active     int64
		LampActive int64
		PoleActive int64
		Expiring   int64
	}
	var result counts
	err := r.session(ctx).Model(&Warranty{}).Select(
		"COUNT(*) AS registered, "+
			"COUNT(CASE WHEN "+lampActiveCond+" OR "+poleActiveCond+" THEN 1 END) AS active, "+
			"COUNT(CASE WHEN "+lampActiveCond+" THEN 1 END) AS lamp_active, "+
			"COUNT(CASE WHEN "+poleActiveCond+" THEN 1 END) AS pole_active, "+
			"COUNT(CASE WHEN "+lampExpiringCond+" OR "+poleExpiringCond+" THEN 1 END) AS expiring",
		todayText, todayText, todayText, todayText,
		todayText, todayText, todayText, todayText,
		todayText, soonText, todayText,
		todayText, soonText, todayText,
	).Scan(&result).Error
	if err != nil {
		return warrantyCoverage{}, fmt.Errorf("统计质保覆盖情况失败: %w", err)
	}
	return warrantyCoverage{
		registered: result.Registered,
		active:     result.Active,
		lampActive: result.LampActive,
		poleActive: result.PoleActive,
		expiring:   result.Expiring,
	}, nil
}

// activeCondition 返回指定部件当前在保的 SQL 片段, 参数依次为截止日、起算日。
func activeCondition(prefix string) string {
	return fmt.Sprintf(
		"(%s_supplier_id IS NOT NULL AND %s_end_at IS NOT NULL AND %s_end_at >= ? "+
			"AND (%s_start_at IS NULL OR %s_start_at <= ?))",
		prefix, prefix, prefix, prefix, prefix,
	)
}

// expiringCondition 返回指定部件在 (今天, 30 天后] 窗口内到期且当前在保的 SQL 片段, 参数依次为今天、窗口截止、今天。
func expiringCondition(prefix string) string {
	return fmt.Sprintf(
		"(%s_supplier_id IS NOT NULL AND %s_end_at IS NOT NULL AND %s_end_at > ? AND %s_end_at <= ? "+
			"AND (%s_start_at IS NULL OR %s_start_at <= ?))",
		prefix, prefix, prefix, prefix, prefix, prefix,
	)
}

// warrantyStateCondition 生成质保状态过滤条件, 列名取自内部常量, 不存在状态时返回空串。
func warrantyStateCondition(component, state string) (string, []any) {
	state = strings.TrimSpace(state)
	if state == "" {
		return "", nil
	}
	today := truncateDay(time.Now())
	soon := today.AddDate(0, 0, ExpiringDays)

	one := func(prefix string) (string, []any) {
		start := prefix + "_start_at"
		end := prefix + "_end_at"
		supplier := prefix + "_supplier_id"
		switch state {
		case "active":
			return fmt.Sprintf(
				"%s IS NOT NULL AND %s IS NOT NULL AND %s >= ? AND (%s IS NULL OR %s <= ?)",
				supplier, end, end, start, start,
			), []any{today, today}
		case "expired":
			return fmt.Sprintf("%s IS NOT NULL AND %s < ?", end, end), []any{today}
		case "expiring":
			return fmt.Sprintf(
				"%s IS NOT NULL AND %s IS NOT NULL AND %s >= ? AND %s <= ? AND (%s IS NULL OR %s <= ?)",
				supplier, end, end, end, start, start,
			), []any{today, soon, today}
		default:
			return "", nil
		}
	}

	switch component {
	case ComponentLamp, ComponentPole:
		return one(component)
	default:
		lampCond, lampArgs := one(ComponentLamp)
		poleCond, poleArgs := one(ComponentPole)
		if lampCond == "" {
			return "", nil
		}
		return "(" + lampCond + " OR " + poleCond + ")", append(lampArgs, poleArgs...)
	}
}

// ---------------------------------------------------------------------------
// 责任工单
// ---------------------------------------------------------------------------

// CreateClaim 新增责任工单。
func (r *Repository) CreateClaim(ctx context.Context, entity *WarrantyClaim) error {
	if err := r.session(ctx).Create(entity).Error; err != nil {
		if isUniqueViolation(err) {
			return apperr.Conflict("故障 %s 已存在责任工单", entity.FaultNo)
		}
		return fmt.Errorf("新增责任工单失败: %w", err)
	}
	return nil
}

// UpdateClaim 保存责任工单全部字段。
func (r *Repository) UpdateClaim(ctx context.Context, entity *WarrantyClaim) error {
	if err := r.session(ctx).Save(entity).Error; err != nil {
		return fmt.Errorf("更新责任工单失败: %w", err)
	}
	return nil
}

// UpdateClaimColumns 局部更新责任工单字段。
func (r *Repository) UpdateClaimColumns(ctx context.Context, id uint, columns map[string]any) error {
	if len(columns) == 0 {
		return nil
	}
	result := r.session(ctx).Model(&WarrantyClaim{}).Where("id = ?", id).Updates(columns)
	if result.Error != nil {
		return fmt.Errorf("更新责任工单失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperr.NotFound("责任工单不存在: id=%d", id)
	}
	return nil
}

// DeleteClaimByFault 按故障 ID 删除责任工单。
func (r *Repository) DeleteClaimByFault(ctx context.Context, faultID uint) error {
	if err := r.session(ctx).Where("fault_id = ?", faultID).Delete(&WarrantyClaim{}).Error; err != nil {
		return fmt.Errorf("删除责任工单失败: %w", err)
	}
	return nil
}

// GetClaim 按主键查询责任工单。
func (r *Repository) GetClaim(ctx context.Context, id uint) (*WarrantyClaim, error) {
	var entity WarrantyClaim
	err := r.session(ctx).First(&entity, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("责任工单不存在: id=%d", id)
	}
	if err != nil {
		return nil, fmt.Errorf("查询责任工单失败: %w", err)
	}
	return &entity, nil
}

// GetClaimByFault 按故障查询责任工单, 不存在时返回 nil。
func (r *Repository) GetClaimByFault(ctx context.Context, faultID uint) (*WarrantyClaim, error) {
	var entity WarrantyClaim
	err := r.session(ctx).Where("fault_id = ?", faultID).First(&entity).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("查询故障责任工单失败: %w", err)
	}
	return &entity, nil
}

// GetClaimsByFaults 批量按故障查询责任工单。
func (r *Repository) GetClaimsByFaults(ctx context.Context, faultIDs []uint) ([]WarrantyClaim, error) {
	entities := make([]WarrantyClaim, 0)
	if len(faultIDs) == 0 {
		return entities, nil
	}
	if err := r.session(ctx).Where("fault_id IN ?", faultIDs).Find(&entities).Error; err != nil {
		return nil, fmt.Errorf("批量查询责任工单失败: %w", err)
	}
	return entities, nil
}

// claimFilter 是仓储层的责任工单查询条件。
type claimFilter struct {
	Keyword    string
	Status     string
	PartyType  string
	Component  string
	SupplierID uint
	InWarranty *bool
	OnlyOpen   bool
}

// ListClaims 分页查询责任工单。
func (r *Repository) ListClaims(ctx context.Context, filter claimFilter, page pagination.Query) ([]WarrantyClaim, int64, error) {
	base := func() *gorm.DB {
		statement := r.session(ctx).Model(&WarrantyClaim{})
		if value := strings.TrimSpace(filter.Keyword); value != "" {
			like := "%" + value + "%"
			statement = statement.Where(
				"fault_no LIKE ? OR lamp_code LIKE ? OR road_name LIKE ? OR supplier_name LIKE ?",
				like, like, like, like,
			)
		}
		if filter.Status != "" {
			statement = statement.Where("status = ?", filter.Status)
		}
		if filter.PartyType != "" {
			statement = statement.Where("party_type = ?", filter.PartyType)
		}
		if filter.Component != "" {
			statement = statement.Where("component = ?", filter.Component)
		}
		if filter.SupplierID > 0 {
			statement = statement.Where("supplier_id = ?", filter.SupplierID)
		}
		if filter.InWarranty != nil {
			statement = statement.Where("in_warranty = ?", *filter.InWarranty)
		}
		if filter.OnlyOpen {
			statement = statement.Where("status <> ?", ClaimStatusClosed)
		}
		return statement
	}

	var total int64
	if err := base().Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计责任工单数量失败: %w", err)
	}

	entities := make([]WarrantyClaim, 0)
	if err := base().Order(page.OrderClause()).Offset(page.Offset()).Limit(page.Limit()).Find(&entities).Error; err != nil {
		return nil, 0, fmt.Errorf("查询责任工单列表失败: %w", err)
	}
	return entities, total, nil
}

// CountClaims 统计责任工单总数。
func (r *Repository) CountClaims(ctx context.Context) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&WarrantyClaim{}).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计责任工单总数失败: %w", err)
	}
	return total, nil
}

// CountClaimsByColumn 按列分组统计责任工单。
func (r *Repository) CountClaimsByColumn(ctx context.Context, column string) (map[string]int64, error) {
	type row struct {
		Label string
		Total int64
	}
	rows := make([]row, 0)
	err := r.session(ctx).Model(&WarrantyClaim{}).
		Select(column + " AS label, COUNT(*) AS total").
		Group(column).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("分组统计责任工单 %s 失败: %w", column, err)
	}
	result := make(map[string]int64, len(rows))
	for _, item := range rows {
		result[item.Label] = item.Total
	}
	return result, nil
}

// CountInWarrantyClaims 统计质保期内判定由厂家承担的工单数量。
func (r *Repository) CountInWarrantyClaims(ctx context.Context) (int64, error) {
	return r.countClaims(ctx, "in_warranty = ?", true)
}

// CountClaimsBySupplier 按责任厂家分组统计工单数量(仅厂家责任工单), 按数量倒序。
func (r *Repository) CountClaimsBySupplier(ctx context.Context) ([]LabelCount, error) {
	type row struct {
		Label string
		Total int64
	}
	rows := make([]row, 0)
	err := r.session(ctx).Model(&WarrantyClaim{}).
		Select("supplier_name AS label, COUNT(*) AS total").
		Where("party_type = ? AND supplier_name <> ''", PartyManufacturer).
		Group("supplier_name").
		Order("total DESC, supplier_name ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("按供应商统计责任工单失败: %w", err)
	}
	result := make([]LabelCount, 0, len(rows))
	for _, item := range rows {
		result = append(result, LabelCount{Label: item.Label, Count: item.Total})
	}
	return result, nil
}

// CountTakenOverClaims 统计累计转由自有班组接手的工单数量。
func (r *Repository) CountTakenOverClaims(ctx context.Context) (int64, error) {
	return r.countClaims(ctx, "status = ?", ClaimStatusTakenOver)
}

// CountManufacturerOpen 统计厂家尚未闭环(仍由厂家负责)的工单数量, 已转自有班组的不计入。
func (r *Repository) CountManufacturerOpen(ctx context.Context) (int64, error) {
	return r.countClaims(ctx,
		"party_type = ? AND status IN ?",
		PartyManufacturer,
		[]string{ClaimStatusPending, ClaimStatusProcessing, ClaimStatusOverdue})
}

// CountResponseOverdue 统计当前厂家响应超时的工单数量。
// 各供应商承诺时限不同, 取回待响应工单后在应用层判定, 兼容 sqlite 与 postgres。
func (r *Repository) CountResponseOverdue(ctx context.Context, now time.Time) (int64, error) {
	entities, err := r.ListPendingResponse(ctx)
	if err != nil {
		return 0, err
	}
	var total int64
	for index := range entities {
		if isResponseOverdue(&entities[index], now) {
			total++
		}
	}
	return total, nil
}

// ListResponseOverdue 查询当前厂家响应超时的工单。
func (r *Repository) ListResponseOverdue(ctx context.Context, now time.Time, limit int) ([]WarrantyClaim, error) {
	if limit <= 0 {
		limit = 10
	}
	entities := make([]WarrantyClaim, 0)
	err := r.session(ctx).Model(&WarrantyClaim{}).
		Where("party_type = ? AND responded_at IS NULL AND status IN ?",
			PartyManufacturer, []string{ClaimStatusPending, ClaimStatusOverdue}).
		Order("assigned_at ASC, id ASC").
		Limit(limit).
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("查询厂家响应超时工单失败: %w", err)
	}
	// 时限逐单不同, 在应用层过滤, 兼容 sqlite 与 postgres。
	filtered := make([]WarrantyClaim, 0, len(entities))
	for _, item := range entities {
		if isResponseOverdue(&item, now) {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

// ListPendingResponse 查询全部待厂家响应(含已超时)的工单, 供定时巡检使用。
func (r *Repository) ListPendingResponse(ctx context.Context) ([]WarrantyClaim, error) {
	entities := make([]WarrantyClaim, 0)
	err := r.session(ctx).Model(&WarrantyClaim{}).
		Where("party_type = ? AND responded_at IS NULL AND status IN ?",
			PartyManufacturer, []string{ClaimStatusPending, ClaimStatusOverdue}).
		Order("assigned_at ASC, id ASC").
		Find(&entities).Error
	if err != nil {
		return nil, fmt.Errorf("查询待厂家响应工单失败: %w", err)
	}
	return entities, nil
}

// countClaims 按条件统计责任工单数量。
func (r *Repository) countClaims(ctx context.Context, condition string, args ...any) (int64, error) {
	var total int64
	if err := r.session(ctx).Model(&WarrantyClaim{}).Where(condition, args...).Count(&total).Error; err != nil {
		return 0, fmt.Errorf("统计责任工单失败: %w", err)
	}
	return total, nil
}

// isResponseOverdue 判断厂家工单是否已超过承诺响应时限。
func isResponseOverdue(entity *WarrantyClaim, now time.Time) bool {
	if entity.PartyType != PartyManufacturer || entity.RespondedAt != nil {
		return false
	}
	if entity.Status != ClaimStatusPending && entity.Status != ClaimStatusOverdue {
		return false
	}
	deadline := entity.AssignedAt.Add(time.Duration(entity.ResponseDeadlineHours) * time.Hour)
	return !now.Before(deadline)
}
