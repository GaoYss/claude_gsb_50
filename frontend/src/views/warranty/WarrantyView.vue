<template>
  <div class="page">
    <PageHeader title="质保与责任方" description="登记灯具与灯杆质保, 故障登记时自动判定责任方, 厂家超时自动提醒并可转自有班组">
      <el-button :icon="Refresh" @click="reloadAll">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openWarrantyCreate">登记质保</el-button>
    </PageHeader>

    <div v-loading="overviewLoading" class="card-grid">
      <StatCard
        label="质保内维修占比"
        :value="overview.in_warranty_repair_ratio"
        suffix="%"
        icon="PieChart"
        color="#409eff"
        :hint="`质保内维修 ${overview.in_warranty_repair_total} / 共 ${overview.repair_total} 次`"
      />
      <StatCard
        label="厂家响应超时"
        :value="overview.overdue_total"
        suffix="单"
        icon="AlarmClock"
        color="#f56c6c"
        :hint="`超过 ${overview.response_timeout_hours} 小时未处理已自动提醒`"
      />
      <StatCard
        label="在保质保"
        :value="overview.warranty_by_state?.active ?? 0"
        suffix="条"
        icon="CircleCheck"
        color="#67c23a"
        :hint="`质保登记共 ${overview.warranty_total} 条`"
      />
      <StatCard
        label="30 天内到期"
        :value="overview.warranty_by_state?.expiring ?? 0"
        suffix="条"
        icon="Warning"
        color="#e6a23c"
        :hint="`已到期 ${overview.warranty_by_state?.expired ?? 0} 条`"
      />
      <StatCard
        label="质保期内指派厂家"
        :value="overview.supplier_assigned"
        suffix="单"
        icon="OfficeBuilding"
        color="#409eff"
        :hint="`超期/无质保转自有班组 ${overview.own_team_assigned} 单`"
      />
      <StatCard
        label="已转自有班组"
        :value="overview.transferred_total"
        suffix="单"
        icon="Switch"
        color="#909399"
        :hint="`供应商档案 ${overview.supplier_total} 家`"
      />
    </div>

    <el-card shadow="never">
      <el-tabs v-model="activeTab">
        <el-tab-pane label="责任方判定" name="assignments">
          <div class="filter-bar">
            <el-input v-model="assignmentPage.query.keyword" placeholder="故障单号 / 路灯编号 / 供应商" clearable @keyup.enter="assignmentPage.search" />
            <el-select v-model="assignmentPage.query.responsible_type" placeholder="责任方" clearable @change="assignmentPage.search">
              <el-option v-for="(item, key) in RESPONSIBLE_TYPE" :key="key" :label="item.label" :value="key" />
            </el-select>
            <el-select v-model="assignmentPage.query.status" placeholder="处理状态" clearable @change="assignmentPage.search">
              <el-option v-for="(item, key) in ASSIGNMENT_STATUS" :key="key" :label="item.label" :value="key" />
            </el-select>
            <el-select v-model="assignmentPage.query.component" placeholder="责任部件" clearable @change="assignmentPage.search">
              <el-option v-for="(item, key) in WARRANTY_COMPONENT" :key="key" :label="item.label" :value="key" />
            </el-select>
            <el-button type="primary" :icon="Search" @click="assignmentPage.search">查询</el-button>
            <el-button :icon="RefreshLeft" @click="assignmentPage.reset">重置</el-button>
          </div>

          <el-table v-loading="assignmentPage.loading.value" :data="assignmentPage.rows.value" stripe>
            <el-table-column prop="fault_no" label="故障单号" width="140" fixed="left" />
            <el-table-column prop="lamp_code" label="路灯编号" width="110" />
            <el-table-column label="责任部件" width="90">
              <template #default="{ row }"><StatusTag :dict="WARRANTY_COMPONENT" :value="row.component" /></template>
            </el-table-column>
            <el-table-column label="责任方" width="100">
              <template #default="{ row }"><StatusTag :dict="RESPONSIBLE_TYPE" :value="row.responsible_type" /></template>
            </el-table-column>
            <el-table-column prop="supplier_name" label="厂家" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">{{ row.supplier_name || '-' }}</template>
            </el-table-column>
            <el-table-column label="联系方式" width="130">
              <template #default="{ row }">{{ row.contact_phone || '-' }}</template>
            </el-table-column>
            <el-table-column label="处理状态" width="120">
              <template #default="{ row }"><StatusTag :dict="ASSIGNMENT_STATUS" :value="row.status" /></template>
            </el-table-column>
            <el-table-column label="响应时限" width="150">
              <template #default="{ row }">{{ formatDateTime(row.deadline) }}</template>
            </el-table-column>
            <el-table-column prop="reason" label="判定依据" min-width="220" show-overflow-tooltip />
            <el-table-column label="操作" width="150" fixed="right">
              <template #default="{ row }">
                <el-button
                  v-if="row.responsible_type === 'supplier'"
                  link
                  type="warning"
                  @click="handleTransfer(row)"
                >转自有班组</el-button>
                <span v-else class="text-muted">-</span>
              </template>
            </el-table-column>
          </el-table>
          <DataPagination
            :page="assignmentPage.query.page"
            :page-size="assignmentPage.query.page_size"
            :total="assignmentPage.total.value"
            @page-change="assignmentPage.changePage"
            @size-change="assignmentPage.changePageSize"
          />
        </el-tab-pane>

        <el-tab-pane label="质保登记" name="warranties">
          <div class="filter-bar">
            <el-input v-model="warrantyPage.query.keyword" placeholder="路灯编号 / 供应商名称" clearable @keyup.enter="warrantyPage.search" />
            <el-select v-model="warrantyPage.query.component" placeholder="部件" clearable @change="warrantyPage.search">
              <el-option v-for="(item, key) in WARRANTY_COMPONENT" :key="key" :label="item.label" :value="key" />
            </el-select>
            <el-select v-model="warrantyPage.query.state" placeholder="质保状态" clearable @change="warrantyPage.search">
              <el-option v-for="(item, key) in WARRANTY_STATE" :key="key" :label="item.label" :value="key" />
            </el-select>
            <el-button type="primary" :icon="Search" @click="warrantyPage.search">查询</el-button>
            <el-button :icon="RefreshLeft" @click="warrantyPage.reset">重置</el-button>
          </div>

          <el-table v-loading="warrantyPage.loading.value" :data="warrantyPage.rows.value" stripe>
            <el-table-column prop="lamp_code" label="路灯编号" width="120" fixed="left" />
            <el-table-column label="部件" width="90">
              <template #default="{ row }"><StatusTag :dict="WARRANTY_COMPONENT" :value="row.component" /></template>
            </el-table-column>
            <el-table-column prop="supplier_name" label="供应商" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">{{ row.supplier_name || '-' }}</template>
            </el-table-column>
            <el-table-column prop="contact_person" label="联系人" width="100">
              <template #default="{ row }">{{ row.contact_person || '-' }}</template>
            </el-table-column>
            <el-table-column prop="contact_phone" label="联系电话" width="130">
              <template #default="{ row }">{{ row.contact_phone || '-' }}</template>
            </el-table-column>
            <el-table-column label="质保开始" width="110">
              <template #default="{ row }">{{ formatDate(row.start_date) }}</template>
            </el-table-column>
            <el-table-column label="质保到期" width="110">
              <template #default="{ row }">{{ formatDate(row.end_date) }}</template>
            </el-table-column>
            <el-table-column label="质保状态" width="100">
              <template #default="{ row }"><StatusTag :dict="WARRANTY_STATE" :value="row.state" /></template>
            </el-table-column>
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openWarrantyEdit(row)">编辑</el-button>
                <el-button link type="danger" @click="handleWarrantyDelete(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <DataPagination
            :page="warrantyPage.query.page"
            :page-size="warrantyPage.query.page_size"
            :total="warrantyPage.total.value"
            @page-change="warrantyPage.changePage"
            @size-change="warrantyPage.changePageSize"
          />
        </el-tab-pane>

        <el-tab-pane label="供应商" name="suppliers">
          <div class="filter-bar">
            <el-input v-model="supplierPage.query.keyword" placeholder="名称 / 联系人 / 电话" clearable @keyup.enter="supplierPage.search" />
            <el-button type="primary" :icon="Search" @click="supplierPage.search">查询</el-button>
            <el-button :icon="RefreshLeft" @click="supplierPage.reset">重置</el-button>
            <el-button type="primary" plain :icon="Plus" @click="openSupplierCreate">新增供应商</el-button>
          </div>

          <el-table v-loading="supplierPage.loading.value" :data="supplierPage.rows.value" stripe>
            <el-table-column prop="name" label="供应商名称" min-width="180" show-overflow-tooltip />
            <el-table-column prop="contact_person" label="联系人" width="110">
              <template #default="{ row }">{{ row.contact_person || '-' }}</template>
            </el-table-column>
            <el-table-column prop="contact_phone" label="联系电话" width="140">
              <template #default="{ row }">{{ row.contact_phone || '-' }}</template>
            </el-table-column>
            <el-table-column prop="email" label="邮箱" min-width="170" show-overflow-tooltip>
              <template #default="{ row }">{{ row.email || '-' }}</template>
            </el-table-column>
            <el-table-column prop="address" label="地址" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">{{ row.address || '-' }}</template>
            </el-table-column>
            <el-table-column label="操作" width="140" fixed="right">
              <template #default="{ row }">
                <el-button link type="primary" @click="openSupplierEdit(row)">编辑</el-button>
                <el-button link type="danger" @click="handleSupplierDelete(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <DataPagination
            :page="supplierPage.query.page"
            :page-size="supplierPage.query.page_size"
            :total="supplierPage.total.value"
            @page-change="supplierPage.changePage"
            @size-change="supplierPage.changePageSize"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <WarrantyFormDialog
      v-model="warrantyFormVisible"
      :model="editingWarranty"
      :preset-lamp="presetLamp"
      :supplier-options="supplierOptions"
      @saved="handleWarrantySaved"
    />
    <SupplierFormDialog v-model="supplierFormVisible" :model="editingSupplier" @saved="handleSupplierSaved" />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatCard from '@/components/common/StatCard.vue'
