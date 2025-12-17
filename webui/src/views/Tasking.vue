<template>
  <div class="tasking-view">
    <div class="header-section">
      <div class="title-group">
        <Button variant="ghost" size="sm" @click="$router.push('/beacons')">← Back</Button>
        <h1>Tasking: {{ beaconId }}</h1>
      </div>
      <div class="header-actions">
        <div class="tabs">
          <button 
            :class="['tab-btn', { active: activeTab === 'console' }]" 
            @click="activeTab = 'console'"
          >
            Console
          </button>
          <button 
            :class="['tab-btn', { active: activeTab === 'processes' }]" 
            @click="activeTab = 'processes'"
          >
            Processes
          </button>
          <button 
            :class="['tab-btn', { active: activeTab === 'files' }]" 
            @click="activeTab = 'files'"
          >
            Files
          </button>
          <button 
            :class="['tab-btn', { active: activeTab === 'tunnels' }]" 
            @click="activeTab = 'tunnels'"
          >
            Tunnels
          </button>
        </div>
        <div class="status-badge">
          <span class="dot" :class="{ online: beacon?.Status === 'active' }"></span> {{ beacon?.Status === 'active' ? 'Online' : 'Offline' }}
        </div>
        <Button variant="warning" size="sm" @click="showShellcodeModal = true">Inject Shellcode</Button>
      </div>
    </div>

    <div class="tasking-layout">
      <div class="main-area">
        <div v-show="activeTab === 'console'" class="console-area">
          <Card class="console-card">
            <div class="console-output" ref="consoleOutput">
              <div v-for="log in consoleLogs" :key="log.id" class="log-entry">
                <div class="log-meta">
                  <span class="log-time">{{ log.time }}</span>
                  <span :class="['log-type', `type-${log.type}`]">{{ log.type }}</span>
                </div>
                
                <!-- Interactive Output for specific commands -->
                <div v-if="log.type === 'output' && log.command === 'upload'" class="log-interactive">
                  <div class="file-download">
                    <span>File uploaded to server: {{ log.content }}</span>
                    <Button variant="primary" size="sm" @click="downloadLoot(log.content)">Download File</Button>
                  </div>
                </div>
                <!-- 截图输出：显示为内嵌图片 -->
                <div v-else-if="log.type === 'output' && log.command === 'screenshot'" class="log-screenshot">
                  <img v-if="screenshotUrls[log.id]" :src="screenshotUrls[log.id]" alt="Screenshot" class="screenshot-img" @click="openScreenshotBlob(log.id)" />
                  <div v-else class="screenshot-loading">Loading screenshot...</div>
                </div>
                <pre v-else class="log-content">{{ log.content }}</pre>
              </div>
            </div>
            <div class="console-input">
              <div class="prompt">beacon></div>
              <input 
                v-model="command" 
                @keyup.enter="sendConsoleCommand"
                type="text" 
                placeholder="Enter command..." 
                autofocus
              />
              <Button variant="primary" size="sm" @click="sendConsoleCommand" :disabled="!command">Send</Button>
            </div>
          </Card>
        </div>
        
        <div v-if="activeTab === 'processes'" class="processes-area">
          <Card class="processes-card">
            <ProcessBrowser :beacon-id="beaconId" :logs="logs" @send-command="handleChildCommand" />
          </Card>
        </div>

        <div v-if="activeTab === 'files'" class="files-area">
          <Card class="files-card">
            <FileBrowser :beacon-id="beaconId" />
          </Card>
        </div>

        <div v-if="activeTab === 'tunnels'" class="tunnels-area">
          <Card class="tunnels-card">
            <TunnelManager :beacon-id="beaconId" />
          </Card>
        </div>
      </div>

      <div class="info-area">
        <Card title="Beacon Info">
          <div class="info-grid">
            <div class="info-item full-width">
              <label>Hostname</label>
              <div class="value-row">
                <span>{{ beacon?.Hostname || 'Loading...' }}</span>
                <span class="status-indicator" :class="{ online: beacon?.Status === 'active' }"></span>
              </div>
            </div>
            <div class="info-item">
              <label>User</label>
              <span>{{ beacon?.Username || '-' }}</span>
            </div>
            <div class="info-item">
              <label>PID</label>
              <span>{{ beacon?.PID || '-' }}</span>
            </div>
            <div class="info-item">
              <label>OS</label>
              <span>{{ beacon?.OS || '-' }}</span>
            </div>
            <div class="info-item">
              <label>Internal IP</label>
              <span>{{ beacon?.InternalIP || '-' }}</span>
            </div>
            <div class="info-item">
              <label>Sleep</label>
              <span>{{ beacon?.Sleep || 0 }}s</span>
            </div>
            <div class="info-item">
              <label>Last Check-in</label>
              <span>{{ beacon ? new Date(beacon.LastSeen).toLocaleTimeString() : '-' }}</span>
            </div>
            <div class="info-item full-width">
               <Button variant="danger" size="sm" @click="deleteBeacon" block>Delete Beacon</Button>
            </div>
          </div>
        </Card>
      </div>
    </div>

    <!-- Shellcode Injection Modal -->
    <div v-if="showShellcodeModal" class="modal-overlay">
      <div class="modal">
        <h3>Inject Shellcode</h3>
        <p class="modal-desc">Select a raw shellcode file (.bin) to inject into the beacon process.</p>
        <div class="form-group">
          <input type="file" @change="handleShellcodeFileSelect" />
        </div>
        <div v-if="shellcodeFile" class="file-info">
          Selected: {{ shellcodeFile.name }} ({{ shellcodeFile.size }} bytes)
        </div>
        <div class="modal-actions">
          <Button variant="ghost" @click="showShellcodeModal = false">Cancel</Button>
          <Button variant="danger" @click="injectShellcode" :disabled="!shellcodeFile">Inject</Button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import Card from '../components/ui/Card.vue'
