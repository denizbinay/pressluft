<script setup lang="ts">
type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info'

interface Props {
  variant?: BadgeVariant
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
})

const variantClasses: Record<BadgeVariant, string> = {
  default: 'bg-surface-800 text-surface-300 border-surface-700',
  success: 'bg-success-900/60 text-success-400 border-success-600/30',
  warning: 'bg-warning-900/60 text-warning-400 border-warning-600/30',
  danger: 'bg-danger-900/60 text-danger-400 border-danger-600/30',
  info: 'bg-primary-700/30 text-primary-300 border-primary-600/30',
}
</script>

<template>
  <span
    :class="[
      'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium',
      variantClasses[props.variant],
    ]"
  >
    <slot />
  </span>
</template>
