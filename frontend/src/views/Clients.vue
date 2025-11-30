<template>
  <div class="page-container">
    <div class="page-header">
      <h2>用户管理</h2>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>
        新建用户
      </el-button>
    </div>

    <el-card>
      <el-table :data="clients" v-loading="loading">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="uuid" label="UUID" width="280">
          <template #default="{ row }">
            <el-tooltip :content="row.uuid" placement="top">
              <span class="uuid-text">{{ row.uuid.substring(0, 8) }}...</span>
            </el-tooltip>
            <el-button size="small" link @click="copyUUID(row.uuid)">复制</el-button>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱" width="180" />
        <el-table-column label="流量" width="150">
          <template #default="{ row }">
            {{ formatBytes(row.used_gb) }} / {{ row.total_gb ? formatBytes(row.total_gb) : '无限制' }}
          </template>
        </el-table-column>
        <el-table-column prop="expire_at" label="到期时间" width="120">
          <template #default="{ row }">
            {{ row.expire_at ? formatDate(row.expire_at) : '永不过期' }}
          </template>
        </el-table-column>
        <el-table-column prop="enable" label="状态" width="80">
          <template #default="{ row }">
            <el-tag :type="row.enable ? 'success' : 'danger'">
              {{ row.enable ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" />
        <el-table-column label="操作" width="250" fixed="right">
          <template #default="{ row }">
            <el-button size="small" @click="openDialog(row)">编辑</el-button>
            <el-button size="small" @click="handleResetTraffic(row)">重置流量</el-button>
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

    <!-- 编辑对话框 -->
    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑用户' : '新建用户'" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="邮箱">
          <el-input v-model="form.email" placeholder="可选" />
        </el-form-item>
        <el-form-item label="流量限制(GB)">
          <el-input-number v-model="form.total_gb" :min="0" />
          <span class="tip">0 表示不限制</span>
        </el-form-item>
        <el-form-item label="到期时间">
          <el-date-picker v-model="form.expire_at" type="datetime" placeholder="不选择表示永不过期" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" rows="2" />
        </el-form-item>
        <el-form-item label="启用" v-if="form.id">
          <el-switch v-model="form.enable" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const dialogVisible = ref(false)
const formRef = ref<FormInstance>()
const clients = ref<any[]>([])

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0,
})

const form = reactive({
  id: 0,
  email: '',
  total_gb: 0,
  expire_at: null as Date | null,
  remark: '',
  enable: true,
})

const rules: FormRules = {}

onMounted(() => {
  fetchData()
})

async function fetchData() {
  loading.value = true
  try {
    const res = await api.clients.list(pagination.page, pagination.pageSize)
    clients.value = res.data.list
    pagination.total = res.data.total
  } finally {
    loading.value = false
  }
}

function openDialog(row?: any) {
  if (row) {
    Object.assign(form, {
      id: row.id,
      email: row.email,
      total_gb: row.total_gb ? row.total_gb / (1024 * 1024 * 1024) : 0,
      expire_at: row.expire_at ? new Date(row.expire_at) : null,
      remark: row.remark,
      enable: row.enable,
    })
  } else {
    Object.assign(form, {
      id: 0,
      email: '',
      total_gb: 0,
      expire_at: null,
      remark: '',
      enable: true,
    })
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  try {
    if (form.id) {
      await api.clients.update(form.id, form)
      ElMessage.success('更新成功')
    } else {
      await api.clients.create(form)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定要删除该用户吗？', '提示', { type: 'warning' })
  try {
    await api.clients.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

async function handleResetTraffic(row: any) {
  await ElMessageBox.confirm('确定要重置该用户的流量吗？', '提示', { type: 'warning' })
  try {
    await api.clients.resetTraffic(row.id)
    ElMessage.success('流量已重置')
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

function copyUUID(uuid: string) {
  navigator.clipboard.writeText(uuid)
  ElMessage.success('已复制到剪贴板')
}

function formatBytes(bytes: number): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(2)} ${units[i]}`
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

.uuid-text {
  font-family: monospace;
}

.tip {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}
</style>
