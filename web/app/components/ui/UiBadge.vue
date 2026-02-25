<script setup lang="ts">
import { computed } from 'vue'

type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info'

interface Props {
  variant?: BadgeVariant
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  size: 'md',
})

const variantClass = computed(() => {
  const mapping: Record<BadgeVariant, string> = {
    default: 'border-surface-700/60 bg-surface-800/60 text-surface-100',
    success: 'border-success-700/40 bg-success-900/40 text-success-300',
    warning: 'border-warning-700/40 bg-warning-900/40 text-warning-300',
    danger: 'border-danger-700/40 bg-danger-900/40 text-danger-300',
    info: 'border-accent-600/40 bg-accent-500/15 text-accent-200',
  }

  return mapping[props.variant]
})

const sizeClass = computed(() => {
  const mapping: Record<NonNullable<Props['size']>, string> = {
    xs: 'px-1.5 py-0.5 text-[10px]',
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-1 text-sm',
    lg: 'px-3 py-1.5 text-sm',
    xl: 'px-3.5 py-2 text-base',
  }

  return mapping[props.size]
})
</script>

<template>
  <span
    class="inline-flex items-center rounded-full border font-medium"
    :class="[variantClass, sizeClass]"
  >
    <slot />
  </span>
</template>
