<template>
  <el-dialog
    :model-value="modelValue"
    title="转由自有班组接手"
    width="520px"
    @update:model-value="$emit('update:modelValue', $event)"
    @open="syncForm"
  >
    <el-alert
      type="warning"
      :closable="false"
      class="takeover-alert"
      :title="`厂家「${claim?.supplier_name || '-'}」超过承诺时限仍未完成处置, 转办后由自有班组接手维修。`"
      description="转办后工单不再计入厂家未闭环数量, 但仍保留质保内标记用于费用追溯。"
      show-icon
    />
    <el-form ref="formRef" :model="form" label-width="90px">
      <el-form-item label="接手班组" required>
        <el-select v-model="form.team" filterable allow-create placeholder="选择或输入班组" style="width: 100%">
          <el-option v-for="item in teamOptions" :key="item" :label="item" :value="item" />
        </el-select>
      </el-form-item>
      <el-form-item label="经手人">
        <el-input v-model="form.by" placeholder="调度员 / 值班长" />
      </el-form-item>
      <el-form-item label="转办原因">
        <el-input
          v-model="form.reason"
          type="textarea"
          :rows="3"
          maxlength="255"
          show-word-limit
          placeholder="例如: 厂家超过 24 小时承诺时限未到场, 为尽快恢复照明应急接管"
        />
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button type="warning" :loading="submitting" @click="handleSubmit">确认转办</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { warrantyApi } from '@/api/warranty'
import { useDictStore } from '@/stores/dict'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  claim: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'saved'])

const dictStore = useDictStore()
const formRef = ref(null)
const submitting = ref(false)

const createForm = () => ({ team: '', by: '', reason: '' })
const form = reactive(createForm())

const teamOptions = computed(() => dictStore.repairMeta.teams ?? [])

function syncForm() {
  Object.assign(form, createForm())
}

async function handleSubmit() {
  if (!form.team) {
    ElMessage.warning('请选择或输入接手班组')
    return
  }
  submitting.value = true
  try {
    await warrantyApi.takeoverClaim(props.claim.id, { ...form })
    ElMessage.success('已转由自有班组接手')
    emit('update:modelValue', false)
    emit('saved')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.takeover-alert {
  margin-bottom: 16px;
}
</style>