import Button from '../components/ui/Button.vue'
import { useToastStore } from '../stores/toast'
import api from '../services/api'
import { webSocketService } from '../services/websocket'
import FileBrowser from '../components/FileBrowser.vue'
import ProcessBrowser from '../components/ProcessBrowser.vue'
import TunnelManager from '../components/TunnelManager.vue'

const route = useRoute()
const router = useRouter()
const toast = useToastStore()
const beaconId = route.params.id as string
const activeTab = ref('console')

const command = ref('')
const consoleOutput = ref<HTMLElement | null>(null)
const beacon = ref<any>(null)
const logs = ref<any[]>([])
const screenshotUrls = ref<Record<string, string>>({}) // 存储已加载的截图 blob URL

// Computed property to filter logs for the console view
const consoleLogs = computed(() => {
  return logs.value.filter(log => {
    // 1. Filter out specific internal commands regardless of source (safety net)
    if (['ps', 'browse'].includes(log.command)) return false
    
    // 2. Filter by source: Only show logs from 'console' or undefined (legacy/unknown)
    // If explicit source is provided and it's NOT console, hide it.
    if (log.source && log.source !== 'console') return false
    
    return true
  })
})

const showShellcodeModal = ref(false)
const shellcodeFile = ref<File | null>(null)

const fetchBeacon = async () => {
  try {
    const response = await api.get(`/beacons/${beaconId}`)
    beacon.value = response.data.data || {}
  } catch (error) {
    console.error(error)
    toast.error('Failed to load beacon info')
  }
}

