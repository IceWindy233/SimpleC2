<template>
  <div class="file-browser">
    <div class="browser-toolbar">
      <div class="toolbar-left">
        <Button 
          variant="ghost" 
          size="sm" 
          @click="navigateUp" 
          :disabled="currentPath === '/' || loading"
        >
          ↑ 上一级
        </Button>
        <Input 
          v-model="currentPath" 
          @keyup.enter="listFiles" 
          placeholder="输入路径" 
          class="path-input"
          :disabled="loading"
        />
        <Button 
          variant="primary" 
          size="sm" 
          @click="listFiles" 
          :disabled="loading"
        >
          前往
        </Button>
      </div>
      <div class="toolbar-right">
        <Button 
          variant="secondary" 
          size="sm" 
          @click="triggerUpload" 
          :disabled="loading"
          class="mr-2"
        >
          上传
        </Button>
        <Button 
          variant="ghost" 
          size="sm" 
          @click="listFiles" 
          :disabled="loading"
        >
          刷新
        </Button>
      </div>
      <input 
        type="file" 
        ref="fileInput" 
        style="display: none" 
        @change="handleFileUpload"
      />
    </div>

    <div class="file-list-container">
      <Table 
        :columns="columns" 
        :data="sortedFiles" 
        :loading="loading"
        :sort-key="sortKey"
        :sort-order="sortOrder"
        :selected-row="selectedRow"
        @row-click="handleRowClick"
        @row-dblclick="handleRowDblClick"
        @sort="handleSort"
      >
        <template #name="{ row }">
          <div class="file-name">
            <span class="icon">{{ row.isDir ? '📁' : '📄' }}</span>
            {{ row.name }}
          </div>
        </template>
        <template #size="{ row }">
          {{ formatSize(row.size) }}
        </template>
        <template #modTime="{ row }">
          {{ row.modTime || '-' }}
        </template>
        <template #actions="{ row }">
          <div class="row-actions">
            <Button variant="outline" size="sm" v-if="!row.isDir" @click.stop="downloadFile(row)">下载</Button>
            <Button variant="danger" size="sm" @click.stop="deleteFile(row)">删除</Button>
          </div>
        </template>
        <template #empty>
          <div class="empty-state">
            <div class="empty-icon">📂</div>
            <div class="empty-text">目录为空</div>
            <div class="empty-hint">双击文件夹进入，或上传文件</div>
          </div>
        </template>
      </Table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import Button from './ui/Button.vue'
import Input from './ui/Input.vue'
import Table from './ui/Table.vue'
import { useToastStore } from '../stores/toast'
import api from '../services/api'
import { webSocketService } from '../services/websocket'

const props = defineProps<{
  beaconId: string
}>()

const toast = useToastStore()
const currentPath = ref('/')
const loading = ref(false)
const files = ref<any[]>([])
const fileInput = ref<HTMLInputElement | null>(null)
const selectedRow = ref<any>(null)
const sortKey = ref('name')
const sortOrder = ref<'asc' | 'desc'>('asc')
const loadingTimeout = ref<any>(null)

const columns = [
  { key: 'name', label: 'Name', width: '50%', sortable: true },
  { key: 'size', label: 'Size', width: '15%', sortable: true },
  { key: 'modTime', label: 'Modified', width: '20%', sortable: true },
  { key: 'actions', label: 'Actions', width: '15%' }
]

// Parse output from 'browse' command
// Format: First line is current path, rest is JSON array of files
const parseBrowseOutput = (output: string) => {
  const lines = output.trim().split('\n')
  if (lines.length === 0) return []

  const firstLine = lines[0]
  if (!firstLine) return []
  const path = firstLine.trim()
  // Update current path if it changed (e.g. after cd)
  if (path && path !== currentPath.value) {
    currentPath.value = path
  }

  const jsonFiles = lines.slice(1).join('\n')
  try {
    const files = JSON.parse(jsonFiles)
    return files.map((f: any) => ({
      name: f.name,
      isDir: f.is_dir,
      size: f.size,
      modTime: f.lastModified,
      permissions: f.permissions
    }))
  } catch (e) {
    console.error("Failed to parse JSON file list:", e)
    return []
  }
}

const listFiles = async () => {
  loading.value = true
  files.value = [] // Clear current list
  
  // Clear any existing timeout
  if (loadingTimeout.value) {
    clearTimeout(loadingTimeout.value)
  }
  
  // Set a timeout to stop loading after 30 seconds
  loadingTimeout.value = setTimeout(() => {
    if (loading.value) {
      loading.value = false
      toast.error('请求超时，beacon 可能未响应')
    }
  }, 30000)
  
  try {
    // Send browse command
    await api.post(`/beacons/${props.beaconId}/tasks`, {
      command: 'browse',
      arguments: currentPath.value
    })
  } catch (error) {
    console.error(error)
    toast.error('请求文件列表失败')
    loading.value = false
    if (loadingTimeout.value) {
      clearTimeout(loadingTimeout.value)
    }
  }
}

