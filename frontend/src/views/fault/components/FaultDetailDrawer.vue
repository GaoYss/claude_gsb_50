<template>
  <el-drawer
    :model-value="modelValue"
    title="故障处理详情"
    size="640px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="load"
  >
    <div v-loading="loading">
      <template v-if="detail.fault">
        <el-descriptions :column="2" border size="small" title="故障信息">
          <el-descriptions-item label="故障单号">{{ detail.fault.fault_no }}</el-descriptions-item>
          <el-descriptions-item label="处理状态">
            <StatusTag :dict="FAULT_STATUS" :value="detail.fault.status" />
          </el-descriptions-item>
          <el-descriptions-item label="故障类型">{{ detail.fault.fault_type }}</el-descriptions-item>
          <el-descriptions-item label="紧急程度">
            <StatusTag :dict="FAULT_LEVEL" :value="detail.fault.fault_level" />
          </el-descriptions-item>
          <el-descriptions-item label="故障来源">{{ dictLabel(FAULT_SOURCE, detail.fault.source) }}</el-descriptions-item>
          <el-descriptions-item label="上报人">{{ detail.fault.reporter || '-' }}</el-descriptions-item>
          <el-descriptions-item label="上报时间">{{ formatDateTime(detail.fault.reported_at) }}</el-descriptions-item>
          <el-descriptions-item label="维修次数">{{ detail.fault.repair_count }} 次</el-descriptions-item>
          <el-descriptions-item label="故障描述" :span="2">{{ detail.fault.description || '-' }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.fault.closed_at" label="关闭时间" :span="1">
            {{ formatDateTime(detail.fault.closed_at) }}
          </el-descriptions-item>
          <el-descriptions-item v-if="detail.fault.closed_at" label="关闭说明" :span="1">
            {{ detail.fault.close_remark || '-' }}
          </el-descriptions-item>
        </el-descriptions>

        <el-descriptions v-if="claim" class="drawer-block" :column="2" border size="small" title="质保与责任方">
          <el-descriptions-item label="责任方">
            <StatusTag :dict="WARRANTY_PARTY" :value="claim.current_party || claim.party_type" />
            <el-tag v-if="claim.overdue" type="danger" size="small" effect="dark" class="overdue-tag">响应超时</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="工单状态">
            <StatusTag :dict="CLAIM_STATUS" :value="claim.status" />
          </el-descriptions-item>
          <el-descriptions-item label="故障部件">
            <StatusTag v-if="claim.component" :dict="WARRANTY_COMPONENT" :value="claim.component" />
            <span v-else class="text-muted">非质保部件</span>
          </el-descriptions-item>
          <el-descriptions-item label="质保范围">
            <el-tag :type="claim.in_warranty ? 'warning' : 'success'" size="small">
              {{ claim.in_warranty ? '质保期内' : '超期 / 非质保' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="供应商 / 班组" :span="2">
            {{ claim.supplier_name || '自有班组' }}
            <span v-if="claim.contact_phone" class="text-muted"> · {{ claim.contact_person || '-' }} {{ claim.contact_phone }}</span>
          </el-descriptions-item>
          <el-descriptions-item v-if="claim.party_type === 'manufacturer'" label="响应情况" :span="2">
            <span class="text-muted">承诺 {{ claim.response_deadline_hours }} 小时内响应; </span>
            <span v-if="claim.responded_at">{{ formatHours(claim.response_used_hours) }}到场; 已催办 {{ claim.remind_count }} 次</span>
            <span v-else>已等待 {{ formatHours(claim.response_used_hours) }}; 已催办 {{ claim.remind_count }} 次</span>
          </el-descriptions-item>
          <el-descriptions-item v-if="claim.status === 'taken_over'" label="接手信息" :span="2">
            {{ claim.takeover_team || '自有班组' }} · {{ claim.takeover_by || '-' }} · {{ claim.takeover_reason || '厂家超时转办' }}
          </el-descriptions-item>
          <el-descriptions-item label="责任处置" :span="2">
            <el-button size="small" type="primary" plain @click="openClaim">查看责任工单 / 催办 / 转办</el-button>
          </el-descriptions-item>
        </el-descriptions>

        <el-descriptions v-if="detail.lamp" class="drawer-block" :column="2" border size="small" title="路灯信息">
          <el-descriptions-item label="路灯编号">{{ detail.lamp.code }}</el-descriptions-item>
          <el-descriptions-item label="运行状态">
            <StatusTag :dict="RUN_STATUS" :value="detail.lamp.run_status" />
          </el-descriptions-item>
          <el-descriptions-item label="所在道路">{{ detail.lamp.road_name }}</el-descriptions-item>
          <el-descriptions-item label="区域">{{ detail.lamp.district || '-' }}</el-descriptions-item>
          <el-descriptions-item label="灯具类型">{{ detail.lamp.lamp_type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="功率">{{ detail.lamp.power ? `${detail.lamp.power} W` : '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="section-title drawer-block">处理时间线</div>
        <el-timeline v-if="detail.timeline?.length">
          <el-timeline-item
            v-for="(event, index) in detail.timeline"
            :key="index"
            :timestamp="formatDateTime(event.timestamp)"
            :type="dictType(TIMELINE_STAGE, event.stage)"
          >
            <div class="timeline-title">{{ event.label }}</div>
            <div class="text-muted timeline-detail">{{ event.operator || '系统' }} · {{ event.detail || '-' }}</div>
          </el-timeline-item>
        </el-timeline>
        <el-empty v-else description="暂无处理记录" :image-size="60" />

        <div class="section-title drawer-block">维修记录</div>
        <el-table :data="detail.repairs" size="small" border>
          <el-table-column prop="repair_no" label="维修单号" width="140" />
          <el-table-column prop="repairman" label="维修人员" width="100" />
          <el-table-column label="状态" width="90">
            <template #default="{ row }"><StatusTag :dict="REPAIR_STATUS" :value="row.status" /></template>
          </el-table-column>
          <el-table-column label="结果" width="90">
            <template #default="{ row }">{{ dictLabel(REPAIR_RESULT, row.result) }}</template>
          </el-table-column>
          <el-table-column label="开工时间" width="140">
            <template #default="{ row }">{{ formatDateTime(row.started_at) }}</template>
          </el-table-column>
          <el-table-column label="完工时间" width="140">
            <template #default="{ row }">{{ formatDateTime(row.finished_at) }}</template>
          </el-table-column>
          <el-table-column prop="content" label="维修内容" min-width="160" show-overflow-tooltip />
        </el-table>
      </template>
      <el-empty v-else description="暂无故障数据" />
    </div>
  </el-drawer>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import StatusTag from '@/components/common/StatusTag.vue'
import { statusApi } from '@/api/status'
import { warrantyApi } from '@/api/warranty'
import {
  CLAIM_STATUS,
  FAULT_LEVEL,
  FAULT_SOURCE,
  FAULT_STATUS,
  REPAIR_RESULT,
  REPAIR_STATUS,
  RUN_STATUS,
  TIMELINE_STAGE,
  WARRANTY_COMPONENT,
  WARRANTY_PARTY,
  dictLabel,
  dictType,
} from '@/constants/dict'
import { formatDateTime, formatHours } from '@/utils/format'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  faultId: { type: [Number, String], default: null },
})

const emit = defineEmits(['update:modelValue'])

const router = useRouter()
const loading = ref(false)
const detail = ref({ fault: null, lamp: null, repairs: [], timeline: [] })
const claim = ref(null)

// 打开抽屉时按故障 ID 拉取完整处理链路与质保责任信息。
async function load() {
  if (!props.faultId) return
  loading.value = true
  try {
    const [track, claimData] = await Promise.all([
      statusApi.track({ fault_id: props.faultId }),
      warrantyApi.claimByFault(props.faultId, { silent: true }).catch(() => null),
    ])
    detail.value = track
    claim.value = claimData
  } catch (error) {
    detail.value = { fault: null, lamp: null, repairs: [], timeline: [] }
    claim.value = null
  } finally {
    loading.value = false
  }
}

// 跳转责任工单页并直接打开该工单详情, 支持催办与转办。
function openClaim() {
  emit('update:modelValue', false)
  router.push({ path: '/warranties/claims', query: { claim_id: claim.value.id } })
}
</script>

<style scoped>
.drawer-block {
  margin-top: 20px;
}

.overdue-tag {
  margin-left: 8px;
}

.timeline-title {
  font-weight: 600;
}

.timeline-detail {
  font-size: 13px;
}
</style>