const fetchTasks = async () => {
  try {
    const response = await api.get(`/beacons/${beaconId}/tasks`)
    const tasks = response.data.data || []
    
    // Transform tasks to logs
    logs.value = tasks.flatMap((task: any) => {
      const entries = []
      // Input log
      const createdAt = new Date(task.CreatedAt)
      entries.push({
        id: task.TaskID + '_in',
        taskId: task.TaskID,
        command: task.Command,
        timestamp: createdAt.getTime(),
        time: createdAt.toLocaleTimeString(),
        type: 'input',
        content: `${task.Command} ${task.Arguments || ''}`,
        source: task.Source
      })
      
      // Output log (if completed)
      if (task.Status === 'completed' && task.Output) {
        const updatedAt = new Date(task.UpdatedAt)
        let content = task.Output
        
        entries.push({
          id: task.TaskID + '_out',
          taskId: task.TaskID,
          command: task.Command,
          timestamp: updatedAt.getTime(),
          time: updatedAt.toLocaleTimeString(),
          type: 'output',
          content: content,
          source: task.Source
        })
      }
      return entries
    }).sort((a: any, b: any) => a.timestamp - b.timestamp)
    
    scrollToBottom()
  } catch (error) {
    console.error(error)
  }
}

const downloadLoot = async (filename: string) => {
  try {
    const response = await api.get(`/loot/${filename}`, { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    
    const contentDisposition = response.headers['content-disposition']
    let downloadName = filename
    
    // Strip the task ID prefix if present (e.g., "UUID_filename.ext" -> "filename.ext")
    const underscoreIndex = downloadName.indexOf('_');
    if (underscoreIndex !== -1 && underscoreIndex < 40) { // Assume UUID prefix is roughly 36 chars
        downloadName = downloadName.substring(underscoreIndex + 1);
    }

    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename="?([^"]+)"?/)
      if (filenameMatch && filenameMatch.length === 2) {
        downloadName = filenameMatch[1]
      }
    }
    
    link.setAttribute('download', downloadName)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch (e) {
    console.error(e)
    toast.error('Download failed')
  }
}

const sendConsoleCommand = () => {
  if (!command.value.trim()) return
  const cmdParts = command.value.trim().split(' ')
  const cmd = cmdParts[0] || ''
  const args = cmdParts.slice(1).join(' ') || ''
  submitTask(cmd, args, 'console')
  command.value = ''
}

const handleChildCommand = (fullCommand: string) => {
  const cmdParts = fullCommand.trim().split(' ')
  const cmd = cmdParts[0] || ''
  const args = cmdParts.slice(1).join(' ') || ''
  submitTask(cmd, args, 'ui')
}

const submitTask = async (cmd: string, args: string, source: string = 'console') => {
  // Add input log immediately for UX
  const now = new Date()
  logs.value.push({
    id: Date.now(),
    timestamp: now.getTime(),
    time: now.toLocaleTimeString(),
    type: 'input',
    command: cmd,
    content: `${cmd} ${args}`,
    source: source // Track source locally immediately
  })

  scrollToBottom()

  try {
    await api.post(`/beacons/${beaconId}/tasks`, {
      command: cmd,
      arguments: args,
      source: source
    })
  } catch (error: any) {
    toast.error('Failed to send command')
  }
}

const deleteBeacon = async () => {
  if (!confirm('Are you sure you want to delete this beacon? This will task it to exit and remove it from the active list.')) return
  try {
    await api.delete(`/beacons/${beaconId}`)
    toast.success('Beacon deleted and exit task queued')
    router.push('/beacons')
  } catch (error) {
    toast.error('Failed to delete beacon')
  }
}

// Shellcode Injection
const handleShellcodeFileSelect = (event: Event) => {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    shellcodeFile.value = target.files[0] || null
  }
}

const injectShellcode = async () => {
  if (!shellcodeFile.value) return

  const reader = new FileReader()
  reader.onload = async () => {
    // reader.result is ArrayBuffer
    const arrayBuffer = reader.result as ArrayBuffer
    const bytes = new Uint8Array(arrayBuffer)
    
    // Convert to Base64
    let binary = ''
    const len = bytes.byteLength
    for (let i = 0; i < len; i++) {
        binary += String.fromCharCode(bytes[i]!)
    }
    const base64String = window.btoa(binary)

    try {
        await submitTask('shellcode', base64String, 'ui')
        toast.success(`Queued shellcode injection (${len} bytes)`)
        showShellcodeModal.value = false
        shellcodeFile.value = null
    } catch (e) {
        toast.error('Failed to queue shellcode task')
    }
  }
  reader.readAsArrayBuffer(shellcodeFile.value)
}

