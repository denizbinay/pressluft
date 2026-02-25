<script setup lang="ts">
import { computed } from 'vue'

type ProgressVariant = 'accent' | 'success' | 'warning' | 'danger'

interface Props {
  value: number
  max?: number
  variant?: ProgressVariant
  showLabel?: boolean
  size?: 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  max: 100,
  variant: 'accent',
  showLabel: false,
  size: 'md',
})

const percentage = computed(() =>
  Math.min(100, Math.max(0, (props.value / props.max) * 100)),
)

const barClass = computed(() => {
  const mapping: Record<ProgressVariant, string> = {
    accent: 'bg-accent-500',
    success: 'bg-success-500',
    warning: 'bg-warning-500',
    danger: 'bg-danger-500',
  }

  return mapping[props.variant]
})

const trackHeight = computed(() => {
  const mapping: Record<NonNullable<Props['size']>, string> = {
    sm: 'h-1.5',
    md: 'h-2.5',
    lg: 'h-3.5',
  }

  return mapping[props.size]
})
</script>

<template>
  <div class="w-full">
    <div v-if="props.showLabel" class="mb-1.5 flex items-center justify-between">
      <slot name="label">
        <span class="text-xs font-medium text-surface-300">Progress</span>
      </slot>
      <span class="text-xs font-mono text-surface-400">{{ Math.round(percentage) }}%</span>
    </div>
    <div class="w-full overflow-hidden rounded-full bg-surface-800/70" :class="trackHeight">
      <div
        class="h-full rounded-full transition-[width] duration-300"
        :class="barClass"
        :style="{ width: `${percentage}%` }"
      />
    </div>
  </div>
</template>
