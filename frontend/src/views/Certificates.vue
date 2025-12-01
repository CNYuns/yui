<template>
  <div class="page-container">
    <div class="page-header">
      <h2>证书管理</h2>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>
        申请证书
      </el-button>
    </div>

    <el-card>
      <el-table :data="certificates" v-loading="loading">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="domain" label="域名" width="200" />
        <el-table-column prop="email" label="邮箱" width="180" />
        <el-table-column prop="provider" label="提供商" width="120" />
        <el-table-column prop="expire_at" label="到期时间" width="120">
          <template #default="{ row }">
            {{ formatDate(row.expire_at) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="auto_renew" label="自动续签" width="100">
          <template #default="{ row }">
            <el-switch v-model="row.auto_renew" @change="handleAutoRenewChange(row)" />
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="handleRenew(row)" :disabled="row.status === 'pending'">
              续签
            </el-button>
            <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        class="pagination"
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        layout="total, prev, pager, next"
        @current-change="fetchData"
      />
    </el-card>

    <!-- 申请对话框 -->
    <el-dialog v-model="dialogVisible" title="申请证书" width="500px">
      <el-alert type="info" :closable="false" style="margin-bottom: 20px">
        请确保域名已正确解析到当前服务器，且80端口可访问。
      </el-alert>

      <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
        <el-form-item label="域名" prop="domain">
          <el-input v-model="form.domain" placeholder="example.com" />
        </el-form-item>
        <el-form-item label="邮箱" prop="email">
          <el-input v-model="form.email" placeholder="用于接收证书到期提醒" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">申请</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const certificates = ref<any[]>([])

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0,
})

const form = reactive({
  domain: '',
  email: '',
})

const rules: FormRules = {
  domain: [{ required: true, message: '请输入域名', trigger: 'blur' }],
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' },
  ],
}

onMounted(() => {
  fetchData()
})

async function fetchData() {
  loading.value = true
  try {
    const res = await api.certificates.list(pagination.page, pagination.pageSize)
    certificates.value = res.data.list
    pagination.total = res.data.total
  } finally {
    loading.value = false
  }
}

function openDialog() {
  form.domain = ''
  form.email = ''
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    await api.certificates.request(form.domain, form.email)
    ElMessage.success('证书申请成功')
    dialogVisible.value = false
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  } finally {
    submitting.value = false
  }
}

async function handleRenew(row: any) {
  await ElMessageBox.confirm('确定要续签该证书吗？', '提示')
  try {
    await api.certificates.renew(row.id)
    ElMessage.success('续签成功')
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定要删除该证书吗？', '提示', { type: 'warning' })
  try {
    await api.certificates.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

async function handleAutoRenewChange(row: any) {
  try {
    await api.certificates.updateAutoRenew(row.id, row.auto_renew)
    ElMessage.success('设置已更新')
  } catch (error: any) {
    row.auto_renew = !row.auto_renew
    ElMessage.error(error.message)
  }
}

function getStatusType(status: string) {
  const map: Record<string, string> = {
    active: 'success',
    pending: 'warning',
    expired: 'danger',
    error: 'danger',
  }
  return map[status] || 'info'
}

function formatDate(date: string): string {
  return new Date(date).toLocaleDateString('zh-CN')
}
</script>

<style scoped lang="scss">
.pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
</style>
