import { ref } from 'vue'
import { warrantyApi } from '@/api/warranty'

// 质保模块字典(部件 / 责任方 / 工单状态 / 默认响应时限), 模块级缓存避免重复请求。
export const warrantyMeta = ref({
  components: [],
  parties: [],
  claim_statuses: [],
  default_response_deadline_hours: 24,
})

let loaded = false
let pending = null

export async function loadWarrantyMeta(force = false) {
  if (loaded && !force) return warrantyMeta.value
  if (pending) return pending
  pending = warrantyApi
    .claimMeta()
    .then((data) => {
      warrantyMeta.value = {
        components: data.components ?? [],
        parties: data.parties ?? [],
        claim_statuses: data.claim_statuses ?? [],
        default_response_deadline_hours: data.default_response_deadline_hours ?? 24,
      }
      loaded = true
      return warrantyMeta.value
    })
    .finally(() => {
      pending = null
    })
  return pending
}
