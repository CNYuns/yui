<template>
  <div class="page-container">
    <div class="page-header">
      <h2>入站管理</h2>
      <el-button type="primary" @click="openDialog()">
        <el-icon><Plus /></el-icon>
        新建入站
      </el-button>
    </div>

    <el-card>
      <el-table :data="inbounds" v-loading="loading">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="tag" label="标签" width="150" />
        <el-table-column prop="protocol" label="协议" width="120">
          <template #default="{ row }">
            <el-tag>{{ row.protocol.toUpperCase() }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="port" label="端口" width="100" />
        <el-table-column label="传输" width="120">
          <template #default="{ row }">
            {{ getTransportType(row) }}
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
            <el-button size="small" @click="showClients(row)">用户</el-button>
            <el-button size="small" @click="openDialog(row)">编辑</el-button>
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
    <el-dialog v-model="dialogVisible" :title="form.id ? '编辑入站' : '新建入站'" width="700px" top="5vh">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-tabs v-model="activeTab">
          <el-tab-pane label="基本设置" name="basic">
            <el-form-item label="标签" prop="tag">
              <el-input v-model="form.tag" placeholder="唯一标识，如 vmess-in" />
            </el-form-item>
            <el-form-item label="协议" prop="protocol">
              <el-select v-model="form.protocol" placeholder="选择协议" @change="onProtocolChange">
                <el-option label="VMess" value="vmess" />
                <el-option label="VLESS" value="vless" />
                <el-option label="Trojan" value="trojan" />
                <el-option label="Shadowsocks" value="shadowsocks" />
                <el-option label="SOCKS" value="socks" />
                <el-option label="HTTP" value="http" />
                <el-option label="Dokodemo-Door (任意门)" value="dokodemo-door" />
                <el-option label="WireGuard" value="wireguard" />
              </el-select>
            </el-form-item>

            <!-- SOCKS/HTTP 认证设置 - 直接显示在协议选择下方 -->
            <template v-if="form.protocol === 'socks' || form.protocol === 'http'">
              <el-form-item label="认证方式">
                <el-radio-group v-model="protocolSettings.auth">
                  <el-radio label="noauth">无需认证</el-radio>
                  <el-radio label="password">密码认证</el-radio>
                </el-radio-group>
              </el-form-item>
              <template v-if="protocolSettings.auth === 'password'">
                <el-form-item label="用户名">
                  <el-input v-model="protocolSettings.user" placeholder="输入用户名" />
                </el-form-item>
                <el-form-item label="密码">
                  <el-input v-model="protocolSettings.pass" type="password" placeholder="输入密码" show-password />
                </el-form-item>
              </template>
              <el-form-item label="允许UDP" v-if="form.protocol === 'socks'">
                <el-switch v-model="protocolSettings.udp" />
              </el-form-item>
            </template>

            <!-- Shadowsocks 设置 - 直接显示 -->
            <template v-if="form.protocol === 'shadowsocks'">
              <el-form-item label="加密方式">
                <el-select v-model="protocolSettings.method">
                  <el-option label="aes-256-gcm" value="aes-256-gcm" />
                  <el-option label="aes-128-gcm" value="aes-128-gcm" />
                  <el-option label="chacha20-poly1305" value="chacha20-poly1305" />
                  <el-option label="2022-blake3-aes-256-gcm" value="2022-blake3-aes-256-gcm" />
                </el-select>
              </el-form-item>
              <el-form-item label="密码">
                <div style="display: flex; gap: 10px;">
                  <el-input v-model="protocolSettings.password" placeholder="Shadowsocks 密码" style="flex: 1" />
                  <el-button @click="generateSSPassword">随机生成</el-button>
                </div>
              </el-form-item>
            </template>

            <el-form-item label="端口" prop="port">
              <el-input-number v-model="form.port" :min="1" :max="65535" />
            </el-form-item>
            <el-form-item label="监听地址">
              <el-input v-model="form.listen" placeholder="0.0.0.0 (所有接口)" />
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.remark" type="textarea" rows="2" />
            </el-form-item>
            <el-form-item label="启用">
              <el-switch v-model="form.enable" />
            </el-form-item>
          </el-tab-pane>

          <el-tab-pane label="协议设置" name="protocol">
            <!-- SOCKS/HTTP 账号密码设置 -->
            <template v-if="form.protocol === 'socks' || form.protocol === 'http'">
              <el-alert type="info" :closable="false" style="margin-bottom: 16px">
                {{ form.protocol === 'socks' ? 'SOCKS5' : 'HTTP' }} 代理认证设置（可选）
              </el-alert>
              <el-form-item label="认证方式">
                <el-radio-group v-model="protocolSettings.auth">
                  <el-radio label="noauth">无需认证</el-radio>
                  <el-radio label="password">密码认证</el-radio>
                </el-radio-group>
              </el-form-item>
              <template v-if="protocolSettings.auth === 'password'">
                <el-form-item label="用户名">
                  <el-input v-model="protocolSettings.user" placeholder="输入用户名" />
                </el-form-item>
                <el-form-item label="密码">
                  <el-input v-model="protocolSettings.pass" type="password" placeholder="输入密码" show-password />
                </el-form-item>
              </template>
              <el-form-item label="允许UDP" v-if="form.protocol === 'socks'">
                <el-switch v-model="protocolSettings.udp" />
              </el-form-item>
            </template>

            <!-- Shadowsocks 设置 -->
            <template v-if="form.protocol === 'shadowsocks'">
              <el-alert type="info" :closable="false" style="margin-bottom: 16px">
                Shadowsocks 加密设置
              </el-alert>
              <el-form-item label="加密方式">
                <el-select v-model="protocolSettings.method">
                  <el-option label="2022-blake3-aes-128-gcm" value="2022-blake3-aes-128-gcm" />
                  <el-option label="2022-blake3-aes-256-gcm" value="2022-blake3-aes-256-gcm" />
                  <el-option label="2022-blake3-chacha20-poly1305" value="2022-blake3-chacha20-poly1305" />
                  <el-option label="aes-256-gcm" value="aes-256-gcm" />
                  <el-option label="aes-128-gcm" value="aes-128-gcm" />
                  <el-option label="chacha20-poly1305" value="chacha20-poly1305" />
                  <el-option label="xchacha20-poly1305" value="xchacha20-poly1305" />
                </el-select>
              </el-form-item>
              <el-form-item label="密码">
                <el-input v-model="protocolSettings.password" placeholder="Shadowsocks 密码" />
                <el-button type="primary" link @click="generateSSPassword">随机生成</el-button>
              </el-form-item>
            </template>

            <!-- Trojan 设置 -->
            <template v-if="form.protocol === 'trojan'">
              <el-alert type="info" :closable="false" style="margin-bottom: 16px">
                Trojan 用户通过"用户管理"添加。创建入站后点击"用户"按钮添加。
              </el-alert>
            </template>

            <!-- VMess/VLESS 设置 -->
            <template v-if="form.protocol === 'vmess' || form.protocol === 'vless'">
              <el-alert type="info" :closable="false" style="margin-bottom: 16px">
                {{ form.protocol.toUpperCase() }} 用户通过"用户管理"添加。创建入站后点击"用户"按钮添加。
              </el-alert>
              <template v-if="form.protocol === 'vless'">
                <el-form-item label="VLESS Flow">
                  <el-select v-model="protocolSettings.flow" placeholder="选择 Flow（可选）">
                    <el-option label="无" value="" />
                    <el-option label="xtls-rprx-vision" value="xtls-rprx-vision" />
                  </el-select>
                </el-form-item>
              </template>
            </template>

            <!-- Dokodemo-door 设置 -->
            <template v-if="form.protocol === 'dokodemo-door'">
              <el-alert type="info" :closable="false" style="margin-bottom: 16px">
                任意门协议用于端口转发或透明代理
              </el-alert>
              <el-form-item label="目标地址">
                <el-input v-model="protocolSettings.address" placeholder="如 1.1.1.1" />
              </el-form-item>
              <el-form-item label="目标端口">
                <el-input-number v-model="protocolSettings.destPort" :min="1" :max="65535" />
              </el-form-item>
              <el-form-item label="网络协议">
                <el-checkbox-group v-model="protocolSettings.networks">
                  <el-checkbox label="tcp" />
                  <el-checkbox label="udp" />
                </el-checkbox-group>
              </el-form-item>
              <el-form-item label="跟随重定向">
                <el-switch v-model="protocolSettings.followRedirect" />
              </el-form-item>
            </template>
          </el-tab-pane>

          <el-tab-pane label="传输配置" name="transport">
            <el-form-item label="传输方式">
              <el-select v-model="streamSettings.network" placeholder="选择传输方式">
                <el-option label="TCP" value="tcp" />
                <el-option label="WebSocket" value="ws" />
                <el-option label="gRPC" value="grpc" />
                <el-option label="HTTP/2" value="h2" />
                <el-option label="QUIC" value="quic" />
              </el-select>
            </el-form-item>

            <!-- WebSocket 设置 -->
            <template v-if="streamSettings.network === 'ws'">
              <el-form-item label="Path">
                <el-input v-model="streamSettings.wsSettings.path" placeholder="/ws" />
              </el-form-item>
              <el-form-item label="Host">
                <el-input v-model="streamSettings.wsSettings.host" placeholder="可选" />
              </el-form-item>
            </template>

            <!-- gRPC 设置 -->
            <template v-if="streamSettings.network === 'grpc'">
              <el-form-item label="serviceName">
                <el-input v-model="streamSettings.grpcSettings.serviceName" placeholder="grpc" />
              </el-form-item>
            </template>

            <!-- HTTP/2 设置 -->
            <template v-if="streamSettings.network === 'h2'">
              <el-form-item label="Path">
                <el-input v-model="streamSettings.httpSettings.path" placeholder="/h2" />
              </el-form-item>
              <el-form-item label="Host">
                <el-input v-model="streamSettings.httpSettings.host" placeholder="可选" />
              </el-form-item>
            </template>

            <el-divider>TLS 设置</el-divider>

            <el-form-item label="TLS">
              <el-select v-model="streamSettings.security">
                <el-option label="无" value="none" />
                <el-option label="TLS" value="tls" />
                <el-option label="Reality" value="reality" />
              </el-select>
            </el-form-item>

            <template v-if="streamSettings.security === 'tls'">
              <el-form-item label="SNI">
                <el-input v-model="streamSettings.tlsSettings.serverName" placeholder="域名" />
              </el-form-item>
              <el-form-item label="ALPN">
                <el-select v-model="streamSettings.tlsSettings.alpn" multiple placeholder="选择ALPN">
                  <el-option label="h2" value="h2" />
                  <el-option label="http/1.1" value="http/1.1" />
                </el-select>
              </el-form-item>
            </template>

            <template v-if="streamSettings.security === 'reality'">
              <el-form-item label="目标地址">
                <el-input v-model="streamSettings.realitySettings.dest" placeholder="example.com:443" />
              </el-form-item>
              <el-form-item label="SNI">
                <el-input v-model="streamSettings.realitySettings.serverNames" placeholder="example.com" />
              </el-form-item>
              <el-form-item label="Private Key">
                <el-input v-model="streamSettings.realitySettings.privateKey" placeholder="私钥" />
              </el-form-item>
              <el-form-item label="Short IDs">
                <el-input v-model="streamSettings.realitySettings.shortIds" placeholder="逗号分隔" />
              </el-form-item>
            </template>
          </el-tab-pane>

          <el-tab-pane label="嗅探设置" name="sniffing">
            <el-form-item label="启用嗅探">
              <el-switch v-model="sniffingEnabled" />
            </el-form-item>
            <template v-if="sniffingEnabled">
              <el-form-item label="目标协议">
                <el-checkbox-group v-model="sniffingDestOverride">
                  <el-checkbox label="http" />
                  <el-checkbox label="tls" />
                  <el-checkbox label="quic" />
                  <el-checkbox label="fakedns" />
                </el-checkbox-group>
              </el-form-item>
              <el-form-item label="仅匹配IP">
                <el-switch v-model="sniffingMetadataOnly" />
              </el-form-item>
            </template>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSubmit">确定</el-button>
      </template>
    </el-dialog>

    <!-- 用户管理对话框 -->
    <el-dialog v-model="clientsDialogVisible" title="入站用户管理" width="600px">
      <div class="clients-header">
        <el-select v-model="selectedClientId" placeholder="选择用户添加" style="width: 300px">
          <el-option
            v-for="client in availableClients"
            :key="client.id"
            :label="client.email || client.uuid.substring(0, 8)"
            :value="client.id"
          />
        </el-select>
        <el-button type="primary" @click="addClientToInbound" :disabled="!selectedClientId">
          添加
        </el-button>
      </div>
      <el-table :data="inboundClients" v-loading="clientsLoading">
        <el-table-column prop="uuid" label="UUID" width="280">
          <template #default="{ row }">
            <span class="uuid-text">{{ row.uuid.substring(0, 16) }}...</span>
          </template>
        </el-table-column>
        <el-table-column prop="email" label="邮箱/备注" />
        <el-table-column label="操作" width="100">
          <template #default="{ row }">
            <el-button size="small" type="danger" @click="removeClientFromInbound(row)">移除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
import api from '@/api'

const loading = ref(false)
const dialogVisible = ref(false)
const clientsDialogVisible = ref(false)
const clientsLoading = ref(false)
const formRef = ref<FormInstance>()
const inbounds = ref<any[]>([])
const activeTab = ref('basic')

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0,
})

