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
        <el-form-item prop="username">
          <el-input v-model="form.username" placeholder="用户名（字母和数字，至少2位）" size="large" prefix-icon="User" />
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

      <div class="password-tips">
        <p>密码要求：</p>
        <ul>
          <li>至少6个字符</li>
          <li>不能包含连续相同字符（如aaa）</li>
          <li>不能是连续字符（如abc、123）</li>
        </ul>
      </div>
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
  username: '',
  password: '',
  confirmPassword: '',
})

// 用户名验证：只允许字母和数字
const validateUsername = (_rule: any, value: string, callback: any) => {
  if (!/^[a-zA-Z0-9]+$/.test(value)) {
    callback(new Error('用户名只能包含字母和数字'))
  } else {
    callback()
  }
}

// 密码验证
const validatePassword = (_rule: any, value: string, callback: any) => {
  // 检查连续相同字符
  if (/(.)\1\1/.test(value)) {
    callback(new Error('密码不能包含3个连续相同的字符'))
    return
  }
  // 检查连续递增/递减字符
  for (let i = 0; i < value.length - 2; i++) {
    const c1 = value.charCodeAt(i)
    const c2 = value.charCodeAt(i + 1)
    const c3 = value.charCodeAt(i + 2)
    if ((c2 === c1 + 1 && c3 === c2 + 1) || (c2 === c1 - 1 && c3 === c2 - 1)) {
      callback(new Error('密码不能包含连续字符（如abc、321）'))
      return
    }
  }
  callback()
}

const validateConfirmPassword = (_rule: any, value: string, callback: any) => {
  if (value !== form.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

const rules: FormRules = {
  username: [
    { required: true, message: '请输入用户名', trigger: 'blur' },
    { min: 2, max: 32, message: '用户名长度为2-32个字符', trigger: 'blur' },
    { validator: validateUsername, trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 6, max: 64, message: '密码长度为6-64个字符', trigger: 'blur' },
    { validator: validatePassword, trigger: 'blur' },
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
    await api.auth.initAdmin(form.username, form.password)
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

.password-tips {
  margin-top: 20px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
  color: #606266;

  p {
    margin: 0 0 8px 0;
    font-weight: 500;
  }

  ul {
    margin: 0;
    padding-left: 20px;
    li {
      margin: 4px 0;
    }
  }
}
</style>
