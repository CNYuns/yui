<template>
  <div class="init-container">
    <div class="init-box">
      <div class="init-header">
        <h1>Y-UI</h1>
        <p>初始化管理员账号</p>
      </div>

      <el-alert
        type="info"
        title="首次使用"
        description="检测到系统尚未初始化，请创建管理员账号"
        :closable="false"
        show-icon
        style="margin-bottom: 20px"
      />

      <el-form ref="formRef" :model="form" :rules="rules" @submit.prevent="handleInit">
        <el-form-item prop="email">
          <el-input v-model="form.email" placeholder="管理员邮箱" size="large" prefix-icon="Message" />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="form.password"
            type="password"
            placeholder="密码（至少6位）"
            size="large"
            prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item prop="confirmPassword">
          <el-input
            v-model="form.confirmPassword"
            type="password"
            placeholder="确认密码"
            size="large"
            prefix-icon="Lock"
            show-password
          />
        </el-form-item>

        <el-form-item>
          <el-button type="primary" size="large" :loading="loading" native-type="submit" class="init-btn">
            创建管理员
          </el-button>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
import api from '@/api'

const router = useRouter()
const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  email: '',
  password: '',
  confirmPassword: '',
})

const validateConfirmPassword = (_rule: any, value: string, callback: any) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  email: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效的邮箱地址', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请确认密码', trigger: 'blur' },
    { validator: validateConfirmPassword, trigger: 'blur' },
  ],
}

async function handleInit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  loading.value = true
  try {
    await api.auth.initAdmin(form.email, form.password)
    ElMessage.success('管理员创建成功，请登录')
    router.push('/login')
  } catch (error: any) {
    ElMessage.error(error.message || '初始化失败')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.init-container {
  width: 100%;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.init-box {
  width: 420px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.init-header {
  text-align: center;
  margin-bottom: 20px;

  h1 {
    font-size: 32px;
    color: #303133;
    margin-bottom: 8px;
  }

  p {
    color: #909399;
    font-size: 14px;
  }
}

.init-btn {
  width: 100%;
}
</style>
