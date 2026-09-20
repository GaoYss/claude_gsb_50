<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑质保登记' : '灯具 / 灯杆质保登记'"
    width="760px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-form ref="formRef" :model="form" label-width="110px">
      <el-form-item v-if="!isEdit" label="路灯" prop="lamp_id">
        <el-select
          v-model="form.lamp_id"
          filterable
          remote
          reserve-keyword
          :remote-method="searchLamps"
          :loading="lampLoading"
          placeholder="输入路灯编号 / 道路 / 名称搜索"
          style="width: 100%"
          @change="handleLampChange"
        >
          <el-option
            v-for="item in lampCandidates"
            :key="item.id"
            :label="`${item.code} · ${item.road_name} · ${item.name || ''}`"
            :value="item.id"
          />
        </el-select>
      </el-form-item>

      <el-descriptions v-if="currentLamp" :column="3" border size="small" class="lamp-summary">
        <el-descriptions-item label="路灯编号">{{ currentLamp.code }}</el-descriptions-item>
        <el-descriptions-item label="所在道路">{{ currentLamp.road_name }}</el-descriptions-item>
        <el-descriptions-item label="灯具类型">{{ currentLamp.lamp_type || '-' }}</el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">灯具质保</el-divider>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="灯具供应商">
            <el-select v-model="form.lamp_supplier_id" filterable clearable placeholder="选择灯具厂家" style="width: 100%">
              <el-option v-for="item in supplierOptions" :key="`l-${item.id}`" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保月数">
            <el-input-number v-model="form.lamp_warranty_months" :min="0" :max="240" :step="12" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保开始日期">
            <el-date-picker
              v-model="form.lamp_start_at"
              type="date"
              value-format="YYYY-MM-DD"
              placeholder="开始日期"
              style="width: 100%"
              @change="syncLampEnd"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保截止日期">
            <el-date-picker v-model="form.lamp_end_at" type="date" value-format="YYYY-MM-DD" placeholder="截止日期" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-divider content-position="left">灯杆质保</el-divider>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="灯杆供应商">
            <el-select v-model="form.pole_supplier_id" filterable clearable placeholder="选择灯杆厂家" style="width: 100%">
              <el-option v-for="item in supplierOptions" :key="`p-${item.id}`" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保月数">
            <el-input-number v-model="form.pole_warranty_months" :min="0" :max="240" :step="12" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保开始日期">
            <el-date-picker
              v-model="form.pole_start_at"
              type="date"
              value-format="YYYY-MM-DD"
              placeholder="开始日期"
              style="width: 100%"
              @change="syncPoleEnd"
            />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保截止日期">
            <el-date-picker v-model="form.pole_end_at" type="date" value-format="YYYY-MM-DD" placeholder="截止日期" style="width: 100%" />
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item label="备注">
        <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="255" show-word-limit />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { lampApi } from '@/api/lamp'
import { warrantyApi } from '@/api/warranty'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  model: { type: Object, default: null },
  presetLamp: { type: Object, default: null },
  supplierOptions: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const formRef = ref(null)
const submitting = ref(false)
const lampLoading = ref(false)
const lampCandidates = ref([])
const selectedLamp = ref(null)

const isEdit = computed(() => Boolean(props.model?.id))
const currentLamp = computed(() => selectedLamp.value ?? props.presetLamp ?? null)

const createForm = () => ({
  lamp_id: undefined,
  lamp_supplier_id: null,
  lamp_start_at: '',
  lamp_end_at: '',
  lamp_warranty_months: 24,
  pole_supplier_id: null,
  pole_start_at: '',
  pole_end_at: '',
  pole_warranty_months: 24,
  remark: '',
})

const form = reactive(createForm())

async function searchLamps(keyword = '') {
  lampLoading.value = true
  try {
    const data = await lampApi.list({ keyword, page: 1, page_size: 20 }, { silent: true })
    lampCandidates.value = data?.items ?? []
  } catch (error) {
    lampCandidates.value = []
  } finally {
    lampLoading.value = false
  }
}

function handleLampChange(id) {
  selectedLamp.value = lampCandidates.value.find((item) => item.id === id) ?? null
}

