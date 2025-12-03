<template>
  <div class="table-responsive">
    <table class="table">
      <thead>
        <tr>
          <th 
            v-for="col in columns" 
            :key="col.key" 
            :style="{ width: col.width, cursor: col.sortable ? 'pointer' : 'default' }"
            @click="col.sortable && $emit('sort', col.key)"
          >
            {{ col.label }}
            <span v-if="col.sortable && sortKey === col.key">
              {{ sortOrder === 'asc' ? '↑' : '↓' }}
            </span>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr v-if="loading">
          <td :colspan="columns.length" class="text-center py-4">
            <div class="spinner"></div> 加载中...
          </td>
        </tr>
        <tr v-else-if="!data || data.length === 0">
          <td :colspan="columns.length" class="text-center py-4 text-muted">
            <slot name="empty">
              暂无数据
            </slot>
          </td>
        </tr>
        <tr 
          v-else 
          v-for="(row, index) in data" 
          :key="index"
          @click="$emit('row-click', row)"
          @dblclick="$emit('row-dblclick', row)"
          :class="{ 'row-clickable': true, 'row-selected': selectedRow === row }"
        >
          <td v-for="col in columns" :key="col.key">
            <slot :name="col.key" :row="row" :value="row[col.key]">
              {{ row[col.key] }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script setup lang="ts">
import { defineProps } from 'vue'

export interface Column {
  key: string
  label: string
  width?: string
  sortable?: boolean
}

defineProps<{
  columns: Column[]
  data: any[]
  loading?: boolean
  sortKey?: string
  sortOrder?: 'asc' | 'desc'
  selectedRow?: any
}>()

defineEmits<{
  (e: 'row-click', row: any): void
  (e: 'row-dblclick', row: any): void
  (e: 'sort', key: string): void
}>()
</script>

<style scoped>
.table-responsive {
  width: 100%;
  height: 100%;
  overflow: auto;
  -webkit-overflow-scrolling: touch;
}

.table {
  width: 100%;
  border-collapse: collapse;
  color: var(--color-text-primary);
}

.table th,
.table td {
  padding: var(--spacing-md);
  vertical-align: middle;
  border-bottom: 1px solid var(--color-border);
  text-align: left;
}

.table th {
  font-weight: 600;
  background-color: var(--color-bg-tertiary);
  color: var(--color-text-primary);
  white-space: nowrap;
}

.table tbody tr:hover {
  background-color: rgba(255, 255, 255, 0.05);
}

.row-clickable {
  cursor: pointer;
}

.row-selected {
  background-color: rgba(255, 255, 255, 0.1) !important;
}

.text-center {
  text-align: center;
}

.text-muted {
  color: var(--color-text-secondary);
}

.spinner {
  display: inline-block;
  width: 1rem;
  height: 1rem;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: spin 0.75s linear infinite;
  margin-right: 0.5rem;
  vertical-align: text-bottom;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}
</style>
