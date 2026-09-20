<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑供应商' : '新增供应商'"
    width="620px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="供应商名称" prop="name">
            <el-input v-model="form.name" placeholder="例如 华明光电股份有限公司" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="简称" prop="short_name">
            <el-input v-model="form.short_name" placeholder="例如 华明光电" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="联系人" prop="contact_person">
            <el-input v-model="form.contact_person" placeholder="售后对接人" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="联系电话" prop="contact_phone">
            <el-input v-model="form.contact_phone" placeholder="联系人手机" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="服务热线" prop="service_phone">
            <el-input v-model="form.service_phone" placeholder="400 售后电话" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="响应时限(小时)" prop="response_deadline_hours">
            <el-input-number v-model="form.response_deadline_hours" :min="1" :max="720" :step="4" style="width: 100%" />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="地址" prop="address">
            <el-input v-model="form.address" placeholder="选填" />
          </el-form-item>
        </el-col>
        <el-col :span="24">
          <el-form-item label="备注" prop="remark">
            <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="255" show-word-limit />
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
import { warrantyApi } from '@/api/warranty'
import { warrantyMeta } from '@/composables/useWarrantyMeta'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  model: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const formRef = ref(null)
const submitting = ref(false)

const isEdit = computed(() => Boolean(props.model?.id))
const defaultDeadline = computed(() => warrantyMeta.value?.default_response_deadline_hours ?? 24)

const createForm = () => ({
  name: '',
  short_name: '',
  contact_person: '',
  contact_phone: '',
  service_phone: '',
  address: '',
  response_deadline_hours: defaultDeadline.value,
  remark: '',
})

const form = reactive(createForm())

const rules = {
  name: [{ required: true, message: '请输入供应商名称', trigger: 'blur' }],
  response_deadline_hours: [{ required: true, message: '请填写承诺响应时限', trigger: 'blur' }],
}

function syncForm() {
  Object.assign(form, createForm())
  if (props.model) {
    Object.assign(form, {
      name: props.model.name ?? '',
      short_name: props.model.short_name ?? '',
      contact_person: props.model.contact_person ?? '',
      contact_phone: props.model.contact_phone ?? '',
      service_phone: props.model.service_phone ?? '',
      address: props.model.address ?? '',
      response_deadline_hours: props.model.response_deadline_hours ?? defaultDeadline.value,
      remark: props.model.remark ?? '',
    })
  }
}

async function handleSubmit() {
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await warrantyApi.updateSupplier(props.model.id, { ...form })
      ElMessage.success('供应商已更新')
    } else {
      await warrantyApi.createSupplier({ ...form })
      ElMessage.success('供应商已新增')
    }
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>
