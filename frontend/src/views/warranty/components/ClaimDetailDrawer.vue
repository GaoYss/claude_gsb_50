<template>
  <el-drawer
    :model-value="modelValue"
    title="质保责任工单"
    size="620px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="load"
  >
    <div v-loading="loading">
      <template v-if="claim">
        <el-descriptions :column="2" border size="small" title="责任判定">
          <el-descriptions-item label="故障单号">{{ claim.fault_no }}</el-descriptions-item>
          <el-descriptions-item label="当前责任方">
            <StatusTag :dict="WARRANTY_PARTY" :value="claim.current_party || claim.party_type" />
          </el-descriptions-item>
          <el-descriptions-item label="故障部件">
            <StatusTag v-if="claim.component" :dict="WARRANTY_COMPONENT" :value="claim.component" />
            <span v-else class="text-muted">非质保部件</span>
          </el-descriptions-item>
          <el-descriptions-item label="是否质保内">
            <el-tag :type="claim.in_warranty ? 'warning' : 'success'" size="small">
              {{ claim.in_warranty ? '质保期内' : '超期 / 非质保' }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="工单状态" :span="2">
            <StatusTag :dict="CLAIM_STATUS" :value="claim.status" />
            <el-tag v-if="claim.overdue" type="danger" size="small" effect="dark" class="overdue-tag">已超承诺时限</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="路灯编号">{{ claim.lamp_code }}</el-descriptions-item>
          <el-descriptions-item label="所在道路">{{ claim.road_name }}</el-descriptions-item>
          <el-descriptions-item v-if="claim.warranty_end_at" label="质保截止日">
            {{ formatDate(claim.warranty_end_at) }}
          </el-descriptions-item>
          <el-descriptions-item label="责任判定时间">
            {{ formatDateTime(claim.assigned_at) }}
          </el-descriptions-item>
        </el-descriptions>

        <template v-if="claim.party_type === 'manufacturer'">
          <el-descriptions class="drawer-block" :column="2" border size="small" title="厂家与联系方式">
            <el-descriptions-item label="供应商">{{ claim.supplier_name || claim.supplier?.name || '-' }}</el-descriptions-item>
            <el-descriptions-item label="承诺响应时限">{{ claim.response_deadline_hours }} 小时</el-descriptions-item>
            <el-descriptions-item label="联系人">{{ claim.contact_person || claim.supplier?.contact_person || '-' }}</el-descriptions-item>
            <el-descriptions-item label="联系电话">
              <el-link v-if="claim.contact_phone" type="primary" :href="`tel:${claim.contact_phone}`">{{ claim.contact_phone }}</el-link>
              <span v-else>-</span>
            </el-descriptions-item>
            <el-descriptions-item label="服务热线" :span="2">
              <el-link v-if="claim.service_phone" type="primary" :href="`tel:${claim.service_phone}`">{{ claim.service_phone }}</el-link>
              <span v-else>-</span>
            </el-descriptions-item>
          </el-descriptions>

          <el-descriptions class="drawer-block" :column="2" border size="small" title="响应过程">
            <el-descriptions-item label="通知时间">{{ formatDateTime(claim.notified_at) }}</el-descriptions-item>
            <el-descriptions-item label="厂家响应时间">{{ formatDateTime(claim.responded_at) }}</el-descriptions-item>
            <el-descriptions-item label="已等待 / 响应耗时">
              {{ formatHours(claim.response_used_hours) }}
            </el-descriptions-item>
            <el-descriptions-item label="催办次数">{{ claim.remind_count }} 次</el-descriptions-item>
            <el-descriptions-item label="最近催办" :span="2">{{ formatDateTime(claim.last_reminded_at) }}</el-descriptions-item>
          </el-descriptions>
        </template>

        <el-descriptions v-if="claim.status === 'taken_over'" class="drawer-block" :column="2" border size="small" title="转办记录">
          <el-descriptions-item label="接手班组">{{ claim.takeover_team || '-' }}</el-descriptions-item>
          <el-descriptions-item label="经手人">{{ claim.takeover_by || '-' }}</el-descriptions-item>
          <el-descriptions-item label="转办时间" :span="2">{{ formatDateTime(claim.takeover_at) }}</el-descriptions-item>
          <el-descriptions-item label="转办原因" :span="2">{{ claim.takeover_reason || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="drawer-block actions">
          <template v-if="canOperate">
            <el-button type="warning" plain :icon="Bell" @click="handleRemind">催办厂家</el-button>
            <el-button type="primary" plain @click="handleResponded">登记厂家已到场</el-button>
            <el-button type="danger" plain :icon="Switch" @click="openTakeover">转自有班组接手</el-button>
          </template>
          <el-tag v-else-if="claim.status === 'taken_over'" type="info" effect="plain">
            已转由{{ claim.takeover_team || '自有班组' }}接手
          </el-tag>
          <el-tag v-else-if="claim.status === 'closed'" type="success" effect="plain">工单已闭环</el-tag>
          <el-tag v-else type="success" effect="plain">由自有班组处置中</el-tag>
        </div>
      </template>
      <el-empty v-else description="暂无责任工单数据" />
    </div>

    <TakeoverDialog v-model="takeoverVisible" :claim="claim" @saved="handleChanged" />
  </el-drawer>
</template>

<script setup>
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Bell, Switch } from '@element-plus/icons-vue'
import StatusTag from '@/components/common/StatusTag.vue'
import TakeoverDialog from './TakeoverDialog.vue'
import { warrantyApi } from '@/api/warranty'
import { CLAIM_STATUS, WARRANTY_COMPONENT, WARRANTY_PARTY } from '@/constants/dict'
import { formatDate, formatDateTime, formatHours } from '@/utils/format'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  claimId: { type: [Number, String], default: null },
})

const emit = defineEmits(['update:modelValue', 'changed'])

const loading = ref(false)
const claim = ref(null)
const takeoverVisible = ref(false)

const canOperate = computed(
  () =>
    claim.value?.party_type === 'manufacturer' &&
    ['pending', 'processing', 'overdue'].includes(claim.value?.status),
)

async function load() {
  if (!props.claimId) return
  loading.value = true
  try {
    claim.value = await warrantyApi.claimDetail(props.claimId)
  } catch (error) {
    claim.value = null
  } finally {
    loading.value = false
  }
}

async function handleRemind() {
  try {
    const { value } = await ElMessageBox.prompt('可填写本次催办备注 (将记录催办时间与次数)', `催办厂家 ${claim.value.supplier_name || ''}`, {
      confirmButtonText: '确认催办',
      cancelButtonText: '取消',
      inputPlaceholder: '催办备注, 选填',
    })
    await warrantyApi.remindClaim(claim.value.id, { remark: value ?? '' })
    ElMessage.success('已记录催办, 请通过服务热线联系厂家')
    handleChanged()
  } catch (error) {
    // 用户取消或请求失败
  }
}

async function handleResponded() {
  try {
    await ElMessageBox.confirm('确认厂家已到场响应? 工单将进入厂家处理中。', '登记厂家响应', {
      type: 'info',
      confirmButtonText: '确认',
      cancelButtonText: '取消',
    })
  } catch (error) {
    return
  }
  await warrantyApi.respondClaim(claim.value.id)
  ElMessage.success('已登记厂家响应')
  handleChanged()
}

function openTakeover() {
  takeoverVisible.value = true
}

function handleChanged() {
  load()
  emit('changed')
}
</script>

<style scoped>
.drawer-block {
  margin-top: 20px;
}

.overdue-tag {
  margin-left: 8px;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
</style>