const form = reactive({
  id: 0,
  tag: '',
  protocol: 'vmess',
  port: 10000,
  listen: '0.0.0.0',
  remark: '',
  enable: true,
})

const streamSettings = reactive({
  network: 'tcp',
  security: 'none',
  wsSettings: { path: '/ws', host: '' },
  grpcSettings: { serviceName: 'grpc' },
  httpSettings: { path: '/h2', host: '' },
  tlsSettings: { serverName: '', alpn: ['h2', 'http/1.1'] as string[] },
  realitySettings: { dest: '', serverNames: '', privateKey: '', shortIds: '' },
})

const sniffingEnabled = ref(true)
const sniffingDestOverride = ref(['http', 'tls'])
const sniffingMetadataOnly = ref(false)

// 协议设置
const protocolSettings = reactive({
  // SOCKS/HTTP
  auth: 'noauth',
  user: '',
  pass: '',
  udp: true,
  // Shadowsocks
  method: 'aes-256-gcm',
  password: '',
  // VLESS
  flow: '',
  // Dokodemo-door
  address: '',
  destPort: 443,
  networks: ['tcp', 'udp'],
  followRedirect: false,
})

// 用户管理相关
const currentInboundId = ref(0)
const inboundClients = ref<any[]>([])
const availableClients = ref<any[]>([])
const selectedClientId = ref<number | null>(null)

