import request from './request'

// 质保与责任方管理接口。
export const warrantyApi = {
  // 供应商档案。
  listSuppliers: (params) => request.get('/warranty/suppliers', { params }),
  supplierOptions: () => request.get('/warranty/suppliers/options'),
  supplierDetail: (id) => request.get(`/warranty/suppliers/${id}`),
  createSupplier: (data) => request.post('/warranty/suppliers', data),
  updateSupplier: (id, data) => request.put(`/warranty/suppliers/${id}`, data),
  removeSupplier: (id) => request.delete(`/warranty/suppliers/${id}`),

  // 灯具/灯杆质保登记。
  listWarranties: (params) => request.get('/warranty/registrations', { params }),
  warrantyDetail: (id) => request.get(`/warranty/registrations/${id}`),
  createWarranty: (data) => request.post('/warranty/registrations', data),
  updateWarranty: (id, data) => request.put(`/warranty/registrations/${id}`, data),
  removeWarranty: (id) => request.delete(`/warranty/registrations/${id}`),

  // 故障责任工单。
  listClaims: (params) => request.get('/warranty/claims', { params }),
  claimDetail: (id) => request.get(`/warranty/claims/${id}`),
  claimByFault: (faultId, config) => request.get(`/warranty/claims/by-fault/${faultId}`, config),
  claimsIndex: (faultIds) =>
    request.get('/warranty/claims/index', { params: { fault_ids: faultIds } }),
  remindClaim: (id, data) => request.post(`/warranty/claims/${id}/remind`, data || {}),
  respondClaim: (id) => request.post(`/warranty/claims/${id}/respond`),
  takeoverClaim: (id, data) => request.post(`/warranty/claims/${id}/takeover`, data),
  claimMeta: () => request.get('/warranty/claims/meta'),
  claimOverview: () => request.get('/warranty/claims/overview'),
}