const navigateUp = () => {
  // Simple path manipulation
  const parts = currentPath.value.split('/').filter(p => p)
  parts.pop()
  currentPath.value = '/' + parts.join('/') || '/'
  // Ensure we don't end up with //
  if (currentPath.value.length > 1 && currentPath.value.endsWith('/')) {
    currentPath.value = currentPath.value.slice(0, -1)
  }
  listFiles()
}

const handleRowClick = (row: any) => {
  selectedRow.value = row
}

const handleRowDblClick = (row: any) => {
  if (row.isDir) {
    currentPath.value = currentPath.value.endsWith('/') 
      ? currentPath.value + row.name 
      : currentPath.value + '/' + row.name
    listFiles()
  }
}

const handleSort = (key: string) => {
  if (sortKey.value === key) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortKey.value = key
    sortOrder.value = 'asc'
  }
}

import { computed } from 'vue'

const sortedFiles = computed(() => {
  return [...files.value].sort((a, b) => {
    let modifier = sortOrder.value === 'asc' ? 1 : -1
    
    // Always keep directories on top
    if (a.isDir !== b.isDir) {
      return a.isDir ? -1 : 1
    }
    
    if (sortKey.value === 'size') {
      return (a.size - b.size) * modifier
    } else if (sortKey.value === 'modTime') {
      return a.modTime.localeCompare(b.modTime) * modifier
    } else {
      return a.name.localeCompare(b.name) * modifier
    }
  })
})

const triggerUpload = () => {
  fileInput.value?.click()
}

const handleFileUpload = async (event: Event) => {
  const target = event.target as HTMLInputElement
  if (!target.files?.length) return
  
  const file = target.files[0]
  if (file) {
    try {
      toast.info(`正在上传 ${file.name} 到服务器...`)
      
      // 1. Init upload
      const initRes = await api.post('/upload/init', { filename: file.name })
      const uploadId = initRes.data.data.upload_id
      
      // 2. Upload chunks
      const chunkSize = 1024 * 1024 // 1MB chunks
      const totalChunks = Math.ceil(file.size / chunkSize)
      
      for (let i = 0; i < totalChunks; i++) {
        const start = i * chunkSize
        const end = Math.min(file.size, start + chunkSize)
        const chunk = file.slice(start, end)
        
        await api.post('/upload/chunk', chunk, {
          headers: {
            'Content-Type': 'application/octet-stream',
            'X-Upload-ID': uploadId,
            'X-Chunk-Number': (i + 1).toString()
          }
        })
      }
      
      // 3. Complete upload
      const completeRes = await api.post('/upload/complete', {
        upload_id: uploadId,
        filename: file.name
      })
      const serverFilePath = completeRes.data.data.filepath
      
      toast.success('文件已上传到服务器，正在发送下发任务...')
      
      // 4. Create task for Beacon to download (Operator Upload)
      // Destination path construction
      let destPath = file.name
      if (currentPath.value && currentPath.value !== '/') {
        destPath = currentPath.value.endsWith('/') 
          ? currentPath.value + file.name 
          : currentPath.value + '/' + file.name
      }

      const downloadArgs = {
        source: serverFilePath,
        destination: destPath,
        file_size: file.size,
        chunk_size: chunkSize
      }
      
      await api.post(`/beacons/${props.beaconId}/tasks`, {
        command: 'download',
        arguments: JSON.stringify(downloadArgs)
      })
      
      toast.success('下发任务已发送')
      
    } catch (error) {
      console.error(error)
      toast.error('上传失败')
    } finally {
      // Reset input
      if (fileInput.value) {
        fileInput.value.value = ''
      }
    }
  }
}

const downloadLoot = async (filename: string) => {
  try {
    const response = await api.get(`/loot/${encodeURIComponent(filename)}`, { responseType: 'blob' })
    const url = window.URL.createObjectURL(new Blob([response.data]))
    const link = document.createElement('a')
    link.href = url
    
    // Get filename from Content-Disposition header or use the original filename
    const contentDisposition = response.headers['content-disposition']
    let downloadName = filename
    
    if (contentDisposition) {
      // Handle RFC 5987 encoded filename (filename*=UTF-8''...)
      const rfc5987Match = contentDisposition.match(/filename\*=UTF-8''([^;\s]+)/)
      if (rfc5987Match && rfc5987Match[1]) {
        downloadName = decodeURIComponent(rfc5987Match[1])
      } else {
        // Fallback to standard filename
        const filenameMatch = contentDisposition.match(/filename="?([^"]+)"?/)
        if (filenameMatch && filenameMatch[1]) {
          downloadName = filenameMatch[1]
        }
      }
    }
    
    link.setAttribute('download', downloadName)
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch (e) {
    console.error(e)
    toast.error('下载文件失败')
  }
}