const rules: FormRules = {
  tag: [{ required: true, message: '请输入标签', trigger: 'blur' }],
  protocol: [{ required: true, message: '请选择协议', trigger: 'change' }],
  port: [{ required: true, message: '请输入端口', trigger: 'blur' }],
}

onMounted(() => {
  fetchData()
})

async function fetchData() {
  loading.value = true
  try {
    const res = await api.inbounds.list(pagination.page, pagination.pageSize)
    inbounds.value = res.data.list
    pagination.total = res.data.total
  } finally {
    loading.value = false
  }
}

function getTransportType(row: any): string {
  try {
    const stream = JSON.parse(row.stream_settings || '{}')
    const network = stream.network || 'tcp'
    const security = stream.security || 'none'
    return security === 'none' ? network : `${network}+${security}`
  } catch {
    return 'tcp'
  }
}

function onProtocolChange() {
  // 根据协议调整默认设置
  if (form.protocol === 'socks') {
    protocolSettings.auth = 'noauth'
    protocolSettings.udp = true
  } else if (form.protocol === 'shadowsocks') {
    protocolSettings.method = 'aes-256-gcm'
    if (!protocolSettings.password) {
      generateSSPassword()
    }
  }
}

function generateSSPassword() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  let password = ''
  for (let i = 0; i < 16; i++) {
    password += chars.charAt(Math.floor(Math.random() * chars.length))
  }
  protocolSettings.password = password
}

