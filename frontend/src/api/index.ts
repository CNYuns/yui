import axios from 'axios'
import { ElMessage } from 'element-plus'

const http = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

// 请求拦截器
http.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器
http.interceptors.response.use(
  (response) => {
    const res = response.data
    if (res.code !== 0) {
      ElMessage.error(res.msg || '请求失败')
      return Promise.reject(new Error(res.msg))
    }
    return res
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    } else {
      ElMessage.error(error.response?.data?.msg || error.message || '网络错误')
    }
    return Promise.reject(error)
  }
)

export default {
  auth: {
    check: () => http.get('/auth/check'),
    login: (username: string, password: string) => http.post('/auth/login', { username, password }),
    logout: () => http.post('/auth/logout'),
    getProfile: () => http.get('/auth/profile'),
    changePassword: (oldPassword: string, newPassword: string) =>
      http.put('/auth/password', { old_password: oldPassword, new_password: newPassword }),
    initAdmin: (username: string, password: string) => http.post('/auth/init', { username, password }),
  },

  users: {
    list: (page = 1, pageSize = 20) => http.get('/users', { params: { page, page_size: pageSize } }),
    get: (id: number) => http.get(`/users/${id}`),
    create: (data: any) => http.post('/users', data),
    update: (id: number, data: any) => http.put(`/users/${id}`, data),
    delete: (id: number) => http.delete(`/users/${id}`),
  },

  clients: {
    list: (page = 1, pageSize = 20) => http.get('/clients', { params: { page, page_size: pageSize } }),
    get: (id: number) => http.get(`/clients/${id}`),
    create: (data: any) => http.post('/clients', data),
    update: (id: number, data: any) => http.put(`/clients/${id}`, data),
    delete: (id: number) => http.delete(`/clients/${id}`),
    resetTraffic: (id: number) => http.post(`/clients/${id}/reset-traffic`),
    getLinks: (id: number, server?: string) => http.get(`/clients/${id}/links`, { params: { server } }),
  },

  inbounds: {
    list: (page = 1, pageSize = 20) => http.get('/inbounds', { params: { page, page_size: pageSize } }),
    get: (id: number) => http.get(`/inbounds/${id}`),
    create: (data: any) => http.post('/inbounds', data),
    update: (id: number, data: any) => http.put(`/inbounds/${id}`, data),
    delete: (id: number) => http.delete(`/inbounds/${id}`),
    getClients: (id: number) => http.get(`/inbounds/${id}/clients`),
    addClient: (id: number, clientId: number) => http.post(`/inbounds/${id}/clients`, { client_id: clientId }),
    removeClient: (id: number, clientId: number) => http.delete(`/inbounds/${id}/clients/${clientId}`),
  },

  outbounds: {
    list: (page = 1, pageSize = 20) => http.get('/outbounds', { params: { page, page_size: pageSize } }),
    get: (id: number) => http.get(`/outbounds/${id}`),
    create: (data: any) => http.post('/outbounds', data),
    update: (id: number, data: any) => http.put(`/outbounds/${id}`, data),
    delete: (id: number) => http.delete(`/outbounds/${id}`),
  },

  stats: {
    getSummary: () => http.get('/stats/summary'),
    getDaily: (days = 30) => http.get('/stats/daily', { params: { days } }),
    getClientTraffic: (id: number, startDate?: string, endDate?: string) =>
      http.get(`/stats/client/${id}`, { params: { start_date: startDate, end_date: endDate } }),
    getInboundTraffic: (id: number, startDate?: string, endDate?: string) =>
      http.get(`/stats/inbound/${id}`, { params: { start_date: startDate, end_date: endDate } }),
  },

  certificates: {
    list: (page = 1, pageSize = 20) => http.get('/certificates', { params: { page, page_size: pageSize } }),
    get: (id: number) => http.get(`/certificates/${id}`),
    request: (domain: string, email: string) => http.post('/certificates', { domain, email }),
    renew: (id: number) => http.post(`/certificates/${id}/renew`),
    updateAutoRenew: (id: number, autoRenew: boolean) => http.put(`/certificates/${id}/auto-renew`, { auto_renew: autoRenew }),
    delete: (id: number) => http.delete(`/certificates/${id}`),
  },

  system: {
    getStatus: () => http.get('/system/status'),
    reload: () => http.post('/system/reload'),
    restart: () => http.post('/system/restart'),
    getConfig: () => http.get('/system/config'),
    checkPort: (port: number) => http.get('/system/check-port', { params: { port } }),
    checkUpdate: () => http.get('/system/check-update'),
  },

  audit: {
    list: (params: { page?: number; page_size?: number; user_id?: number; action?: string; resource?: string }) =>
      http.get('/audits', { params }),
  },
}
