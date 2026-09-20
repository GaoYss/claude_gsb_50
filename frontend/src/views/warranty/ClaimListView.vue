<template>
  <div class="page">
    <PageHeader title="故障责任工单" description="故障登记时自动判定责任方: 质保期内指派厂家, 超期转自有班组; 厂家超时可催办或转办">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
    </PageHeader>

    <div class="card-grid stat-grid">
      <StatCard label="责任工单总数" :value="overview.claim_total" suffix="条" icon="Tickets" color="#409eff" />
      <StatCard
        label="质保内维修占比"
        :value="ratePercent"
        suffix="%"
        icon="PieChart"
        color="#e6a23c"
        :hint="`厂家 ${overview.in_warranty_total} 条 / 自有班组 ${overview.out_warranty_total} 条`"
      />
      <StatCard
        label="厂家响应超时"
        :value="overview.response_overdue"
        suffix="条"
        icon="AlarmClock"
        color="#f56c6c"
        :hint="`厂家未闭环 ${overview.manufacturer_open} 条`"
      />
      <StatCard
        label="转自有班组接手"
        :value="overview.taken_over_total"
        suffix="条"
        icon="Switch"
        color="#909399"
        hint="厂家超时后应急接管"
      />
    </div>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="故障单号 / 路灯编号 / 道路 / 供应商" clearable @keyup.enter="handleSearch" />
        <el-select v-model="query.status" placeholder="工单状态" clearable @change="handleSearch">
          <el-option v-for="(item, key) in CLAIM_STATUS" :key="key" :label="item.label" :value="key" />
        </el-select>
        <el-select v-model="query.party_type" placeholder="责任方" clearable @change="handleSearch">
          <el-option v-for="(item, key) in WARRANTY_PARTY" :key="key" :label="item.label" :value="key" />
        </el-select>
        <el-select v-model="query.component" placeholder="故障部件" clearable @change="handleSearch">
          <el-option label="灯具" value="lamp" />
          <el-option label="灯杆" value="pole" />
        </el-select>
        <el-select v-model="inWarrantyFilter" placeholder="质保范围" clearable @change="handleSearch">
          <el-option label="质保期内(厂家)" :value="true" />
          <el-option label="超期 / 非质保" :value="false" />
        </el-select>
        <el-checkbox v-model="query.only_open" @change="handleSearch">仅看未闭环</el-checkbox>
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="fault_no" label="故障单号" width="140" fixed="left" />
        <el-table-column prop="lamp_code" label="路灯编号" width="105" />
        <el-table-column prop="road_name" label="所在道路" min-width="105" show-overflow-tooltip />
        <el-table-column label="部件" width="80" align="center">
          <template #default="{ row }">
            <StatusTag v-if="row.component" :dict="WARRANTY_COMPONENT" :value="row.component" />
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="责任方" width="100">
          <template #default="{ row }">
            <StatusTag :dict="WARRANTY_PARTY" :value="row.current_party || row.party_type" />
          </template>
        </el-table-column>
        <el-table-column label="质保" width="90" align="center">
          <template #default="{ row }">
            <el-tag :type="row.in_warranty ? 'warning' : 'info'" size="small" effect="plain">
              {{ row.in_warranty ? '质保内' : '超期' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="supplier_name" label="供应商 / 班组" min-width="170" show-overflow-tooltip>
          <template #default="{ row }">{{ row.supplier_name || '自有班组' }}</template>
        </el-table-column>
        <el-table-column label="工单状态" width="120">
          <template #default="{ row }">
            <StatusTag :dict="CLAIM_STATUS" :value="row.status" />
          </template>
        </el-table-column>
        <el-table-column label="响应情况" width="150">
          <template #default="{ row }">
            <el-tag v-if="row.overdue" type="danger" size="small" effect="dark">超时 {{ formatHours(row.response_used_hours) }}</el-tag>
            <el-tag v-else-if="row.responded_at" type="success" size="small" effect="plain">
              {{ formatHours(row.response_used_hours) }}到场
            </el-tag>
            <el-tag v-else-if="row.party_type === 'manufacturer'" type="warning" size="small" effect="plain">
              已等 {{ formatHours(row.response_used_hours) }}
            </el-tag>
            <span v-else class="text-muted">-</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="230" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openDetail(row)">详情</el-button>
            <template v-if="row.party_type === 'manufacturer' && ['pending', 'overdue', 'processing'].includes(row.status)">
              <el-button link type="warning" @click="handleQuickRemind(row)">催办</el-button>
              <el-button link type="danger" @click="openTakeover(row)">转接手</el-button>
            </template>
          </template>
        </el-table-column>
      </el-table>
      <DataPagination
        :page="query.page"
        :page-size="query.page_size"
        :total="total"
        @page-change="changePage"
        @size-change="changePageSize"
      />
    </el-card>

    <ClaimDetailDrawer v-model="detailVisible" :claim-id="activeClaimId" @changed="refreshAll" />
    <TakeoverDialog v-model="takeoverVisible" :claim="takeoverTarget" @saved="refreshAll" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatCard from '@/components/common/StatCard.vue'
import StatusTag from '@/components/common/StatusTag.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import ClaimDetailDrawer from './components/ClaimDetailDrawer.vue'
import TakeoverDialog from './components/TakeoverDialog.vue'
import { warrantyApi } from '@/api/warranty'
import { CLAIM_STATUS, WARRANTY_COMPONENT, WARRANTY_PARTY } from '@/constants/dict'
import { formatHours } from '@/utils/format'
import { useListPage } from '@/composables/useListPage'

const { loading, rows, total, query, load, search, reset, changePage, changePageSize } = useListPage(
  (params) => warrantyApi.listClaims(normalizeParams(params)),
  {
    keyword: '',
    status: '',
    party_type: '',
    component: '',
    in_warranty: '',
    only_open: false,
  },
)

const route = useRoute()
const router = useRouter()
const inWarrantyFilter = ref('')
const overview = ref(emptyOverview())
const detailVisible = ref(false)
const activeClaimId = ref(null)
const takeoverVisible = ref(false)
const takeoverTarget = ref(null)

const ratePercent = computed(() => {
  if (!overview.value.claim_total) return 0
  return (overview.value.in_warranty_rate * 100).toFixed(1)
})

function emptyOverview() {
  return {
    claim_total: 0,
    in_warranty_total: 0,
    out_warranty_total: 0,
    in_warranty_rate: 0,
    manufacturer_open: 0,
    response_overdue: 0,
    taken_over_total: 0,
  }
}

// in_warranty 是布尔查询参数, 空值不下发。
function normalizeParams(params) {
  const result = { ...params }
  if (inWarrantyFilter.value === '' || inWarrantyFilter.value === null) {
    delete result.in_warranty
  } else {
    result.in_warranty = inWarrantyFilter.value
  }
  return result
}

async function loadOverview() {
  try {
    overview.value = await warrantyApi.claimOverview()
  } catch (error) {
    overview.value = emptyOverview()
  }
}

function refreshAll() {
  load()
  loadOverview()
}

function handleSearch() {
  search()
}

function handleReset() {
  inWarrantyFilter.value = ''
  reset()
}

function openDetail(row) {
  activeClaimId.value = row.id
  detailVisible.value = true
}

async function handleQuickRemind(row) {
  try {
    await ElMessageBox.confirm(`确认向厂家「${row.supplier_name}」发送催办提醒?`, '催办确认', {
      type: 'warning',
      confirmButtonText: '确认催办',
      cancelButtonText: '取消',
    })
  } catch (error) {
    return
  }
  try {
    await warrantyApi.remindClaim(row.id, {})
    ElMessage.success('已记录催办')
    refreshAll()
  } catch (error) {
    // 错误提示由拦截器处理
  }
}

function openTakeover(row) {
  takeoverTarget.value = { ...row }
  takeoverVisible.value = true
}

// 支持从故障详情携带 claim_id 直接打开责任工单。
async function applyRouteQuery() {
  const claimId = Number(route.query.claim_id)
  if (!claimId) return
  activeClaimId.value = claimId
  detailVisible.value = true
  router.replace({ path: '/warranties/claims' })
}

onMounted(() => {
  loadOverview()
  applyRouteQuery()
})
</script>

<style scoped>
.stat-grid {
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
}
</style>
