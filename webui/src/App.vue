<template>
  <MainLayout />
  <ToastContainer />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import MainLayout from './components/MainLayout.vue'
import ToastContainer from './components/ui/ToastContainer.vue'
import { webSocketService } from './services/websocket'
import { useAuthStore } from './stores/auth'

const authStore = useAuthStore()

// Watch authentication state to handle login/logout
watch(() => authStore.isAuthenticated, (isAuthenticated) => {
  if (isAuthenticated) {
    webSocketService.connect()
  } else {
    webSocketService.disconnect()
  }
})

onMounted(() => {
  if (authStore.isAuthenticated) {
    webSocketService.connect()
  }
})

onUnmounted(() => {
  webSocketService.disconnect()
})
</script>

<style>
/* Global styles are imported in main.ts */
</style>
