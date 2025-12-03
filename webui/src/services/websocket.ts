import { useToastStore } from '../stores/toast'
import { useAuthStore } from '../stores/auth'

class WebSocketService {
    private ws: WebSocket | null = null
    private reconnectInterval: number = 3000
    private shouldReconnect: boolean = true
    private messageHandlers: ((message: any) => void)[] = []

    addMessageHandler(handler: (message: any) => void) {
        this.messageHandlers.push(handler)
    }

    removeMessageHandler(handler: (message: any) => void) {
        this.messageHandlers = this.messageHandlers.filter(h => h !== handler)
    }

    connect() {
        const authStore = useAuthStore()
        const token = authStore.token

        if (!token) {
            console.warn('Cannot connect to WebSocket: No token available')
            return
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        // Use relative path for proxy to handle, or explicit localhost:8080 if needed
        // Since we set up proxy, we can try relative path but WS proxying might need explicit config
        // Let's try direct connection to backend port for now as proxying WS can be tricky with simple config
        const wsUrl = `${protocol}//localhost:8080/api/ws?token=${token}`

        this.ws = new WebSocket(wsUrl)

        this.ws.onopen = () => {
            console.log('WebSocket connected')
        }

        this.ws.onmessage = (event) => {
            if (typeof event.data === 'string') {
                const lines = event.data.split('\n');
                for (const line of lines) {
                    const trimmedLine = line.trim();
                    if (!trimmedLine) continue;
                    
                    try {
                        const message = JSON.parse(trimmedLine);
                        this.handleMessage(message);
                    } catch (e) {
                        console.error('Failed to parse WebSocket message line:', e);
                        console.error('Raw line data:', trimmedLine);
                    }
                }
            } else {
                 // Handle binary data if necessary, though currently we expect text
                 try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                 } catch (e) {
                    console.error('Failed to parse WebSocket non-string message:', e);
                 }
            }
        }

        this.ws.onclose = () => {
            console.log('WebSocket disconnected')
            if (this.shouldReconnect) {
                setTimeout(() => this.connect(), this.reconnectInterval)
            }
        }

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error)
            this.ws?.close()
        }
    }

    disconnect() {
        this.shouldReconnect = false
        if (this.ws) {
            this.ws.close()
        }
    }

    private handleMessage(message: any) {
        const toast = useToastStore()

        // Notify all registered handlers
        this.messageHandlers.forEach(handler => handler(message))

        switch (message.type) {
            case 'BEACON_DELETED':
                toast.info(`Beacon deleted: ${message.payload.Hostname || message.payload.ID}`)
                // Trigger store update if needed
                break
            case 'LISTENER_STARTED':
                toast.success(`Listener started: ${message.payload.Name}`)
                break
            case 'LISTENER_STOPPED':
                toast.warning(`Listener stopped: ${message.payload.Name}`)
                break
            case 'TASK_QUEUED':
                toast.info(`Task queued for beacon ${message.payload.BeaconID}`)
                break
            case 'TASK_CANCELED':
                toast.warning(`Task ${message.payload.TaskID} canceled`)
                break
            case 'CLIENT_AUTHENTICATED':
                // toast.info(`User ${message.payload.username} authenticated`)
                break
            case 'TASK_OUTPUT':
            case 'TASK_DISPATCHED':
            case 'BEACON_CHECKIN':
            case 'BEACON_METADATA_UPDATED':
            case 'BEACON_NEW':
            case 'FILE_DOWNLOAD_STARTED':
            case 'FILE_UPLOAD_COMPLETED':
                // Handled by specific components
                break
            default:
                console.log('Unknown event type:', message.type)
        }
    }
}

export const webSocketService = new WebSocketService()
