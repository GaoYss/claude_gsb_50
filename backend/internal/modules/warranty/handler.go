package warranty

import (
	"github.com/gin-gonic/gin"

	"streetlight/internal/httpx"
	"streetlight/internal/response"
)

// Handler 处理质保与责任方相关的 HTTP 请求。
type Handler struct {
	service *Service
}

// NewHandler 构造质保与责任方处理器。
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// ---------- 供应商 ----------

// ListSuppliers 查询供应商列表。
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
	items, err := h.service.SupplierOptions(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, items)
}

// CreateSupplier 新增供应商。
func (h *Handler) CreateSupplier(c *gin.Context) {
	var req SupplierCreateRequest
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

// UpdateSupplier 更新供应商。
func (h *Handler) UpdateSupplier(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req SupplierUpdateRequest
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

// ---------- 质保登记 ----------

// ListWarranties 查询质保登记列表。
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

// GetWarranty 查询质保详情。
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

// CreateWarranty 登记质保。
func (h *Handler) CreateWarranty(c *gin.Context) {
	var req WarrantyCreateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.CreateWarranty(c.Request.Context(), req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, entity)
}

// UpdateWarranty 更新质保登记。
func (h *Handler) UpdateWarranty(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req WarrantyUpdateRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.UpdateWarranty(c.Request.Context(), id, req)
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

// ---------- 责任判定单 ----------

// ListAssignments 查询责任判定单列表。
func (h *Handler) ListAssignments(c *gin.Context) {
	var query AssignmentListQuery
	if err := httpx.BindQuery(c, &query); err != nil {
		response.Fail(c, err)
		return
	}
	items, total, page, err := h.service.ListAssignments(c.Request.Context(), query)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, response.NewPageData(items, total, page.Page, page.PageSize))
}

// GetAssignmentByFault 查询指定故障的责任判定单。
func (h *Handler) GetAssignmentByFault(c *gin.Context) {
	faultID, err := httpx.ParseID(c, "faultId")
	if err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.GetAssignmentByFault(c.Request.Context(), faultID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// TransferAssignment 厂家超时后转自有班组接手。
func (h *Handler) TransferAssignment(c *gin.Context) {
	id, err := httpx.ParseID(c, "id")
	if err != nil {
		response.Fail(c, err)
		return
	}
	var req TransferRequest
	if err := httpx.BindJSON(c, &req); err != nil {
		response.Fail(c, err)
		return
	}
	entity, err := h.service.TransferAssignment(c.Request.Context(), id, req)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, entity)
}

// ---------- 概览与字典 ----------

// Overview 质保与责任方概览。
func (h *Handler) Overview(c *gin.Context) {
	overview, err := h.service.Overview(c.Request.Context())
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, overview)
}

// Metadata 返回质保模块字典。
func (h *Handler) Metadata(c *gin.Context) {
	response.OK(c, h.service.Metadata())
}
