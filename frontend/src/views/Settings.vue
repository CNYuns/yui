<template>
  <div class="page-container">
    <div class="page-header">
      <h2>系统设置</h2>
    </div>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card>
          <template #header>
            <span>Xray 控制</span>
          </template>

          <div class="control-item">
            <span>运行状态:</span>
            <el-tag :type="xrayRunning ? 'success' : 'danger'" size="large">
              {{ xrayRunning ? '运行中' : '已停止' }}
            </el-tag>
          </div>

          <div class="control-item">
            <span>Xray 版本:</span>
            <span>{{ xrayVersion }}</span>
          </div>

          <el-divider />

          <el-space>
            <el-button type="primary" @click="handleReload" :loading="reloading">
              重载配置
            </el-button>
            <el-button type="warning" @click="handleRestart" :loading="restarting">
              重启 Xray
            </el-button>
          </el-space>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card>
          <template #header>
            <span>系统信息</span>
          </template>

          <el-descriptions :column="1" border>
            <el-descriptions-item label="主机名">{{ systemStatus.hostname }}</el-descriptions-item>
            <el-descriptions-item label="平台">{{ systemStatus.platform }}</el-descriptions-item>
            <el-descriptions-item label="系统/架构">{{ systemStatus.os }}/{{ systemStatus.arch }}</el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ formatUptime(systemStatus.uptime) }}</el-descriptions-item>
          </el-descriptions>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20" style="margin-top: 20px">
      <el-col :span="24">
        <el-card>
          <template #header>
            <span>当前 Xray 配置</span>
          </template>

          <el-button @click="fetchConfig" :loading="loadingConfig" style="margin-bottom: 10px">
            刷新配置
          </el-button>

          <pre class="config-preview">{{ configText }}</pre>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'

const reloading = ref(false)
const restarting = ref(false)
const loadingConfig = ref(false)
const xrayRunning = ref(false)
const xrayVersion = ref('')
const configText = ref('')

const systemStatus = reactive({
  hostname: '',
  platform: '',
  os: '',
  arch: '',
  uptime: 0,
})

onMounted(async () => {
  await fetchStatus()
  await fetchConfig()
})

async function fetchStatus() {
  try {
    const res = await api.system.getStatus()
    Object.assign(systemStatus, res.data)
    xrayRunning.value = res.data.xray_running
    xrayVersion.value = res.data.xray_version
  } catch (e) {
    console.error(e)
  }
}

async function fetchConfig() {
  loadingConfig.value = true
  try {
    const res = await api.system.getConfig()
    configText.value = JSON.stringify(res.data, null, 2)
  } catch {
    configText.value = '加载失败'
  } finally {
    loadingConfig.value = false
  }
}

async function handleReload() {
  reloading.value = true
  try {
    await api.system.reload()
    ElMessage.success('配置重载成功')
    await fetchStatus()
  } catch (error: any) {
    ElMessage.error(error.message)
  } finally {
    reloading.value = false
  }
}

async function handleRestart() {
  restarting.value = true
  try {
    await api.system.restart()
    ElMessage.success('Xray 重启成功')
    await fetchStatus()
  } catch (error: any) {
    ElMessage.error(error.message)
  } finally {
    restarting.value = false
  }
}

function formatUptime(seconds: number): string {
  if (!seconds) return '0秒'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${days}天 ${hours}时 ${mins}分`
}
</script>

<style scoped lang="scss">
.control-item {
  display: flex;
  align-items: center;
  margin-bottom: 16px;

  span:first-child {
    width: 100px;
    color: #606266;
  }
}

.config-preview {
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 4px;
  padding: 16px;
  max-height: 400px;
  overflow: auto;
  font-family: monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>
