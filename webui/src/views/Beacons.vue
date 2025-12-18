<template>
  <div class="beacons-view">
    <div class="header-section">
      <h1>Beacons</h1>
    </div>

    <Card>
      <Table 
        :columns="columns" 
        :data="beacons" 
        :loading="loading"
        @row-click="interact"
      >
        <template #BeaconID="{ value }">
           <span class="mono-font" :title="value">{{ value.split('-')[0] }}</span>
        </template>
        <template #Status="{ value }">
          <span class="status-dot" :class="{ online: value === 'active' }"></span>
          {{ value === 'active' ? 'Online' : 'Offline' }}
        </template>
        <template #LastSeen="{ value }">
          {{ formatTimeAgo(value) }}
        </template>
        <template #Note="{ row }">
          <div class="note-cell" @click.stop="openEditNote(row)">
            <span v-if="row.Note" class="note-text">{{ row.Note }}</span>
            <span v-else class="note-placeholder">Add note...</span>
            <span class="edit-icon">✎</span>
          </div>
        </template>
      </Table>
    </Card>

    <!-- Edit Note Modal -->
    <div v-if="showNoteModal" class="modal-overlay">
      <div class="modal">
        <h3>Edit Note</h3>
        <p class="modal-desc">Add a note/tag for beacon {{ editingBeacon?.Hostname }}</p>
        <div class="form-group">
          <input 
            v-model="noteInput" 
            @keyup.enter="saveNote"
            ref="noteInputRef"
            type="text" 
            placeholder="Enter note..." 
            class="full-width-input"
          />
        </div>
        <div class="modal-actions">
          <Button variant="ghost" @click="closeNoteModal">Cancel</Button>
          <Button variant="primary" @click="saveNote">Save</Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import Card from '../components/ui/Card.vue'
import Table from '../components/ui/Table.vue'
import Button from '../components/ui/Button.vue'
import api, { updateBeacon } from '../services/api'
import { useToastStore } from '../stores/toast'
import { webSocketService } from '../services/websocket'

const router = useRouter()
const toast = useToastStore()
const loading = ref(false)

const columns = [
  { key: 'BeaconID', label: 'ID', width: '100px' },
  { key: 'Status', label: 'Status', width: '100px' },
  { key: 'Listener', label: 'Listener' },
  { key: 'InternalIP', label: 'Internal IP' },
  { key: 'Hostname', label: 'Hostname' },
  { key: 'Username', label: 'User' },
  { key: 'OS', label: 'OS' },
  { key: 'Note', label: 'Note' },
  { key: 'LastSeen', label: 'Last' },
]

const beacons = ref<any[]>([])
const showNoteModal = ref(false)
const editingBeacon = ref<any>(null)
const noteInput = ref('')
const noteInputRef = ref<HTMLInputElement | null>(null)
const now = ref(Date.now())
let timer: any = null

const fetchBeacons = async () => {
  loading.value = true
  try {
    const response = await api.get('/beacons')
    beacons.value = response.data.data || []
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

const formatTimeAgo = (timestamp: string | number) => {
  if (!timestamp) return '-'
  const date = new Date(timestamp)
  // Use the reactive 'now' ref to trigger re-renders
  const diffMs = now.value - date.getTime()
  const diffSec = Math.max(0, Math.floor(diffMs / 1000))

  if (diffSec < 60) return `${diffSec}s ago`
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`
  return `${Math.floor(diffSec / 86400)}d ago`
}

const interact = (row: any) => {
  router.push(`/beacons/${row.BeaconID}`)
}

const openEditNote = (row: any) => {
  editingBeacon.value = row
  noteInput.value = row.Note || ''
  showNoteModal.value = true
  nextTick(() => {
    noteInputRef.value?.focus()
  })
}

const closeNoteModal = () => {
  showNoteModal.value = false
  editingBeacon.value = null
  noteInput.value = ''
}

const saveNote = async () => {
  if (!editingBeacon.value) return
  
  try {
    const updated = await updateBeacon(editingBeacon.value.BeaconID, { note: noteInput.value })
    // Optimistic update or wait for re-fetch/socket? 
    // WebSocket usually broadcasts update, but we can update local state immediately
    editingBeacon.value.Note = noteInput.value
    toast.success('Note updated')
    closeNoteModal()
  } catch (error) {
    console.error(error)
    toast.error('Failed to update note')
  }
}

const handleWebSocketMessage = (message: any) => {
  if (message.type === 'BEACON_CHECKIN') {
    const beacon = beacons.value.find((b: any) => b.BeaconID === message.payload.beacon_id)
    if (beacon) {
      beacon.LastSeen = message.payload.last_seen
    }
  } else if (message.type === 'BEACON_METADATA_UPDATED') {
    const index = beacons.value.findIndex((b: any) => b.BeaconID === message.payload.BeaconID)
    if (index !== -1) {
      // Update existing beacon
      beacons.value[index] = { ...beacons.value[index], ...message.payload }
    } else {
      // Add if not exists (unlikely for metadata update but safe)
    }
  } else if (message.type === 'BEACON_NEW') {
    beacons.value.push(message.payload)
  } else if (message.type === 'BEACON_DELETED') {
    beacons.value = beacons.value.filter((b: any) => b.BeaconID !== message.payload.BeaconID)
  }
}

onMounted(() => {
  fetchBeacons()
  webSocketService.addMessageHandler(handleWebSocketMessage)
  timer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  webSocketService.removeMessageHandler(handleWebSocketMessage)
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.beacons-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-lg);
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.status-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--color-text-secondary);
  margin-right: 6px;
}

.status-dot.online {
  background-color: var(--color-success);
  box-shadow: 0 0 6px var(--color-success);
}

.mono-font {
  font-family: 'Fira Code', monospace;
  font-size: 0.9em;
}

.note-cell {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background-color 0.2s;
}

.note-cell:hover {
  background-color: var(--color-bg-secondary);
}

.note-text {
  color: var(--color-text-primary);
}

.note-placeholder {
  color: var(--color-text-secondary);
  font-style: italic;
  font-size: 0.9em;
}

.edit-icon {
  opacity: 0;
  transition: opacity 0.2s;
  color: var(--color-primary);
}

.note-cell:hover .edit-icon {
  opacity: 1;
}

/* Modal Styles */
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

.modal-desc {
  color: var(--color-text-secondary);
  font-size: 0.9rem;
  margin-bottom: var(--spacing-md);
}

.form-group {
  margin: var(--spacing-md) 0;
}

.full-width-input {
  width: 100%;
  padding: 8px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  color: var(--color-text-primary);
  border-radius: var(--radius-sm);
  outline: none;
}

.full-width-input:focus {
  border-color: var(--color-primary);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
}
</style>
