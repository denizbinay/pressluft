<script setup lang="ts">
type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info'

interface Props {
  variant?: BadgeVariant
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'default',
  size: 'md',
})

// Map old variants to NuxtUI variants
const nuxtUiVariant = computed(() => {
  const mapping: Record<BadgeVariant, 'solid' | 'soft' | 'outline' | 'subtle'> = {
    default: 'soft',
    success: 'soft',
    warning: 'soft',
    danger: 'soft',
    info: 'soft',
  }
  return mapping[props.variant]
})

const nuxtUiColor = computed(() => {
  const mapping: Record<BadgeVariant, 'primary' | 'success' | 'warning' | 'error' | 'info' | 'neutral'> = {
    default: 'neutral',
    success: 'success',
    warning: 'warning',
    danger: 'error',
    info: 'info',
  }
  return mapping[props.variant]
})
</script>

<template>
  <UBadge
    :variant="nuxtUiVariant"
    :color="nuxtUiColor"
    :size="props.size"
  >
    <slot />
  </UBadge>
</template>