const loadScreenshot = async (logId: string, path: string) => {
  // 避免重复加载
  if (screenshotUrls.value[logId]) return
  
  try {
    const response = await api.get(`/loot/${encodeURIComponent(path)}`, { responseType: 'blob' })
    const blobUrl = window.URL.createObjectURL(new Blob([response.data]))
    screenshotUrls.value[logId] = blobUrl
  } catch (error) {
    console.error('Failed to load screenshot:', error)
  }
}

const openScreenshotBlob = (logId: string) => {
  const blobUrl = screenshotUrls.value[logId]
  if (blobUrl) {
    // 创建一个包含图片的 HTML 页面
    const newWindow = window.open('', '_blank')
    if (newWindow) {
      newWindow.document.write(`
        <!DOCTYPE html>
        <html>
          <head>
            <title>Screenshot</title>
            <style>
              body {
                margin: 0;
                display: flex;
                justify-content: center;
                align-items: center;
                min-height: 100vh;
                background: #1e1e1e;
              }
              img {
                max-width: 100%;
                max-height: 100vh;
              }
            </style>
          </head>
          <body>
            <img src="${blobUrl}" alt="Screenshot" />
          </body>
        </html>
      `)
      newWindow.document.close()
    }
  }
}

const scrollToBottom = () => {
  nextTick(() => {
    if (consoleOutput.value) {
      consoleOutput.value.scrollTop = consoleOutput.value.scrollHeight
    }
  })
}

const handleWebSocketMessage = (message: any) => {
  console.log('WS Message:', message)
  if (message.type === 'TASK_OUTPUT' && message.payload.BeaconID === beaconId) {
    const task = message.payload
    const updatedAt = new Date(task.UpdatedAt || Date.now()) // Fallback if UpdatedAt is missing in event
    
    let content = task.Output
    logs.value.push({
      id: task.TaskID + '_out',
      taskId: task.TaskID,
      command: task.Command,
      timestamp: updatedAt.getTime(),
      time: updatedAt.toLocaleTimeString(),
      type: 'output',
      content: content,
      source: task.Source
    })
    scrollToBottom()
  } else if (message.type === 'BEACON_CHECKIN' && message.payload.beacon_id === beaconId) {
    if (beacon.value) {
      beacon.value.LastSeen = message.payload.last_seen
    }
  } else if (message.type === 'BEACON_METADATA_UPDATED' && message.payload.BeaconID === beaconId) {
    if (beacon.value) {
      // Update all properties
      Object.assign(beacon.value, message.payload)
    }
  }
}

// 监听 logs 变化，自动加载截图
watch(logs, (newLogs) => {
  newLogs.forEach((log: any) => {
    if (log.type === 'output' && log.command === 'screenshot' && log.content) {
      loadScreenshot(log.id, log.content)
    }
  })
}, { deep: true })

onMounted(() => {
  fetchBeacon()
  fetchTasks()
  scrollToBottom()
  webSocketService.addMessageHandler(handleWebSocketMessage)
})


onUnmounted(() => {
  webSocketService.removeMessageHandler(handleWebSocketMessage)
})
</script>

<style scoped>
.tasking-view {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-md);
  height: calc(100vh - 100px); /* Adjust based on header/padding */
}

.header-section {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-lg);
}

.action-buttons {
  display: flex;
  gap: var(--spacing-sm);
  margin-right: var(--spacing-md);
}

.tabs {
  display: flex;
  gap: var(--spacing-xs);
  background: var(--color-bg-secondary);
  padding: 4px;
  border-radius: var(--radius-md);
}

