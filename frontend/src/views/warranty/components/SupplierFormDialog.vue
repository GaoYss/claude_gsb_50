<template>
  <el-dialog
    :model-value="modelValue"
    :title="isEdit ? '编辑供应商' : '新增供应商'"
    width="560px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
      <el-form-item label="名称" prop="name">
        <el-input v-model="form.name" placeholder="例如 明辉照明设备有限公司" />
      </el-form-item>
      <el-row :gutter="16">
        <el-col :span="12">
          <el-form-item label="联系人" prop="contact_person">
            <el-input v-model="form.contact_person" placeholder="厂家联系人" />
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item label="联系电话" prop="contact_phone">
            <el-input v-model="form.contact_phone" placeholder="手机或座机" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="邮箱" prop="email">
        <el-input v-model="form.email" placeholder="选填" />
      </el-form-item>
      <el-form-item label="地址" prop="address">
        <el-input v-model="form.address" placeholder="选填" />
      </el-form-item>
      <el-form-item label="备注" prop="remark">
        <el-input v-model="form.remark" type="textarea" :rows="2" maxlength="255" show-word-limit placeholder="例如: 灯具与驱动电源供应商" />
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
import { warrantyApi } from '@/api/warranty'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  model: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const formRef = ref(null)
const submitting = ref(false)
const isEdit = computed(() => Boolean(props.model?.id))

const createForm = () => ({
  name: '',
  contact_person: '',
  contact_phone: '',
  email: '',
  address: '',
  remark: '',
})

const form = reactive(createForm())

const rules = {
  name: [{ required: true, message: '请输入供应商名称', trigger: 'blur' }],
}

// 打开弹窗时同步表单数据, 区分新增与编辑。
function syncForm() {
  Object.assign(form, createForm())
  if (props.model) {
    Object.assign(form, {
      name: props.model.name,
      contact_person: props.model.contact_person,
      contact_phone: props.model.contact_phone,
      email: props.model.email,
      address: props.model.address,
      remark: props.model.remark,
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
      ElMessage.success('供应商已创建')
    }
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>
