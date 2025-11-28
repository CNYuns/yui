<template>
  <div class="page-container">
    <div class="page-header">
      <h2>流量统计</h2>
    </div>

    <el-row :gutter="20">
      <el-col :span="24">
        <el-card>
          <template #header>
            <span>流量趋势</span>
          </template>
          <div ref="chartRef" class="chart"></div>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import * as echarts from 'echarts'
import api from '@/api'

const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts

onMounted(async () => {
  await fetchData()
})

async function fetchData() {
  try {
    const res = await api.stats.getDaily(30)
    initChart(res.data || [])
  } catch {
    initChart([])
  }
}

function initChart(data: any[]) {
  if (!chartRef.value) return

  chart = echarts.init(chartRef.value)
  chart.setOption({
    tooltip: {
      trigger: 'axis',
      formatter: (params: any) => {
        const date = params[0].name
        let html = `<div style="font-weight:bold">${date}</div>`
        params.forEach((p: any) => {
          html += `<div>${p.seriesName}: ${formatBytes(p.value)}</div>`
        })
        return html
      },
    },
    legend: {
      data: ['上传', '下载'],
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
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
        areaStyle: { opacity: 0.3 },
        data: data.map((d) => d.upload),
      },
      {
        name: '下载',
        type: 'line',
        smooth: true,
        areaStyle: { opacity: 0.3 },
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
</script>

<style scoped lang="scss">
.chart {
  height: 400px;
}
</style>
