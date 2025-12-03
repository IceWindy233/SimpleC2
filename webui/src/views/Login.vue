<template>
  <div class="login-container">
    <Card class="login-card">
      <template #header>
        <div class="login-header">
          <h2>SimpleC2</h2>
          <p>Sign in to your account</p>
        </div>
      </template>
      
      <form @submit.prevent="handleLogin">
        <Input 
          label="Username" 
          v-model="username" 
          placeholder="Enter username" 
          :disabled="loading"
        />
        <Input 
          label="Password" 
          type="password" 
          v-model="password" 
          placeholder="Enter password" 
          :disabled="loading"
        />
        
        <div class="form-actions">
          <Button 
            variant="primary" 
            block 
            type="submit" 
            :loading="loading"
          >
            Sign In
          </Button>
        </div>
      </form>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useToastStore } from '../stores/toast'
import Card from '../components/ui/Card.vue'
import Input from '../components/ui/Input.vue'
import Button from '../components/ui/Button.vue'

const router = useRouter()
const authStore = useAuthStore()
const toastStore = useToastStore()

const username = ref('')
const password = ref('')
const loading = ref(false)

const handleLogin = async () => {
  if (!username.value || !password.value) {
    toastStore.error('Please enter both username and password')
    return
  }

  loading.value = true
  try {
    await authStore.login(username.value, password.value)
    toastStore.success('Login successful')
    router.push('/')
  } catch (error: any) {
    const message = error.response?.data?.message || 'Login failed'
    toastStore.error(message)
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: var(--color-bg-tertiary);
}

.login-card {
  width: 100%;
  max-width: 400px;
}

.login-header {
  text-align: center;
  margin-bottom: var(--spacing-sm);
}

.login-header h2 {
  color: var(--color-primary);
  margin-bottom: var(--spacing-xs);
}

.login-header p {
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
}

.form-actions {
  margin-top: var(--spacing-lg);
}
</style>
