<script setup lang="ts">
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

// Map variants to NuxtUI colors
const nuxtUiColor = computed(() => {
  const mapping: Record<ProgressVariant, 'primary' | 'success' | 'warning' | 'error'> = {
    accent: 'primary',
    success: 'success',
    warning: 'warning',
    danger: 'error',
  }
  return mapping[props.variant]
})

// Map sizes to NuxtUI sizes
const nuxtUiSize = computed(() => {
  const mapping: Record<string, 'xs' | 'sm' | 'md' | 'lg' | 'xl'> = {
    sm: 'xs',
    md: 'sm',
    lg: 'md',
  }
  return mapping[props.size] || 'sm'
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
    <UProgress
      :model-value="percentage"
      :color="nuxtUiColor"
      :size="nuxtUiSize"
      :status="false"
    />
  </div>
</template>
