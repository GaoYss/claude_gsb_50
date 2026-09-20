package warranty

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"streetlight/internal/apperr"
	"streetlight/internal/httpx"
	"streetlight/internal/response"
)

// Handler 处理质保与责任方管理相关的 HTTP 请求。
type Handler struct {
	service *Service
}

// NewHandler 构造质保管理处理器。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ---------------------------------------------------------------------------
// 供应商
// ---------------------------------------------------------------------------

// ListSuppliers 分页查询供应商。
func (h *Handler) ListSuppliers(c *gin.Context) {
	var query SupplierListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	items, total, page, err := h.service.ListSuppliers(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, page.Page, page.PageSize))
}

// SupplierOptions 供应商下拉选项。
func (h *Handler) SupplierOptions(c *gin.Context) {
	options, err := h.service.SupplierOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, options)
}

// GetSupplier 供应商详情。
func (h *Handler) GetSupplier(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.GetSupplier(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// CreateSupplier 新增供应商。
func (h *Handler) CreateSupplier(c *gin.Context) {
	var req SupplierUpsertRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.CreateSupplier(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, entity)
}

// UpdateSupplier 修改供应商。
func (h *Handler) UpdateSupplier(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req SupplierUpsertRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.UpdateSupplier(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// DeleteSupplier 删除供应商。
func (h *Handler) DeleteSupplier(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.DeleteSupplier(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// ---------------------------------------------------------------------------
// 质保登记
// ---------------------------------------------------------------------------

// ListWarranties 分页查询质保登记。
func (h *Handler) ListWarranties(c *gin.Context) {
	var query WarrantyListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	items, total, page, err := h.service.ListWarranties(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, page.Page, page.PageSize))
}

// GetWarranty 质保登记详情。
func (h *Handler) GetWarranty(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.GetWarranty(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// CreateWarranty 登记灯具/灯杆质保。
func (h *Handler) CreateWarranty(c *gin.Context) {
	var req WarrantyUpsertRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.UpsertWarranty(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, entity)
}

// UpdateWarranty 修改质保登记, 以路径 ID 为准, 忽略请求体中的 lamp_id。
func (h *Handler) UpdateWarranty(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req WarrantyUpsertRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	existing, err := h.service.GetWarranty(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	req.LampID = existing.LampID
	entity, err := h.service.UpsertWarranty(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// DeleteWarranty 删除质保登记。
func (h *Handler) DeleteWarranty(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	if err := h.service.DeleteWarranty(c.Request.Context(), id); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// ---------------------------------------------------------------------------
// 责任工单
// ---------------------------------------------------------------------------

// ListClaims 分页查询责任工单。
func (h *Handler) ListClaims(c *gin.Context) {
	var query ClaimListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	items, total, page, err := h.service.ListClaims(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, page.Page, page.PageSize))
}

// GetClaim 责任工单详情。
func (h *Handler) GetClaim(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.GetClaim(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// ClaimByFault 按故障查询责任工单。
func (h *Handler) ClaimByFault(c *gin.Context) {
	faultID, err := httpx.ParseID(c, "faultId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.GetClaimByFault(c.Request.Context(), faultID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// ClaimIndex 按故障 ID 批量返回责任工单, 参数 fault_ids 为逗号分隔的 ID。
func (h *Handler) ClaimIndex(c *gin.Context) {
	raw := strings.TrimSpace(c.Query("fault_ids"))
	ids := make([]uint, 0)
	if raw != "" {
		for _, part := range strings.Split(raw, ",") {
			value, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64)
			if err != nil || value == 0 {
				response.Fail(c, apperr.BadRequest("fault_ids 包含非法 ID: %q", part))
				return
			}
			ids = append(ids, uint(value))
		}
	}
	result, err := h.service.ClaimsByFaults(c.Request.Context(), ids)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Remind 人工催办厂家。
func (h *Handler) Remind(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req RemindRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Remind(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// MarkResponded 登记厂家已响应(到场)。
func (h *Handler) MarkResponded(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.MarkResponded(c.Request.Context(), id)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Takeover 将厂家工单转由自有班组接手。
func (h *Handler) Takeover(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req TakeoverRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.Takeover(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// Overview 质保与责任方概览。
func (h *Handler) Overview(c *gin.Context) {
	result, err := h.service.Overview(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, result)
}

// Metadata 质保模块字典。
func (h *Handler) Metadata(c *gin.Context) {
	response.OK(c, h.service.Metadata())
}
