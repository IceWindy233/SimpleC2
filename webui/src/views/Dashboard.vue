<template>
  <div class="dashboard">
    <div class="header-section">
      <h1>Dashboard</h1>
      <Button variant="primary" size="sm" @click="refreshData">
        Refresh Data
      </Button>
    </div>

    <div class="stats-grid">
      <Card title="Active Listeners">
        <div class="stat-value">{{ stats.listeners }}</div>
        <div class="stat-label">Running on 2 ports</div>
      </Card>
      <Card title="Active Beacons">
        <div class="stat-value">{{ stats.beacons }}</div>
        <div class="stat-label">3 check-ins in last 5m</div>
      </Card>
      <Card title="Tasks Pending">
        <div class="stat-value">{{ stats.tasks }}</div>
        <div class="stat-label">Waiting for pickup</div>
      </Card>
      <Card title="System Status">
        <div class="stat-value text-success">Online</div>
        <div class="stat-label">Uptime: 2d 4h</div>
      </Card>
    </div>

    <div class="recent-activity">
      <Card title="Recent Activity">
        <Table :columns="activityColumns" :data="recentActivity" />
      </Card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Card from '../components/ui/Card.vue'
import Button from '../components/ui/Button.vue'
import Table from '../components/ui/Table.vue'
import { useToastStore } from '../stores/toast'
import api from '../services/api'

const toast = useToastStore()

const stats = ref({
  listeners: 0,
  beacons: 0,
  tasks: 0
})

const activityColumns = [
  { key: 'time', label: 'Time', width: '150px' },
  { key: 'type', label: 'Type', width: '100px' },
  { key: 'description', label: 'Description' }
]

const recentActivity = ref<any[]>([])

const fetchData = async () => {
  try {
    const [listenersRes, beaconsRes] = await Promise.all([
      api.get('/listeners'),
      api.get('/beacons')
    ])
    
    stats.value.listeners = listenersRes.data.meta.total
    stats.value.beacons = beaconsRes.data.meta.total
    // Tasks stat would need a separate endpoint or calculation
    
  } catch (error) {
    console.error('Failed to fetch dashboard data', error)
  }
}

const refreshData = async () => {
  toast.info('Refreshing dashboard data...')
  await fetchData()
  toast.success('Dashboard updated')
}

onMounted(() => {
  fetchData()
})
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-xl);
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: var(--spacing-lg);
}

.stat-value {
  font-size: 2.5rem;
  font-weight: 700;
  color: var(--color-primary);
  line-height: 1.2;
}

.stat-label {
  color: var(--color-text-secondary);
  font-size: var(--font-size-sm);
  margin-top: var(--spacing-xs);
}

.text-success {
  color: var(--color-success);
}
</style>
