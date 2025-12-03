import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import api from '../services/api'
import router from '../router'

export const useAuthStore = defineStore('auth', () => {
    const token = ref<string | null>(localStorage.getItem('token'))
    const user = ref<string | null>(localStorage.getItem('user'))

    const isAuthenticated = computed(() => !!token.value)

    const login = async (username: string, password: string) => {
        try {
            const response = await api.post('/auth/login', { username, password })
            const { token: newToken } = response.data.data

            token.value = newToken
            user.value = username

            localStorage.setItem('token', newToken)
            localStorage.setItem('user', username)

            return true
        } catch (error) {
            throw error
        }
    }

    const logout = async () => {
        try {
            if (token.value) {
                await api.post('/auth/logout')
            }
        } catch (e) {
            // Ignore errors on logout
        } finally {
            token.value = null
            user.value = null
            localStorage.removeItem('token')
            localStorage.removeItem('user')
            router.push('/login')
        }
    }

    return {
        token,
        user,
        isAuthenticated,
        login,
        logout
    }
})
