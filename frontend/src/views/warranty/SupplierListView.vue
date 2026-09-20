<template>
  <div class="page">
    <PageHeader title="质保供应商" description="维护灯具与灯杆厂家档案、联系方式及承诺响应时限">
      <el-button :icon="Refresh" @click="load">刷新</el-button>
      <el-button type="primary" :icon="Plus" @click="openCreate">新增供应商</el-button>
    </PageHeader>

    <el-card shadow="never">
      <div class="filter-bar">
        <el-input v-model="query.keyword" placeholder="名称 / 简称 / 联系人 / 电话" clearable @keyup.enter="handleSearch" />
        <el-button type="primary" :icon="Search" @click="handleSearch">查询</el-button>
        <el-button :icon="RefreshLeft" @click="handleReset">重置</el-button>
      </div>
    </el-card>

    <el-card shadow="never">
      <el-table v-loading="loading" :data="rows" stripe>
        <el-table-column prop="name" label="供应商名称" min-width="200" show-overflow-tooltip />
        <el-table-column prop="short_name" label="简称" width="120" />
        <el-table-column prop="contact_person" label="联系人" width="100" />
        <el-table-column prop="contact_phone" label="联系电话" width="130" />
        <el-table-column prop="service_phone" label="服务热线" width="130" />
        <el-table-column label="响应时限" width="120" align="center">
          <template #default="{ row }">
            <el-tag type="warning" effect="plain">{{ row.response_deadline_hours }} 小时</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="address" label="地址" min-width="160" show-overflow-tooltip />
        <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
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

    <SupplierFormDialog v-model="formVisible" :model="editing" @saved="handleSaved" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, RefreshLeft, Search } from '@element-plus/icons-vue'
import PageHeader from '@/components/common/PageHeader.vue'
import DataPagination from '@/components/common/DataPagination.vue'
import SupplierFormDialog from './components/SupplierFormDialog.vue'
import { warrantyApi } from '@/api/warranty'
import { useListPage } from '@/composables/useListPage'

const { loading, rows, total, query, load, search, reset, changePage, changePageSize } = useListPage(
  warrantyApi.listSuppliers,
  { keyword: '' },
)

const formVisible = ref(false)
const editing = ref(null)

function handleSearch() {
  search()
}

function handleReset() {
  reset()
}

function openCreate() {
  editing.value = null
  formVisible.value = true
}

function openEdit(row) {
  editing.value = { ...row }
  formVisible.value = true
}

async function handleDelete(row) {
  try {
    await ElMessageBox.confirm(`确认删除供应商「${row.name}」? 被质保登记或未闭环工单引用时无法删除。`, '删除确认', {
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
    load()
  } catch (error) {
    // 错误提示由请求拦截器统一处理
  }
}

function handleSaved() {
  load()
}
</script>