function openDialog(row?: any) {
  activeTab.value = 'basic'
  if (row) {
    Object.assign(form, {
      id: row.id,
      tag: row.tag,
      protocol: row.protocol,
      port: row.port,
      listen: row.listen,
      remark: row.remark,
      enable: row.enable,
    })
    // 解析传输设置
    try {
      const stream = JSON.parse(row.stream_settings || '{}')
      streamSettings.network = stream.network || 'tcp'
      streamSettings.security = stream.security || 'none'
      if (stream.wsSettings) Object.assign(streamSettings.wsSettings, stream.wsSettings)
      if (stream.grpcSettings) Object.assign(streamSettings.grpcSettings, stream.grpcSettings)
      if (stream.httpSettings) Object.assign(streamSettings.httpSettings, stream.httpSettings)
      if (stream.tlsSettings) {
        streamSettings.tlsSettings.serverName = stream.tlsSettings.serverName || ''
        streamSettings.tlsSettings.alpn = stream.tlsSettings.alpn || ['h2', 'http/1.1']
      }
    } catch {}
    // 解析嗅探设置
    try {
      const sniffing = JSON.parse(row.sniffing || '{}')
      sniffingEnabled.value = sniffing.enabled !== false
      sniffingDestOverride.value = sniffing.destOverride || ['http', 'tls']
      sniffingMetadataOnly.value = sniffing.metadataOnly || false
    } catch {}
  } else {
    Object.assign(form, {
      id: 0,
      tag: '',
      protocol: 'vmess',
      port: Math.floor(Math.random() * 55535) + 10000,
      listen: '0.0.0.0',
      remark: '',
      enable: true,
    })
    Object.assign(streamSettings, {
      network: 'tcp',
      security: 'none',
      wsSettings: { path: '/ws', host: '' },
      grpcSettings: { serviceName: 'grpc' },
      httpSettings: { path: '/h2', host: '' },
      tlsSettings: { serverName: '', alpn: ['h2', 'http/1.1'] },
      realitySettings: { dest: '', serverNames: '', privateKey: '', shortIds: '' },
    })
    sniffingEnabled.value = true
    sniffingDestOverride.value = ['http', 'tls']
    sniffingMetadataOnly.value = false
  }
  dialogVisible.value = true
}

