import request from './request'

// 质保与责任方接口。
export const warrantyApi = {
  // 质保登记
  list: (params) => request.get('/warranties', { params }),
  detail: (id) => request.get(`/warranties/${id}`),
  create: (data) => request.post('/warranties', data),
  update: (id, data) => request.put(`/warranties/${id}`, data),
  remove: (id) => request.delete(`/warranties/${id}`),
  // 供应商
  suppliers: (params) => request.get('/warranties/suppliers', { params }),
  supplierOptions: () => request.get('/warranties/suppliers/options'),
  createSupplier: (data) => request.post('/warranties/suppliers', data),
  updateSupplier: (id, data) => request.put(`/warranties/suppliers/${id}`, data),
  removeSupplier: (id) => request.delete(`/warranties/suppliers/${id}`),
  // 责任判定
  assignments: (params) => request.get('/warranties/assignments', { params }),
  assignmentByFault: (faultId, config) => request.get(`/warranties/assignments/fault/${faultId}`, config),
  transfer: (id, data) => request.post(`/warranties/assignments/${id}/transfer`, data),
  // 概览与字典
  overview: () => request.get('/warranties/overview'),
  meta: () => request.get('/warranties/meta'),
}