const downloadFile = async (row: any) => {
  toast.info(`正在请求下载 ${row.name}...`)
  try {
    const fullPath = currentPath.value.endsWith('/') 
      ? currentPath.value + row.name 
      : currentPath.value + '/' + row.name
    
    // Command 'upload' tells Beacon to upload file to Teamserver (Operator Download)
    // Arguments should be the file path string, not JSON
    await api.post(`/beacons/${props.beaconId}/tasks`, {
      command: 'upload',
      arguments: fullPath
    })
    toast.success('下载任务已发送')
  } catch (error) {
    toast.error('请求下载失败')
  }
}

const deleteFile = async (row: any) => {
  if (!confirm(`确定要删除 ${row.name} 吗？此操作不可逆。`)) return

  toast.info(`正在请求删除 ${row.name}...`)
  try {
    const fullPath = currentPath.value.endsWith('/') 
      ? currentPath.value + row.name 
      : currentPath.value + '/' + row.name
    
    await api.post(`/beacons/${props.beaconId}/tasks`, {
      command: 'rm',
      arguments: fullPath
    })
    toast.success('删除任务已发送')
  } catch (error) {
    console.error(error)
    toast.error('请求删除失败')
  }
}

const formatSize = (bytes: number) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

// WebSocket handler for browse output
const handleWebSocketMessage = (message: any) => {
  try {
    if (message.type === 'TASK_OUTPUT' || message.type === 'FILE_UPLOAD_COMPLETED') {
      // Handle generic TASK_OUTPUT first (legacy/standard path)
      if (message.type === 'TASK_OUTPUT' && message.payload.BeaconID === props.beaconId) {
        const task = message.payload
        if (task.Command === 'browse') {
          // Only process if task is completed or has output
          if (task.Status === 'completed' || task.Output) {
            console.log('Parsing browse output...', 'Output length:', task.Output?.length)
            try {
              const newFiles = parseBrowseOutput(task.Output || '')
              files.value = newFiles
              
              // Clear timeout and loading
              if (loadingTimeout.value) {
                clearTimeout(loadingTimeout.value)
                loadingTimeout.value = null
              }
              loading.value = false
              
              if (newFiles.length === 0) {
                toast.info('目录为空')
              } else {
                console.log('Loaded', newFiles.length, 'files')
              }
            } catch (parseError) {
              console.error('Failed to parse browse output:', parseError)
              console.error('Output was:', task.Output)
              if (loadingTimeout.value) {
                clearTimeout(loadingTimeout.value)
                loadingTimeout.value = null
              }
              loading.value = false
              toast.error('解析文件列表失败')
            }
          } else if (task.Status === 'failed') {
            console.log('Browse task failed')
            if (loadingTimeout.value) {
              clearTimeout(loadingTimeout.value)
              loadingTimeout.value = null
            }
            loading.value = false
            toast.error('浏览目录失败')
          }
        } else if (task.Command === 'upload' && task.Status === 'completed') {
           toast.success('文件已下载到服务器 (Loot)')
           if (task.Output) {
             downloadLoot(task.Output)
           }
        } else if (task.Command === 'download' && task.Status === 'completed') {
           toast.success('文件已成功下发到目标')
        } else if (task.Command === 'rm' && task.Status === 'completed') {
           toast.success('文件删除成功')
           listFiles() // Refresh the list
        }
      }
      
      // Also handle specialized FILE_UPLOAD_COMPLETED event if it exists
      if (message.type === 'FILE_UPLOAD_COMPLETED' && message.payload.beacon_id === props.beaconId) {
          // This event ensures we catch the completion even if TASK_OUTPUT logic changes
          // Check if we already downloaded via TASK_OUTPUT to avoid double download
          // But currently TASK_OUTPUT is the primary driver.
      }
    }
  } catch (error) {
    console.error('Error in handleWebSocketMessage:', error)
  }
}

onMounted(() => {
  webSocketService.addMessageHandler(handleWebSocketMessage)
  // Initial list
  listFiles()
})

import { onUnmounted } from 'vue'
onUnmounted(() => {
  webSocketService.removeMessageHandler(handleWebSocketMessage)
})
</script>

<style scoped>
.file-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.browser-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: var(--spacing-md);
  padding-top: var(--spacing-sm);
  padding-bottom: var(--spacing-sm);
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex: 1;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  flex-shrink: 0;
}

.path-input {
  flex: 1;
  min-width: 200px;
}

.file-list-container {
  flex: 1;
  overflow: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-secondary);
  min-height: 0; /* Important for flex children to scroll properly */
}

.file-name {
  display: flex;
  align-items: center;
  gap: var(--spacing-sm);
  cursor: pointer;
  user-select: none;
}

.icon {
  font-size: 1.2em;
  line-height: 1;
}

.row-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--spacing-sm);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-xl);
  color: var(--color-text-secondary);
  text-align: center;
  min-height: 300px;
}

.empty-icon {
  font-size: 3rem;
  margin-bottom: var(--spacing-md);
  opacity: 0.5;
}

.empty-text {
  font-size: var(--font-size-lg);
  font-weight: 500;
  margin-bottom: var(--spacing-xs);
  color: var(--color-text-primary);
}

.empty-hint {
  font-size: var(--font-size-sm);
  opacity: 0.7;
}
</style>

