import axios from 'axios'
import { useToastStore } from '../stores/toast'

const api = axios.create({
    baseURL: '/api', // Proxy will handle this in dev, or relative path in prod
    timeout: 10000,
    headers: {
        'Content-Type': 'application/json'
    }
})

// Request interceptor
api.interceptors.request.use(
    (config) => {
        // Add auth token here if needed
        const token = localStorage.getItem('token')
        if (token) {
            config.headers.Authorization = `Bearer ${token}`
        }
        return config
    },
    (error) => {
        return Promise.reject(error)
    }
)

// Response interceptor
api.interceptors.response.use(
    (response) => {
        return response
    },
    (error) => {
        const toast = useToastStore()

        if (error.response) {
            // Server responded with a status code outside 2xx
            const message = error.response.data?.message || 'An error occurred'
            toast.error(`Error ${error.response.status}: ${message}`)

            if (error.response.status === 401) {
                // Handle unauthorized (redirect to login)
                localStorage.removeItem('token')
                localStorage.removeItem('user')
                window.location.href = '/login'
            }
        } else if (error.request) {
            // Request was made but no response received
            toast.error('Network error: No response received')
        } else {
            // Something happened in setting up the request
            toast.error(`Error: ${error.message}`)
        }

        return Promise.reject(error)
    }
)

export const downloadLootFile = async (filename: string) => {
    try {
        const response = await api.get(`/loot/${filename}`, { responseType: 'blob' })
        return response.data
    } catch (error) {
        throw error
    }
}

export const getTunnels = async () => {
    try {
        const response = await api.get('/tunnels')
        return response.data
    } catch (error) {
        throw error
    }
}

export const startTunnel = async (beaconId: string, target: string) => {
    try {
        const response = await api.post('/tunnels/start', { beacon_id: beaconId, target })
        return response.data
    } catch (error) {
        throw error
    }
}

export const closeTunnel = async (tunnelId: string) => {
    try {
        const response = await api.post(`/tunnels/${tunnelId}/close`)
        return response.data
    } catch (error) {
        throw error
    }
}

export default api