import StatusTag from '@/components/common/StatusTag.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import WarrantyFormDialog from './components/WarrantyFormDialog.vue'
import SupplierFormDialog from './components/SupplierFormDialog.vue'
import { warrantyApi } from '@/api/warranty'
import { ASSIGNMENT_STATUS, RESPONSIBLE_TYPE, WARRANTY_COMPONENT, WARRANTY_STATE } from '@/constants/dict'
import { formatDate, formatDateTime } from '@/utils/format'
import { useListPage } from '@/composables/useListPage'

const route = useRoute()
const router = useRouter()

const activeTab = ref('assignments')

const assignmentPage = useListPage(warrantyApi.assignments, {
  keyword: '',
  responsible_type: '',
  status: '',
  component: '',
})
const warrantyPage = useListPage(warrantyApi.list, { keyword: '', component: '', state: '' })
const supplierPage = useListPage(warrantyApi.suppliers, { keyword: '' })

const emptyOverview = () => ({
  supplier_total: 0,
  warranty_total: 0,
  warranty_by_state: {},
  assignment_total: 0,
  supplier_assigned: 0,
  own_team_assigned: 0,
  overdue_total: 0,
  transferred_total: 0,
  repair_total: 0,
  in_warranty_repair_total: 0,
  in_warranty_repair_ratio: 0,
  response_timeout_hours: 24,
})

