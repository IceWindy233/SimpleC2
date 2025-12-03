<template>
  <div class="toast-container">
    <TransitionGroup name="toast">
      <div
        v-for="toast in store.toasts"
        :key="toast.id"
        :class="['toast', `toast-${toast.type}`]"
      >
        <div class="toast-content">{{ toast.message }}</div>
        <button class="toast-close" @click="store.remove(toast.id)">&times;</button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { useToastStore } from '../../stores/toast'

const store = useToastStore()
</script>

<style scoped>
.toast-container {
  position: fixed;
  top: var(--spacing-md);
  right: var(--spacing-md);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  gap: var(--spacing-sm);
  pointer-events: none;
}

.toast {
  pointer-events: auto;
  min-width: 300px;
  max-width: 400px;
  padding: var(--spacing-md);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  color: #fff;
  font-size: var(--font-size-sm);
  line-height: 1.4;
}

.toast-success {
  background-color: var(--color-success);
}

.toast-error {
  background-color: var(--color-danger);
}

.toast-info {
  background-color: var(--color-info);
}

.toast-warning {
  background-color: var(--color-warning);
  color: #000;
}

.toast-content {
  flex: 1;
  margin-right: var(--spacing-sm);
}

.toast-close {
  background: none;
  border: none;
  color: inherit;
  font-size: 1.25rem;
  line-height: 1;
  cursor: pointer;
  opacity: 0.7;
  padding: 0;
}

.toast-close:hover {
  opacity: 1;
}

/* Transitions */
.toast-enter-active,
.toast-leave-active {
  transition: all 0.3s ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateX(30px);
}
</style>
