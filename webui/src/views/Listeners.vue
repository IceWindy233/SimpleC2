<template>
  <div class="listeners-view">
    <div class="header-section">
      <h1>Listeners</h1>
      <Button variant="primary" @click="showCreateModal = true">
        + Generate Certs
      </Button>
    </div>

    <Card>
      <Table :columns="columns" :data="listeners" :loading="loading">
        <template #active="{ value }">
          <span :class="value ? 'text-success' : 'text-danger'">
            {{ value ? 'Online' : 'Offline' }}
          </span>
        </template>
        <template #actions="{ row }">
          <div class="actions">
            <Button variant="ghost" size="sm" @click="startListener(row)" title="Start">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"></polygon></svg>
            </Button>
            <Button variant="ghost" size="sm" @click="stopListener(row)" title="Stop">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="6" y="4" width="4" height="16"></rect><rect x="14" y="4" width="4" height="16"></rect></svg>
            </Button>
            <Button variant="ghost" size="sm" @click="restartListener(row)" title="Restart">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M23 4v6h-6"></path><path d="M1 20v-6h6"></path><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"></path></svg>
            </Button>
            <Button variant="ghost" size="sm" class="text-danger" @click="deleteListener(row)" title="Delete">
                <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="3 6 5 6 21 6"></polyline><path d="M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"></path></svg>
            </Button>
          </div>
        </template>
      </Table>
    </Card>

    <Modal v-model="showCreateModal" title="Generate mTLS Certificates">
      <div class="form-stack">
        <Input label="Name" v-model="newListener.name" placeholder="e.g. HTTP-8080" />
        <Input label="Port" v-model="newListener.port" type="number" placeholder="8080" />
        <Input label="Type" v-model="newListener.type" disabled />
      </div>
      <template #footer>
        <Button variant="ghost" @click="showCreateModal = false">Cancel</Button>
        <Button variant="primary" @click="createListener">Generate & Download</Button>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import Card from '../components/ui/Card.vue'
import Table from '../components/ui/Table.vue'
import Button from '../components/ui/Button.vue'
import Modal from '../components/ui/Modal.vue'
import Input from '../components/ui/Input.vue'
import { useToastStore } from '../stores/toast'
import api from '../services/api'
import { webSocketService } from '../services/websocket'

const toast = useToastStore()
const loading = ref(false)
const showCreateModal = ref(false)

const columns = [
  { key: 'Name', label: 'Name' },
  { key: 'Type', label: 'Type' },
  { key: 'active', label: 'Status' },
  { key: 'actions', label: 'Actions', width: '200px' }
]

const listeners = ref<any[]>([])

const handleWebSocketMessage = (message: any) => {
    if (message.type === 'LISTENER_STARTED' || message.type === 'LISTENER_STOPPED') {
        const updatedListener = message.payload
        const index = listeners.value.findIndex(l => l.Name === updatedListener.Name)
        if (index !== -1) {
            // Update active status locally
            listeners.value[index].active = updatedListener.active
        } else {
             // Or reload if new/unknown
             fetchListeners() 
        }
    }
}

const newListener = ref({
  name: '',
  port: '8888',
  type: 'HTTP'
})

const fetchListeners = async () => {
  loading.value = true
  try {
    const response = await api.get('/listeners')
    listeners.value = response.data.data || []
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const createListener = async () => {
  loading.value = true
  try {
    // Default to 8888 if empty
    const portVal = newListener.value.port ? Number(newListener.value.port) : 8888
    
    // Construct config JSON
    const config = JSON.stringify({ port: portVal })
    
    // Request with responseType 'blob' to handle zip download
    const response = await api.post('/listeners', {
      name: newListener.value.name,
      type: newListener.value.type,
      config
    }, {
      responseType: 'blob'
    })
    
    // Trigger download
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    link.setAttribute('download', `listener_certs_${newListener.value.name}.zip`)
    document.body.appendChild(link)
    link.click()
    link.parentNode?.removeChild(link)
    
    toast.success('mTLS certificates generated. Please deploy them to your listener.')
    showCreateModal.value = false
    newListener.value = { name: '', port: '8888', type: 'HTTP' }
    // No need to fetch listeners immediately
  } catch (error: any) {
    console.error(error)
    toast.error('Failed to generate listener configuration')
  } finally {
    loading.value = false
  }
}

const startListener = async (row: any) => {
    try {
        await api.post(`/listeners/${row.Name}/start`)
        toast.success(`Started listener ${row.Name}`)
    } catch (error: any) {
        toast.error(`Failed to start listener: ${error.response?.data?.error || error.message}`)
    }
}

const stopListener = async (row: any) => {
    try {
        await api.post(`/listeners/${row.Name}/stop`)
        toast.success(`Stopped listener ${row.Name}`)
    } catch (error: any) {
        toast.error(`Failed to stop listener: ${error.response?.data?.error || error.message}`)
    }
}

const restartListener = async (row: any) => {
    try {
        await api.post(`/listeners/${row.Name}/restart`)
        toast.success(`Restarted listener ${row.Name}`)
    } catch (error: any) {
        toast.error(`Failed to restart listener: ${error.response?.data?.error || error.message}`)
    }
}

const deleteListener = async (row: any) => {
  if (confirm(`Are you sure you want to delete ${row.Name}?`)) {
    try {
      await api.delete(`/listeners/${row.Name}`)
      toast.success('Listener deleted')
      fetchListeners()
    } catch (error: any) {
      toast.error('Failed to delete listener')
    }
  }
}


onMounted(() => {
  fetchListeners()
  webSocketService.addMessageHandler(handleWebSocketMessage)
})

onUnmounted(() => {
  webSocketService.removeMessageHandler(handleWebSocketMessage)
})
</script>

<style scoped>
.listeners-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.text-success { color: var(--color-success); font-weight: 600; }
.text-danger { color: var(--color-danger); font-weight: 600; }

.actions {
  display: flex;
  gap: var(--spacing-xs);
}

.form-stack {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}
</style>
