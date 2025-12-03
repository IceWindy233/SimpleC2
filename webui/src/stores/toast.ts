import { defineStore } from 'pinia'
import { ref } from 'vue'

export interface Toast {
    id: string
    message: string
    type: 'success' | 'error' | 'info' | 'warning'
    duration?: number
}

export const useToastStore = defineStore('toast', () => {
    const toasts = ref<Toast[]>([])

    const add = (toast: Omit<Toast, 'id'>) => {
        const id = Date.now().toString()
        const newToast = { ...toast, id }
        toasts.value.push(newToast)

        if (toast.duration !== 0) {
            setTimeout(() => {
                remove(id)
            }, toast.duration || 3000)
        }
    }

    const remove = (id: string) => {
        toasts.value = toasts.value.filter((t) => t.id !== id)
    }

    const success = (message: string, duration?: number) => add({ message, type: 'success', duration })
    const error = (message: string, duration?: number) => add({ message, type: 'error', duration })
    const info = (message: string, duration?: number) => add({ message, type: 'info', duration })
    const warning = (message: string, duration?: number) => add({ message, type: 'warning', duration })

    return {
        toasts,
        add,
        remove,
        success,
        error,
        info,
        warning
    }
})
