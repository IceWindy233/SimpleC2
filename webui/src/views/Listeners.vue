<template>
  <div class="listeners-view">
    <div class="header-section">
      <h1>Listeners</h1>
      <Button variant="primary" @click="showCreateModal = true">
        + New Listener
      </Button>
    </div>

    <Card>
      <Table :columns="columns" :data="listeners" :loading="loading">
        <template #status="{ value }">
          <span :class="value === 'Running' ? 'text-success' : 'text-danger'">
            {{ value }}
          </span>
        </template>
        <template #actions="{ row }">
          <div class="actions">
            <Button variant="ghost" size="sm" @click="stopListener(row)">Stop</Button>
            <Button variant="ghost" size="sm" class="text-danger" @click="deleteListener(row)">Delete</Button>
          </div>
        </template>
      </Table>
    </Card>

    <Modal v-model="showCreateModal" title="Create Listener">
      <div class="form-stack">
        <Input label="Name" v-model="newListener.name" placeholder="e.g. HTTP-8080" />
        <Input label="Port" v-model="newListener.port" type="number" placeholder="8080" />
        <Input label="Type" v-model="newListener.type" disabled />
      </div>
      <template #footer>
        <Button variant="ghost" @click="showCreateModal = false">Cancel</Button>
        <Button variant="primary" @click="createListener">Create</Button>
      </template>
    </Modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '../components/ui/Card.vue'
import Table from '../components/ui/Table.vue'
import Button from '../components/ui/Button.vue'
import Modal from '../components/ui/Modal.vue'
import Input from '../components/ui/Input.vue'
import { useToastStore } from '../stores/toast'
import api from '../services/api'

const toast = useToastStore()
const loading = ref(false)
const showCreateModal = ref(false)

const columns = [
  { key: 'Name', label: 'Name' },
  { key: 'Type', label: 'Type' },
  // Port is inside Config JSON string, might need parsing or backend adjustment
  // For now, let's just show Name and Type as returned by backend
  { key: 'actions', label: 'Actions', width: '150px' }
]

const listeners = ref<any[]>([])

const newListener = ref({
  name: '',
  port: '',
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
    // Construct config JSON
    const config = JSON.stringify({ port: Number(newListener.value.port) })
    
    await api.post('/listeners', {
      name: newListener.value.name,
      type: newListener.value.type,
      config
    })
    
    toast.success('Listener created successfully')
    showCreateModal.value = false
    newListener.value = { name: '', port: '', type: 'HTTP' }
    fetchListeners()
  } catch (error: any) {
    const message = error.response?.data?.message || 'Failed to create listener'
    toast.error(message)
  } finally {
    loading.value = false
  }
}

const stopListener = (row: any) => {
  // Backend doesn't have stop endpoint, only delete
  toast.info(`Stopping listener ${row.Name}...`)
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