async function handleSubmit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  // 构建传输设置
  const stream: any = {
    network: streamSettings.network,
    security: streamSettings.security,
  }
  if (streamSettings.network === 'ws') {
    stream.wsSettings = { path: streamSettings.wsSettings.path }
    if (streamSettings.wsSettings.host) {
      stream.wsSettings.headers = { Host: streamSettings.wsSettings.host }
    }
  } else if (streamSettings.network === 'grpc') {
    stream.grpcSettings = { serviceName: streamSettings.grpcSettings.serviceName }
  } else if (streamSettings.network === 'h2') {
    stream.httpSettings = { path: streamSettings.httpSettings.path }
    if (streamSettings.httpSettings.host) {
      stream.httpSettings.host = [streamSettings.httpSettings.host]
    }
  }
  if (streamSettings.security === 'tls') {
    stream.tlsSettings = {
      serverName: streamSettings.tlsSettings.serverName,
      alpn: streamSettings.tlsSettings.alpn,
    }
  } else if (streamSettings.security === 'reality') {
    stream.realitySettings = {
      dest: streamSettings.realitySettings.dest,
      serverNames: streamSettings.realitySettings.serverNames.split(',').map(s => s.trim()),
      privateKey: streamSettings.realitySettings.privateKey,
      shortIds: streamSettings.realitySettings.shortIds.split(',').map(s => s.trim()),
    }
  }

  // 构建嗅探设置
  const sniffing = sniffingEnabled.value ? {
    enabled: true,
    destOverride: sniffingDestOverride.value,
    metadataOnly: sniffingMetadataOnly.value,
  } : { enabled: false }

  // 构建协议设置
  let settings: any = {}
  if (form.protocol === 'socks') {
    settings = {
      auth: protocolSettings.auth,
      udp: protocolSettings.udp,
      ip: '127.0.0.1',
    }
    if (protocolSettings.auth === 'password' && protocolSettings.user && protocolSettings.pass) {
      settings.accounts = [{ user: protocolSettings.user, pass: protocolSettings.pass }]
    }
  } else if (form.protocol === 'http') {
    settings = {}
    if (protocolSettings.auth === 'password' && protocolSettings.user && protocolSettings.pass) {
      settings.accounts = [{ user: protocolSettings.user, pass: protocolSettings.pass }]
    }
  } else if (form.protocol === 'shadowsocks') {
    settings = {
      method: protocolSettings.method,
      password: protocolSettings.password,
      network: 'tcp,udp',
    }
  } else if (form.protocol === 'dokodemo-door') {
    settings = {
      address: protocolSettings.address || '',
      port: protocolSettings.destPort || 0,
      network: protocolSettings.networks.join(','),
      followRedirect: protocolSettings.followRedirect,
    }
  } else if (form.protocol === 'vless' && protocolSettings.flow) {
    settings = { decryption: 'none', flow: protocolSettings.flow }
  }

  const submitData = {
    ...form,
    settings: Object.keys(settings).length > 0 ? settings : undefined,
    stream_settings: stream,
    sniffing: sniffing,
  }

  try {
    if (form.id) {
      await api.inbounds.update(form.id, submitData)
      ElMessage.success('更新成功')
    } else {
      await api.inbounds.create(submitData)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

async function handleDelete(row: any) {
  await ElMessageBox.confirm('确定要删除该入站配置吗？', '提示', { type: 'warning' })
  try {
    await api.inbounds.delete(row.id)
    ElMessage.success('删除成功')
    fetchData()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

// 用户管理
async function showClients(row: any) {
  currentInboundId.value = row.id
  clientsDialogVisible.value = true
  await fetchInboundClients()
  await fetchAvailableClients()
}

async function fetchInboundClients() {
  clientsLoading.value = true
  try {
    const res = await api.inbounds.getClients(currentInboundId.value)
    inboundClients.value = res.data || []
  } finally {
    clientsLoading.value = false
  }
}

async function fetchAvailableClients() {
  try {
    const res = await api.clients.list(1, 100)
    // 过滤掉已添加的用户
    const existingIds = new Set(inboundClients.value.map((c: any) => c.id))
    availableClients.value = (res.data.list || []).filter((c: any) => !existingIds.has(c.id))
  } catch {}
}

async function addClientToInbound() {
  if (!selectedClientId.value) return
  try {
    await api.inbounds.addClient(currentInboundId.value, selectedClientId.value)
    ElMessage.success('添加成功')
    selectedClientId.value = null
    await fetchInboundClients()
    await fetchAvailableClients()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
}

async function removeClientFromInbound(client: any) {
  await ElMessageBox.confirm('确定要从入站移除该用户吗？', '提示', { type: 'warning' })
  try {
    await api.inbounds.removeClient(currentInboundId.value, client.id)
    ElMessage.success('移除成功')
    await fetchInboundClients()
    await fetchAvailableClients()
  } catch (error: any) {
    ElMessage.error(error.message)
  }
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

.clients-header {
  display: flex;
  gap: 10px;
  margin-bottom: 20px;
}

.uuid-text {
  font-family: monospace;
  font-size: 12px;
}
</style>