.tab-btn {
  padding: 6px 16px;
  border: none;
  background: transparent;
  color: var(--color-text-secondary);
  cursor: pointer;
  border-radius: var(--radius-sm);
  font-weight: 500;
  transition: all 0.2s;
}

.tab-btn.active {
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  box-shadow: var(--shadow-sm);
}

.title-group {
  display: flex;
  align-items: center;
  gap: var(--spacing-md);
}

.status-badge {
  display: flex;
  align-items: center;
  gap: var(--spacing-xs);
  color: var(--color-text-secondary); /* Default to offline color */
  font-weight: 600;
}

.dot {
  width: 8px;
  height: 8px;
  background-color: var(--color-text-secondary); /* Default offline */
  border-radius: 50%;
}

.dot.online {
  background-color: var(--color-success);
  box-shadow: 0 0 8px var(--color-success);
}

.tasking-layout {
  display: flex;
  gap: var(--spacing-lg);
  flex: 1;
  min-height: 0; /* Important for nested scrolling */
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.console-area, .files-area, .processes-area, .tunnels-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
}

.console-card, .files-card, .processes-card, .tunnels-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

/* Deep selector to target Card's body if needed, but Card structure allows direct styling if we put content in default slot */
:deep(.card-body) {
  display: flex;
  flex-direction: column;
  padding: 0;
  height: 100%;
}

.console-output {
  flex: 1;
  overflow-y: auto;
  padding: var(--spacing-md);
  background-color: #1e1e1e;
  color: #e0e0e0;
  font-family: 'Fira Code', monospace;
  font-size: 0.9rem;
}

.log-entry {
  margin-bottom: var(--spacing-sm);
}

.log-meta {
  display: flex;
  gap: var(--spacing-sm);
  font-size: 0.75rem;
  opacity: 0.7;
  margin-bottom: 2px;
}

.type-input { color: var(--color-primary); }
.type-output { color: var(--color-text-light); }
.type-system { color: var(--color-warning); }

.log-content {
  white-space: pre-wrap;
  margin: 0;
}

.log-screenshot {
  margin: var(--spacing-sm) 0;
}

.screenshot-loading {
  color: var(--color-text-secondary);
  font-style: italic;
  padding: var(--spacing-sm);
}

.screenshot-img {
  max-width: 100%;
  max-height: 400px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition: transform 0.2s, box-shadow 0.2s;
}

.screenshot-img:hover {
  transform: scale(1.02);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.console-input {
  display: flex;
  align-items: center;
  padding: var(--spacing-sm);
  background-color: var(--color-bg-secondary);
  border-top: 1px solid var(--color-border);
}

.prompt {
  font-family: monospace;
  margin-right: var(--spacing-sm);
  color: var(--color-primary);
  font-weight: bold;
}

.console-input input {
  flex: 1;
  border: none;
  background: transparent;
  padding: var(--spacing-sm);
  color: var(--color-text-primary);
  font-family: monospace;
  outline: none;
}

.info-area {
  width: 300px;
}

.info-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--spacing-md);
}

.info-item {
  display: flex;
  flex-direction: column;
  background: var(--color-bg-tertiary);
  padding: var(--spacing-md);
  border-radius: var(--radius-sm);
}

.info-item.full-width {
  grid-column: span 2;
}

.info-item label {
  font-size: 0.75rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
}

.info-item span {
  font-weight: 500;
  font-family: 'Fira Code', monospace;
  font-size: 0.9rem;
  word-break: break-all;
}

.value-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.status-indicator {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: var(--color-text-secondary); /* Default offline */
}

.status-indicator.online {
  background-color: var(--color-success);
  box-shadow: 0 0 8px var(--color-success);
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

.file-info {
  margin: var(--spacing-sm) 0;
  font-size: 0.85rem;
  color: var(--color-primary);
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-sm);
}
</style>
