<template>
  <button
    :class="['btn', `btn-${variant}`, `btn-${size}`, { 'btn-block': block, 'btn-loading': loading }]"
    :disabled="disabled || loading"
    @click="$emit('click', $event)"
  >
    <span v-if="loading" class="spinner"></span>
    <span v-else>
      <slot></slot>
    </span>
  </button>
</template>

<script setup lang="ts">
import { defineProps } from 'vue'

withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'success' | 'danger' | 'warning' | 'info' | 'outline' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  block?: boolean
  disabled?: boolean
  loading?: boolean
}>(), {
  variant: 'primary',
  size: 'md',
  block: false,
  disabled: false,
  loading: false
})

defineEmits(['click'])
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 500;
  text-align: center;
  vertical-align: middle;
  user-select: none;
  border: 1px solid transparent;
  transition: all 0.2s ease-in-out;
  cursor: pointer;
  border-radius: var(--radius-md);
  outline: none;
}

.btn:disabled {
  opacity: 0.65;
  cursor: not-allowed;
}

/* Sizes */
.btn-sm {
  padding: 0.25rem 0.5rem;
  font-size: var(--font-size-sm);
}

.btn-md {
  padding: 0.5rem 1rem;
  font-size: var(--font-size-base);
}

.btn-lg {
  padding: 0.75rem 1.5rem;
  font-size: var(--font-size-lg);
}

.btn-block {
  display: flex;
  width: 100%;
}

/* Variants */
.btn-primary {
  background-color: var(--color-primary);
  color: #fff;
}
.btn-primary:hover:not(:disabled) {
  background-color: var(--color-primary-hover);
}

.btn-secondary {
  background-color: var(--color-bg-tertiary);
  color: var(--color-text-primary);
}
.btn-secondary:hover:not(:disabled) {
  background-color: #dbe0e5; /* Slightly darker than tertiary */
}

.btn-danger {
  background-color: var(--color-danger);
  color: #fff;
}
.btn-danger:hover:not(:disabled) {
  filter: brightness(0.9);
}

.btn-success {
  background-color: var(--color-success);
  color: #fff;
}
.btn-success:hover:not(:disabled) {
  filter: brightness(0.9);
}

.btn-outline {
  background-color: transparent;
  border-color: var(--color-border);
  color: var(--color-text-primary);
}
.btn-outline:hover:not(:disabled) {
  background-color: var(--color-bg-tertiary);
}

.btn-ghost {
  background-color: transparent;
  color: var(--color-text-primary);
}
.btn-ghost:hover:not(:disabled) {
  background-color: var(--color-bg-tertiary);
}

/* Loading Spinner */
.spinner {
  width: 1em;
  height: 1em;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.75s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
