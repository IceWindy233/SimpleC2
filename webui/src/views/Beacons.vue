<template>
  <div class="beacons-view">
    <div class="header-section">
      <h1>Beacons</h1>
    </div>

    <Card>
      <Table :columns="columns" :data="beacons" :loading="loading">
        <template #Status="{ value }">
          <span class="status-dot" :class="{ online: value === 'active' }"></span>
          {{ value === 'active' ? 'Online' : 'Offline' }}
        </template>
        <template #lastCheckin="{ value }">
          {{ formatTime(value) }}
        </template>
        <template #actions="{ row }">
          <Button variant="primary" size="sm" @click="interact(row)">Interact</Button>
        </template>
      </Table>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Card from '../components/ui/Card.vue'
import Table from '../components/ui/Table.vue'
import Button from '../components/ui/Button.vue'
import api from '../services/api'

const router = useRouter()
const loading = ref(false)

const columns = [
  { key: 'BeaconID', label: 'ID', width: '80px' },
  { key: 'Status', label: 'Status', width: '100px' },
  { key: 'Hostname', label: 'Hostname' },
  { key: 'Username', label: 'User' },
  { key: 'OS', label: 'OS' },
  { key: 'LastSeen', label: 'Last Check-in' },
  { key: 'actions', label: 'Actions', width: '100px' }
]

const beacons = ref<any[]>([])

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

const formatTime = (timestamp: number) => {
  // Backend returns unix timestamp in seconds, JS needs milliseconds
  const date = new Date(timestamp * 1000)
  const diff = Date.now() - date.getTime()
  if (diff < 60000) return 'Just now'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}m ago`
  return date.toLocaleTimeString()
}

const interact = (row: any) => {
  router.push(`/beacons/${row.BeaconID}`)
}

import { webSocketService } from '../services/websocket'

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
      // Or add if not exists? Usually metadata update implies existence.
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
})

import { onUnmounted } from 'vue'
onUnmounted(() => {
  webSocketService.removeMessageHandler(handleWebSocketMessage)
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
</style>
