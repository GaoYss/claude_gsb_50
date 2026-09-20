<template>
  <div class="page">
    <PageHeader title="灯具 / 灯杆质保登记" description="为每盏路灯登记灯具与灯杆的供应商、质保起止日期及联系方式">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">登记质保</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="路灯编号 / 道路 / 供应商" clearable @keyup.enter="handleSearch" />
        <el-select v-model="query.supplier_id" placeholder="供应商" clearable filterable @change="handleSearch">
          <el-option v-for="item in supplierOptions" :key="item.id" :label="item.name" :value="item.id" />
        </el-select>
        <el-select v-model="query.component" placeholder="部件" clearable @change="handleSearch">
          <el-option label="灯具" value="lamp" />
          <el-option label="灯杆" value="pole" />
        </el-select>
        <el-select v-model="query.warranty_state" placeholder="质保状态" clearable @change="handleSearch">
          <el-option label="质保中" value="active" />
          <el-option label="30 天内到期" value="expiring" />
          <el-option label="已超期" value="expired" />
        </el-select>
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="lamp_code" label="路灯编号" width="110" fixed="left" />
        <el-table-column prop="lamp_name" label="路灯名称" min-width="140" show-overflow-tooltip />
        <el-table-column prop="road_name" label="所在道路" min-width="110" />
        <el-table-column label="灯具供应商 / 质保期" min-width="240">
          <template #default="{ row }">
            <template v-if="row.lamp_supplier">
              <div class="cell-line">
                <StatusTag :dict="WARRANTY_STATE" :value="row.state.lamp" />
                <span>{{ row.lamp_supplier.name }}</span>
              </div>
              <div class="text-muted cell-sub">
                {{ formatDate(row.lamp_start_at) }} 至 {{ formatDate(row.lamp_end_at) }}
                <el-tag v-if="row.state.lamp === 'active'" size="small" :type="row.lamp_remaining_days <= 30 ? 'warning' : 'info'" effect="plain">
                  剩余 {{ row.lamp_remaining_days }} 天
                </el-tag>
              </div>
            </template>
            <span v-else class="text-muted">未登记</span>
          </template>
        </el-table-column>
        <el-table-column label="灯杆供应商 / 质保期" min-width="240">
          <template #default="{ row }">
            <template v-if="row.pole_supplier">
              <div class="cell-line">
                <StatusTag :dict="WARRANTY_STATE" :value="row.state.pole" />
                <span>{{ row.pole_supplier.name }}</span>
              </div>
              <div class="text-muted cell-sub">
                {{ formatDate(row.pole_start_at) }} 至 {{ formatDate(row.pole_end_at) }}
                <el-tag v-if="row.state.pole === 'active'" size="small" :type="row.pole_remaining_days <= 30 ? 'warning' : 'info'" effect="plain">
                  剩余 {{ row.pole_remaining_days }} 天
                </el-tag>
              </div>
            </template>
            <span v-else class="text-muted">未登记</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
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

    <WarrantyFormDialog
      v-model="formVisible"
      :model="editing"
      :preset-lamp="presetLamp"
      :supplier-options="supplierOptions"
      @saved="handleSaved"
    />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import StatusTag from '@/components/common/StatusTag.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import WarrantyFormDialog from './components/WarrantyFormDialog.vue'
import { lampApi } from '@/api/lamp'
import { warrantyApi } from '@/api/warranty'
import { WARRANTY_STATE } from '@/constants/dict'
import { formatDate } from '@/utils/format'
import { useListPage } from '@/composables/useListPage'

const route = useRoute()
const router = useRouter()

const { loading, rows, total, query, load, search, reset, changePage, changePageSize } = useListPage(
  warrantyApi.listWarranties,
  { keyword: '', supplier_id: '', component: '', warranty_state: '' },
)

const supplierOptions = ref([])
const formVisible = ref(false)
const editing = ref(null)
const presetLamp = ref(null)

async function loadSuppliers() {
  try {
    supplierOptions.value = await warrantyApi.supplierOptions()
  } catch (error) {
    supplierOptions.value = []
  }
}

function handleSearch() {
  search()
}

function handleReset() {
  reset()
}

function openCreate() {
  editing.value = null
  presetLamp.value = null
  formVisible.value = true
}

function openEdit(row) {
  editing.value = { ...row }
  presetLamp.value = null
  formVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(
      `确认删除路灯 ${row.lamp_code} 的质保登记? 存在厂家处理中故障时将被拦截。`,
      '删除确认',
      { type: 'warning', confirmButtonText: '确认删除', cancelButtonText: '取消' },
    )
  } catch (error) {
    return
  }
  try {
    await warrantyApi.removeWarranty(row.id)
    ElMessage.success('质保登记已删除')
    load()
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  }
}

function handleSaved() {
  load()
}

// 支持从路灯台账携带 lamp_id 直接登记质保。
async function applyRouteQuery() {
  const lampId = Number(route.query.lamp_id)
  if (!lampId) return
  try {
    presetLamp.value = await lampApi.detail(lampId, { silent: true })
    editing.value = null
    formVisible.value = true
  } catch (error) {
    presetLamp.value = null
  }
  router.replace({ path: '/warranties' })
}

onMounted(async () => {
  await loadSuppliers()
  await applyRouteQuery()
})
</script>

<style scoped>
.cell-line {
  display: flex;
  align-items: center;
  gap: 8px;
}

.cell-sub {
  margin-top: 4px;
  font-size: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
</style>
