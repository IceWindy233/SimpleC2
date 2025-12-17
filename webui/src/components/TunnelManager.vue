<template>
  <div class="tunnel-manager">
    <div class="toolbar">
      <Button variant="primary" size="sm" @click="showCreateModal = true">
        New Tunnel
      </Button>
      <Button variant="ghost" size="sm" @click="fetchTunnels">
        Refresh
      </Button>
    </div>

    <div class="tunnels-list">
      <div v-if="tunnels.length === 0" class="empty-state">
        No active tunnels.
      </div>
      <div v-else class="tunnel-grid">
        <div v-for="tunnel in tunnels" :key="tunnel.ID" class="tunnel-card">
          <div class="tunnel-header">
            <span class="tunnel-target">{{ tunnel.Target }}</span>
            <span class="tunnel-status" :class="tunnel.Status">{{ tunnel.Status }}</span>
          </div>
          <div class="tunnel-meta">
            <div>ID: {{ tunnel.ID.substring(0, 8) }}...</div>
            <div>Active: {{ new Date(tunnel.LastActivity).toLocaleTimeString() }}</div>
          </div>
          <div class="tunnel-actions">
            <Button variant="danger" size="sm" @click="closeTunnel(tunnel.ID)">Close</Button>
          </div>
        </div>
      </div>
    </div>

    <!-- Create Tunnel Modal -->
    <div v-if="showCreateModal" class="modal-overlay">
      <div class="modal">
        <h3>Create Tunnel</h3>
        <div class="form-group">
          <label>Target (host:port)</label>
          <input v-model="newTunnelTarget" placeholder="e.g., 127.0.0.1:22" />
        </div>
        <div class="modal-actions">
          <Button variant="ghost" @click="showCreateModal = false">Cancel</Button>
          <Button variant="primary" @click="createTunnel" :disabled="!newTunnelTarget">Create</Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from './ui/Button.vue'
import { getTunnels, startTunnel, closeTunnel as apiCloseTunnel } from '../services/api'
import { useToastStore } from '../stores/toast'

const props = defineProps<{
  beaconId: string
}>()

const toast = useToastStore()
const tunnels = ref<any[]>([])
const showCreateModal = ref(false)
const newTunnelTarget = ref('')

const fetchTunnels = async () => {
  try {
    const res = await getTunnels()
    // Filter for current beacon
    if (res.data) {
        tunnels.value = res.data.filter((t: any) => t.BeaconID === props.beaconId)
    } else {
        tunnels.value = []
    }
  } catch (e) {
    console.error(e)
  }
}

const createTunnel = async () => {
  try {
    await startTunnel(props.beaconId, newTunnelTarget.value)
    toast.success('Tunnel creation initiated')
    showCreateModal.value = false
    newTunnelTarget.value = ''
    setTimeout(fetchTunnels, 1000)
  } catch (e) {
    toast.error('Failed to create tunnel')
  }
}

const closeTunnel = async (id: string) => {
  try {
    await apiCloseTunnel(id)
    toast.success('Tunnel closed')
    fetchTunnels()
  } catch (e) {
    toast.error('Failed to close tunnel')
  }
}

onMounted(() => {
  fetchTunnels()
  // Poll for updates?
  setInterval(fetchTunnels, 5000)
})
</script>

<style scoped>
.tunnel-manager {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
  height: 100%;
}

.toolbar {
  display: flex;
  gap: var(--spacing-sm);
}

.tunnel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: var(--spacing-md);
  overflow-y: auto;
}

.tunnel-card {
  background: var(--color-bg-secondary);
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
}

.tunnel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
}

.tunnel-status {
  font-size: 0.8rem;
  padding: 2px 6px;
  border-radius: 4px;
  text-transform: uppercase;
}

.tunnel-status.active { background: rgba(var(--color-success-rgb), 0.2); color: var(--color-success); }
.tunnel-status.pending { background: rgba(var(--color-warning-rgb), 0.2); color: var(--color-warning); }
.tunnel-status.closed { background: var(--color-bg-tertiary); color: var(--color-text-secondary); }
.tunnel-status.error { background: rgba(var(--color-danger-rgb), 0.2); color: var(--color-danger); }

.tunnel-meta {
  font-size: 0.85rem;
  color: var(--color-text-secondary);
}

.tunnel-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: auto;
}

.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0,0,0,0.7);
  display: flex;
  justify-content: center;
  align-items: center;
  z-index: 1000;
}

.modal {
  background: var(--color-bg-primary);
  padding: var(--spacing-lg);
  border-radius: var(--radius-lg);
  width: 400px;
  border: 1px solid var(--color-border);
}

.form-group {
  margin: var(--spacing-md) 0;
}

.form-group input {
  width: 100%;
  padding: 8px;
  margin-top: 4px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text-primary);
  border-radius: var(--radius-sm);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
}
</style>