const overviewLoading = ref(false)
const overview = ref(emptyOverview())
const supplierOptions = ref([])

const warrantyFormVisible = ref(false)
const supplierFormVisible = ref(false)
const editingWarranty = ref(null)
const editingSupplier = ref(null)
const presetLamp = ref(null)

async function loadOverview() {
  overviewLoading.value = true
  try {
    overview.value = await warrantyApi.overview()
  } catch (error) {
    overview.value = emptyOverview()
  } finally {
    overviewLoading.value = false
  }
}

async function loadSupplierOptions() {
  try {
    supplierOptions.value = await warrantyApi.supplierOptions()
  } catch (error) {
    supplierOptions.value = []
  }
}

function reloadAll() {
  loadOverview()
  assignmentPage.load()
  warrantyPage.load()
  supplierPage.load()
}

function openWarrantyCreate() {
  editingWarranty.value = null
  presetLamp.value = null
  warrantyFormVisible.value = true
}

function openWarrantyEdit(row) {
  editingWarranty.value = { ...row }
  presetLamp.value = null
  warrantyFormVisible.value = true
}

async function handleWarrantyDelete(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除路灯 ${row.lamp_code} 的质保登记? 已生成的责任判定不受影响。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' },
    )
  } catch (error) {
    return
  }
  try {
    await warrantyApi.remove(row.id)
    ElMessage.success('质保登记已删除')
    handleWarrantySaved()
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  }
}

function handleWarrantySaved() {
  warrantyPage.load()
  loadOverview()
}

async function handleTransfer(row) {
  try {
    const { value } = await ElMessageBox.prompt(
      `确认将故障 ${row.fault_no} 转由自有班组接手? 厂家:${row.supplier_name || '-'}`,
      '转自有班组',
      {
        confirmButtonText: '确认转派',
        cancelButtonText: '取消',
        inputPlaceholder: '转派说明(选填), 例如: 厂家超时未响应, 一班接手',
      },
    )
    await warrantyApi.transfer(row.id, { remark: value ?? '' })
    ElMessage.success('已转自有班组接手')
    assignmentPage.load()
    loadOverview()
  } catch (error) {
    // 用户取消或请求失败, 提示由拦截器处理
  }
}

function openSupplierCreate() {
  editingSupplier.value = null
  supplierFormVisible.value = true
}

function openSupplierEdit(row) {
  editingSupplier.value = { ...row }
  supplierFormVisible.value = true
}

async function handleSupplierDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除供应商 ${row.name} ? 被质保记录引用时无法删除。`, '删除确认', {
      type: 'warning',
      confirmButtonText: '确认删除',
      cancelButtonText: '取消',
    })
  } catch (error) {
    return
  }
  try {
    await warrantyApi.removeSupplier(row.id)
    ElMessage.success('供应商已删除')
    handleSupplierSaved()
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  }
}

function handleSupplierSaved() {
  supplierPage.load()
  loadSupplierOptions()
  loadOverview()
}

// 支持从路灯台账页携带 lamp_id 直接登记质保。
async function applyRouteQuery() {
  const lampId = Number(route.query.lamp_id)
  if (!lampId) return
  presetLamp.value = { id: lampId, code: route.query.lamp_code ?? '' }
  editingWarranty.value = null
  activeTab.value = 'warranties'
  warrantyFormVisible.value = true
  router.replace({ path: '/warranty' })
}

onMounted(async () => {
  loadOverview()
  loadSupplierOptions()
  await applyRouteQuery()
})
</script>
