<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑质保登记' : '登记质保'"
    width="640px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="所属路灯" prop="lamp_id">
        <el-select
          v-model="form.lamp_id"
          filterable
          remote
          reserve-keyword
          :disabled="isEdit"
          :remote-method="searchLamps"
          :loading="lampLoading"
          placeholder="输入路灯编号 / 道路名称搜索"
          style="width: 100%"
          @change="handleLampChange"
        >
          <el-option
            v-for="item in lampCandidates"
            :key="item.id"
            :label="`${item.code} · ${item.road_name} · ${item.name || '未命名'}`"
            :value="item.id"
          />
        </el-select>
      </el-form-item>

      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="质保部件" prop="component">
            <el-select v-model="form.component" style="width: 100%">
              <el-option v-for="(item, key) in WARRANTY_COMPONENT" :key="key" :label="item.label" :value="key" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="供应商" prop="supplier_id">
            <el-select
              v-model="form.supplier_id"
              filterable
              clearable
              placeholder="选择供应商后自动带出联系方式"
              style="width: 100%"
              @change="handleSupplierChange"
            >
              <el-option v-for="item in supplierOptions" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="联系人" prop="contact_person">
            <el-input v-model="form.contact_person" placeholder="厂家联系人" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="联系电话" prop="contact_phone">
            <el-input v-model="form.contact_phone" placeholder="厂家联系电话" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保开始" prop="start_date">
            <el-date-picker v-model="form.start_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="质保到期" prop="end_date">
            <el-date-picker v-model="form.end_date" type="date" value-format="YYYY-MM-DD" placeholder="选择日期" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="备注" prop="remark">
            <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="255" show-word-limit placeholder="例如: 整灯质保, 含驱动电源" />
          </el-form-item>
        </el-col>
      </el-row>
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
import { WARRANTY_COMPONENT } from '@/constants/dict'

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

const isEdit = computed(() => Boolean(props.model?.id))

const createForm = () => ({
  lamp_id: undefined,
  component: 'luminaire',
  supplier_id: undefined,
  contact_person: '',
  contact_phone: '',
  start_date: '',
  end_date: '',
  remark: '',
})

const form = reactive(createForm())

const rules = {
  lamp_id: [{ required: true, message: '请选择所属路灯', trigger: 'change' }],
  component: [{ required: true, message: '请选择质保部件', trigger: 'change' }],
  start_date: [{ required: true, message: '请选择质保开始日期', trigger: 'change' }],
  end_date: [{ required: true, message: '请选择质保到期日期', trigger: 'change' }],
}

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

function handleLampChange() {
  // 仅用于联动展示, 无需额外处理
}

// 选择供应商后自动带出档案中的联系方式, 仍可手动修改。
function handleSupplierChange(id) {
  const supplier = props.supplierOptions.find((item) => item.id === id)
  if (supplier) {
    form.contact_person = supplier.contact_person || ''
    form.contact_phone = supplier.contact_phone || ''
  }
}

// 打开弹窗时初始化: 编辑模式回填数据, 新增模式支持从台账页带入路灯。
async function syncForm() {
  Object.assign(form, createForm())

  if (props.model) {
    Object.assign(form, {
      lamp_id: props.model.lamp_id,
      component: props.model.component,
      supplier_id: props.model.supplier_id ?? undefined,
      contact_person: props.model.contact_person,
      contact_phone: props.model.contact_phone,
      start_date: props.model.start_date ? String(props.model.start_date).slice(0, 10) : '',
      end_date: props.model.end_date ? String(props.model.end_date).slice(0, 10) : '',
      remark: props.model.remark,
    })
    try {
      const lamp = await lampApi.detail(props.model.lamp_id, { silent: true })
      lampCandidates.value = lamp ? [lamp] : []
    } catch (error) {
      lampCandidates.value = []
    }
    return
  }

  await searchLamps('')
  if (props.presetLamp?.id) {
    form.lamp_id = props.presetLamp.id
    if (!lampCandidates.value.some((item) => item.id === props.presetLamp.id)) {
      lampCandidates.value = [props.presetLamp, ...lampCandidates.value]
    }
  }
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const payload = { ...form }
    if (!payload.supplier_id) {
      delete payload.supplier_id
    }
    if (isEdit.value) {
      const { lamp_id: _ignored, ...rest } = payload
      await warrantyApi.update(props.model.id, rest)
      ElMessage.success('质保登记已更新')
    } else {
      await warrantyApi.create(payload)
      ElMessage.success('质保登记成功')
    }
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>
