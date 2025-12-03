<template>
  <div class="form-group">
    <label v-if="label" :for="id" class="form-label">{{ label }}</label>
    <div class="input-wrapper">
      <input
        :id="id"
        :type="type"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :class="['form-control', { 'is-invalid': error }]"
        @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
        @blur="$emit('blur', $event)"
      />
    </div>
    <div v-if="error" class="invalid-feedback">{{ error }}</div>
    <div v-if="hint && !error" class="form-text">{{ hint }}</div>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue'

withDefaults(defineProps<{
  modelValue: string | number
  label?: string
  id?: string
  type?: string
  placeholder?: string
  disabled?: boolean
  error?: string
  hint?: string
}>(), {
  type: 'text',
  disabled: false
})

defineEmits(['update:modelValue', 'blur'])
</script>

<style scoped>
.form-group {
  margin-bottom: var(--spacing-md);
}

.form-label {
  display: block;
  margin-bottom: var(--spacing-xs);
  font-weight: 500;
  color: var(--color-text-primary);
}

.form-control {
  display: block;
  width: 100%;
  padding: 0.5rem 0.75rem;
  font-size: var(--font-size-base);
  font-family: inherit;
  color: var(--color-text-primary);
  background-color: var(--color-bg-secondary);
  background-clip: padding-box;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  transition: border-color 0.15s ease-in-out, box-shadow 0.15s ease-in-out;
}

.form-control:focus {
  border-color: var(--color-primary);
  outline: 0;
  box-shadow: 0 0 0 0.2rem rgba(13, 110, 253, 0.25);
}

.form-control:disabled {
  background-color: var(--color-bg-tertiary);
  opacity: 1;
}

.is-invalid {
  border-color: var(--color-danger);
}

.is-invalid:focus {
  border-color: var(--color-danger);
  box-shadow: 0 0 0 0.2rem rgba(220, 53, 69, 0.25);
}

.invalid-feedback {
  display: block;
  width: 100%;
  margin-top: 0.25rem;
  font-size: 0.875em;
  color: var(--color-danger);
}

.form-text {
  margin-top: 0.25rem;
  font-size: 0.875em;
  color: var(--color-text-secondary);
}
</style>
