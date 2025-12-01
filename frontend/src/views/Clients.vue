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
        <el-table-column label="操作" width="300" fixed="right">
          <template #default="{ row }">
            <el-button size="small" type="primary" @click="showLinks(row)">订阅</el-button>
            <el-button size="small" @click="openDialog(row)">编辑</el-button>
            <el-button size="small" @click="handleResetTraffic(row)">重置</el-button>
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

    <!-- 订阅链接对话框 -->
    <el-dialog v-model="linksDialogVisible" title="订阅链接" width="700px">
      <div class="links-header">
        <el-input v-model="serverAddr" placeholder="服务器地址" style="width: 200px">
          <template #prepend>服务器</template>
        </el-input>
        <el-button @click="fetchLinks" :loading="linksLoading">刷新</el-button>
        <el-button type="success" @click="copySubUrl">复制订阅地址</el-button>
      </div>

      <el-divider>订阅地址</el-divider>
      <div class="sub-url">
        <el-input :model-value="subUrl" readonly>
          <template #append>
            <el-button @click="copySubUrl">复制</el-button>
          </template>
        </el-input>
        <div class="sub-tip">
          将此地址添加到代理客户端（如 V2rayN、Clash 等）的订阅功能中
        </div>
      </div>

      <el-divider>单独链接</el-divider>
      <el-table :data="clientLinks" v-loading="linksLoading" max-height="300">
        <el-table-column prop="protocol" label="协议" width="100">
          <template #default="{ row }">
            <el-tag>{{ row.protocol.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="tag" label="入站" width="120" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column prop="remark" label="备注" />
        <el-table-column label="操作" width="150">
          <template #default="{ row }">
            <el-button size="small" @click="copyLink(row.link)">复制链接</el-button>
            <el-button size="small" @click="showQRCode(row)">二维码</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 二维码对话框 -->
    <el-dialog v-model="qrDialogVisible" title="扫描二维码" width="350px">
      <div class="qr-container">
        <canvas ref="qrCanvas"></canvas>
        <p>{{ currentQRRemark }}</p>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import QRCode from 'qrcode'
import api from '@/api'

const loading = ref(false)
const dialogVisible = ref(false)
const linksDialogVisible = ref(false)
const qrDialogVisible = ref(false)
const linksLoading = ref(false)
const formRef = ref<FormInstance>()
const qrCanvas = ref<HTMLCanvasElement>()
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

// 订阅相关
const currentClient = ref<any>(null)
const clientLinks = ref<any[]>([])
const serverAddr = ref('')
const currentQRRemark = ref('')

const subUrl = computed(() => {
  if (!currentClient.value) return ''
  const host = serverAddr.value || window.location.host
  return `${window.location.protocol}//${host}/api/v1/sub/${currentClient.value.uuid}`
})

onMounted(() => {
  fetchData()
  // 默认使用当前访问的地址
  serverAddr.value = window.location.hostname
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

// 订阅链接功能
async function showLinks(row: any) {
  currentClient.value = row
  linksDialogVisible.value = true
  await fetchLinks()
}

async function fetchLinks() {
  if (!currentClient.value) return
  linksLoading.value = true
  try {
    const res = await api.clients.getLinks(currentClient.value.id, serverAddr.value)
    clientLinks.value = res.data || []
  } catch (error: any) {
    ElMessage.error(error.message)
    clientLinks.value = []
  } finally {
    linksLoading.value = false
  }
}

function copyLink(link: string) {
  navigator.clipboard.writeText(link)
  ElMessage.success('链接已复制')
}

function copySubUrl() {
  navigator.clipboard.writeText(subUrl.value)
  ElMessage.success('订阅地址已复制')
}

async function showQRCode(row: any) {
  currentQRRemark.value = row.remark || row.tag
  qrDialogVisible.value = true
  await nextTick()
  if (qrCanvas.value) {
    QRCode.toCanvas(qrCanvas.value, row.link, {
      width: 280,
      margin: 2,
    })
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
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

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

.links-header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}

.sub-url {
  .sub-tip {
    margin-top: 8px;
    font-size: 12px;
    color: #909399;
  }
}

.qr-container {
  text-align: center;

  p {
    margin-top: 10px;
    color: #606266;
  }
}
</style>