// 选择开始日期并填写了质保月数时, 自动推算截止日期, 用户仍可手工调整。
function syncLampEnd() {
  if (form.lamp_start_at && form.lamp_warranty_months > 0 && !form.lamp_end_at) {
    form.lamp_end_at = addMonths(form.lamp_start_at, form.lamp_warranty_months)
  }
}

function syncPoleEnd() {
  if (form.pole_start_at && form.pole_warranty_months > 0 && !form.pole_end_at) {
    form.pole_end_at = addMonths(form.pole_start_at, form.pole_warranty_months)
  }
}

// 按本地日期做月份加法, 输出 YYYY-MM-DD, 避免 toISOString 的时区偏移。
function addMonths(value, months) {
  const [year, month, day] = value.split('-').map(Number)
  const date = new Date(year, month - 1, day)
  date.setMonth(date.getMonth() + months)
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

async function syncForm() {
  Object.assign(form, createForm())
  selectedLamp.value = null
  lampCandidates.value = []

  if (props.model) {
    Object.assign(form, {
      lamp_id: props.model.lamp_id,
      lamp_supplier_id: props.model.lamp_supplier_id ?? null,
      lamp_start_at: props.model.lamp_start_at?.slice(0, 10) ?? '',
      lamp_end_at: props.model.lamp_end_at?.slice(0, 10) ?? '',
      lamp_warranty_months: props.model.lamp_warranty_months || 0,
      pole_supplier_id: props.model.pole_supplier_id ?? null,
      pole_start_at: props.model.pole_start_at?.slice(0, 10) ?? '',
      pole_end_at: props.model.pole_end_at?.slice(0, 10) ?? '',
      pole_warranty_months: props.model.pole_warranty_months || 0,
      remark: props.model.remark ?? '',
    })
    try {
      selectedLamp.value = await lampApi.detail(props.model.lamp_id, { silent: true })
    } catch (error) {
      selectedLamp.value = null
    }
    return
  }

  if (props.presetLamp) {
    form.lamp_id = props.presetLamp.id
    selectedLamp.value = props.presetLamp
  }
  await searchLamps('')
}

async function handleSubmit() {
  if (!form.lamp_id) {
    ElMessage.warning('请选择需要登记质保的路灯')
    return
  }
  const hasLamp = Boolean(form.lamp_supplier_id && form.lamp_start_at && form.lamp_end_at)
  const hasPole = Boolean(form.pole_supplier_id && form.pole_start_at && form.pole_end_at)
  if (!hasLamp && !hasPole) {
    ElMessage.warning('请至少完整登记灯具或灯杆其中一项的供应商与起止日期')
    return
  }
  if ((form.lamp_supplier_id && (!form.lamp_start_at || !form.lamp_end_at)) ||
      (form.pole_supplier_id && (!form.pole_start_at || !form.pole_end_at))) {
    ElMessage.warning('供应商与质保起止日期需同时填写')
    return
  }
  if (hasLamp && form.lamp_end_at < form.lamp_start_at) {
    ElMessage.warning('灯具质保截止日期不能早于开始日期')
    return
  }
  if (hasPole && form.pole_end_at < form.pole_start_at) {
    ElMessage.warning('灯杆质保截止日期不能早于开始日期')
    return
  }

  const payload = { ...form }
  if (!form.lamp_supplier_id) {
    payload.lamp_supplier_id = null
    payload.lamp_start_at = ''
    payload.lamp_end_at = ''
    payload.lamp_warranty_months = 0
  }
  if (!form.pole_supplier_id) {
    payload.pole_supplier_id = null
    payload.pole_start_at = ''
    payload.pole_end_at = ''
    payload.pole_warranty_months = 0
  }

  submitting.value = true
  try {
    if (isEdit.value) {
      await warrantyApi.updateWarranty(props.model.id, payload)
      ElMessage.success('质保登记已更新')
    } else {
      await warrantyApi.createWarranty(payload)
      ElMessage.success('质保登记已保存')
    }
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.lamp-summary {
  margin-bottom: 8px;
}
</style>
