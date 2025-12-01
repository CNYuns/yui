<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <el-row :gutter="20" class="stat-row">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-value">{{ stats.totalClients }}</div>
          <div class="stat-label">总用户数</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-value">{{ stats.activeClients }}</div>
          <div class="stat-label">活跃用户</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-value">{{ formatBytes(stats.totalUpload) }}</div>
          <div class="stat-label">总上传</div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-value">{{ formatBytes(stats.totalDownload) }}</div>
          <div class="stat-label">总下载</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 系统状态 -->
    <el-row :gutter="20">
      <el-col :span="12">
        <el-card class="system-card">
          <template #header>
            <div class="card-header">
              <span>系统状态</span>
              <div class="xray-controls">
                <el-tag :type="systemStatus.xrayRunning ? 'success' : 'danger'" size="small">
                  Xray: {{ systemStatus.xrayRunning ? '运行中' : '已停止' }}
                </el-tag>
                <el-button-group size="small" style="margin-left: 10px">
                  <el-button @click="restartXray" :loading="xrayLoading">重启</el-button>
                  <el-button @click="reloadXray" :loading="xrayLoading">重载</el-button>
                </el-button-group>
              </div>
            </div>
          </template>

          <el-descriptions :column="2" size="small">
            <el-descriptions-item label="主机名">{{ systemStatus.hostname }}</el-descriptions-item>
            <el-descriptions-item label="系统">{{ systemStatus.os }}/{{ systemStatus.arch }}</el-descriptions-item>
            <el-descriptions-item label="运行时间">{{ formatUptime(systemStatus.uptime) }}</el-descriptions-item>
            <el-descriptions-item label="Xray版本">{{ systemStatus.xrayVersion }}</el-descriptions-item>
          </el-descriptions>

          <el-divider />

          <div class="resource-item">
            <span>CPU 使用率</span>
            <el-progress :percentage="cpuUsage" :stroke-width="8" />
          </div>

          <div class="resource-item">
            <span>内存使用率</span>
            <el-progress :percentage="memoryUsage" :stroke-width="8" />
          </div>

          <div class="resource-item">
            <span>磁盘使用率</span>
            <el-progress :percentage="diskUsage" :stroke-width="8" />
          </div>
        </el-card>
      </el-col>

      <el-col :span="12">
        <el-card class="chart-card">
          <template #header>
            <span>流量趋势（近30天）</span>
          </template>
          <div ref="chartRef" class="chart"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onBeforeUnmount, computed } from 'vue'
import { ElMessage } from 'element-plus'
import * as echarts from 'echarts'
import api from '@/api'

const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null
const xrayLoading = ref(false)

const stats = reactive({
  totalClients: 0,
  activeClients: 0,
  totalUpload: 0,
  totalDownload: 0,
})

const systemStatus = reactive<any>({
  hostname: '',
  os: '',
  arch: '',
  uptime: 0,
  xrayRunning: false,
  xrayVersion: '',
  cpu: { usage: [] },
  memory: { usedPercent: 0 },
  disk: { usedPercent: 0 },
})

const cpuUsage = computed(() => {
  if (!systemStatus.cpu?.usage?.length) return 0
  const avg = systemStatus.cpu.usage.reduce((a: number, b: number) => a + b, 0) / systemStatus.cpu.usage.length
  return Math.round(avg)
})

const memoryUsage = computed(() => {
  return Math.round(systemStatus.memory?.used_percent || 0)
})

const diskUsage = computed(() => {
  return Math.round(systemStatus.disk?.used_percent || 0)
})

onMounted(async () => {
  await Promise.all([fetchStats(), fetchSystemStatus(), fetchDailyStats()])
})

onBeforeUnmount(() => {
  if (chart) {
    chart.dispose()
    chart = null
  }
})

async function fetchStats() {
  try {
    const res = await api.stats.getSummary()
    Object.assign(stats, {
      totalClients: res.data.total_clients,
      activeClients: res.data.active_clients,
      totalUpload: res.data.total_upload,
      totalDownload: res.data.total_download,
    })
  } catch (e) {
    console.error(e)
  }
}

async function fetchSystemStatus() {
  try {
    const res = await api.system.getStatus()
    Object.assign(systemStatus, {
      hostname: res.data.hostname,
      os: res.data.os,
      arch: res.data.arch,
      uptime: res.data.uptime,
      xrayRunning: res.data.xray_running,
      xrayVersion: res.data.xray_version,
      cpu: res.data.cpu,
      memory: res.data.memory,
      disk: res.data.disk,
    })
  } catch (e) {
    console.error(e)
  }
}

async function restartXray() {
  xrayLoading.value = true
  try {
    await api.system.restart()
    ElMessage.success('Xray 重启成功')
    await fetchSystemStatus()
  } catch (e: any) {
    ElMessage.error(e.message || '重启失败')
  } finally {
    xrayLoading.value = false
  }
}

async function reloadXray() {
  xrayLoading.value = true
  try {
    await api.system.reload()
    ElMessage.success('Xray 配置重载成功')
    await fetchSystemStatus()
  } catch (e: any) {
    ElMessage.error(e.message || '重载失败')
  } finally {
    xrayLoading.value = false
  }
}

async function fetchDailyStats() {
  try {
    const res = await api.stats.getDaily(30)
    initChart(res.data || [])
  } catch (e) {
    initChart([])
  }
}

function initChart(data: any[]) {
  if (!chartRef.value) return

  chart = echarts.init(chartRef.value)
  chart.setOption({
    tooltip: { trigger: 'axis' },
    legend: { data: ['上传', '下载'] },
    xAxis: {
      type: 'category',
      data: data.map((d) => d.date),
    },
    yAxis: {
      type: 'value',
      axisLabel: {
        formatter: (value: number) => formatBytes(value, true),
      },
    },
    series: [
      {
        name: '上传',
        type: 'line',
        smooth: true,
        data: data.map((d) => d.upload),
      },
      {
        name: '下载',
        type: 'line',
        smooth: true,
        data: data.map((d) => d.download),
      },
    ],
  })
}

function formatBytes(bytes: number, short = false): string {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  const value = (bytes / Math.pow(1024, i)).toFixed(2)
  return short ? `${value}${units[i]}` : `${value} ${units[i]}`
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
.dashboard {
  .stat-row {
    margin-bottom: 20px;
  }

  .stat-card {
    text-align: center;

    .stat-value {
      font-size: 28px;
      font-weight: 600;
      color: #409eff;
    }

    .stat-label {
      margin-top: 8px;
      color: #909399;
      font-size: 14px;
    }
  }

  .system-card {
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
    }

    .xray-controls {
      display: flex;
      align-items: center;
    }

    .resource-item {
      margin-bottom: 16px;

      span {
        display: block;
        margin-bottom: 8px;
        font-size: 14px;
        color: #606266;
      }
    }
  }

  .chart-card {
    .chart {
      height: 300px;
    }
  }
}
</style>
