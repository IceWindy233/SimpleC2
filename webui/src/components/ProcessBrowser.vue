<template>
  <div class="process-browser">
    <div class="toolbar">
      <Button variant="primary" size="sm" @click="refreshProcesses" :disabled="loading">
        {{ loading ? 'Refreshing...' : 'Refresh Processes' }}
      </Button>
      <div class="search">
        <input v-model="searchQuery" placeholder="Filter processes..." />
      </div>
    </div>

    <div class="table-container">
      <table class="process-table">
        <thead>
          <tr>
            <th @click="sortBy('pid')">PID</th>
            <th @click="sortBy('name')">Name</th>
            <th @click="sortBy('user')">User</th>
            <th @click="sortBy('arch')">Arch</th>
            <th @click="sortBy('session')">Session</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="processes.length === 0">
            <td colspan="6" class="empty-state">
              {{ hasRunPs ? 'No processes found.' : 'Click Refresh to list processes.' }}
            </td>
          </tr>
          <tr v-for="proc in filteredProcesses" :key="processKey(proc)">
            <td>{{ proc.pid }}</td>
            <td>{{ proc.name }}</td>
            <td>{{ proc.user }}</td>
            <td>{{ proc.arch || '-' }}</td>
            <td>{{ proc.session || '-' }}</td>
            <td>
              <Button variant="danger" size="sm" @click="killProcess(proc.pid)">Kill</Button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import Button from './ui/Button.vue'

const props = defineProps<{
  beaconId: string
  logs: any[]
}>()

const emit = defineEmits(['sendCommand'])

const loading = ref(false)
const searchQuery = ref('')
const processes = ref<any[]>([])
const hasRunPs = ref(false)
const sortKey = ref('pid')
const sortDesc = ref(false)

const refreshProcesses = async () => {
  loading.value = true
  emit('sendCommand', 'ps')
  // We don't await the result here, we watch logs
  setTimeout(() => { loading.value = false }, 2000) // Reset loading state after a bit
}

const killProcess = (pid: number) => {
  if (confirm(`Are you sure you want to kill process ${pid}?`)) {
    emit('sendCommand', `kill ${pid}`)
  }
}

const processKey = (proc: any) => `${proc.pid}-${proc.name}`

const sortBy = (key: string) => {
  if (sortKey.value === key) {
    sortDesc.value = !sortDesc.value
  } else {
    sortKey.value = key
    sortDesc.value = false
  }
}

// Parse logs to find the latest 'ps' output
watch(() => props.logs, (newLogs) => {
  // Find the last output log for command 'ps'
  const psLog = [...newLogs].reverse().find(log => 
    log.type === 'output' && log.command === 'ps'
  )

  if (psLog) {
    try {
      const parsed = JSON.parse(psLog.content)
      if (Array.isArray(parsed)) {
        processes.value = parsed
        hasRunPs.value = true
        loading.value = false
      }
    } catch (e) {
      // Not JSON or parse error, ignore
    }
  }
}, { deep: true, immediate: true })

const filteredProcesses = computed(() => {
  let result = processes.value.filter(p => {
    const q = searchQuery.value.toLowerCase()
    return (
      p.name.toLowerCase().includes(q) ||
      String(p.pid).includes(q) ||
      (p.user && p.user.toLowerCase().includes(q))
    )
  })

  return result.sort((a, b) => {
    const valA = a[sortKey.value]
    const valB = b[sortKey.value]
    if (valA < valB) return sortDesc.value ? 1 : -1
    if (valA > valB) return sortDesc.value ? -1 : 1
    return 0
  })
})
</script>

<style scoped>
.process-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
  gap: var(--spacing-md);
}

.toolbar {
  display: flex;
  justify-content: space-between;
}

.search input {
  padding: 6px 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  background: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  outline: none;
}

.table-container {
  flex: 1;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}

.process-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.process-table th, .process-table td {
  padding: 8px 12px;
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.process-table th {
  background: var(--color-bg-secondary);
  font-weight: 600;
  cursor: pointer;
  user-select: none;
}

.process-table th:hover {
  background: var(--color-bg-tertiary);
}

.empty-state {
  text-align: center;
  padding: 24px;
  color: var(--color-text-secondary);
}
</style>
