<script setup lang="ts">
import { computed, useAttrs } from 'vue'

type ButtonVariantOld = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger'
type ButtonSizeOld = 'sm' | 'md' | 'lg'

interface Props {
  variant?: ButtonVariantOld
  size?: ButtonSizeOld
  disabled?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
})

const attrs = useAttrs()

const sizeClass = computed(() => {
  const mapping: Record<ButtonSizeOld, string> = {
    sm: 'h-8 px-3 text-xs',
    md: 'h-10 px-4 text-sm',
    lg: 'h-11 px-5 text-sm',
  }

  return mapping[props.size]
})

const variantClass = computed(() => {
  const mapping: Record<ButtonVariantOld, string> = {
    primary: 'bg-primary text-primary-foreground hover:bg-primary/90',
    secondary: 'bg-surface-800 text-surface-100 hover:bg-surface-700',
    outline: 'border border-surface-700 text-surface-100 hover:bg-surface-800/60',
    ghost: 'text-surface-200 hover:bg-surface-800/50',
    danger: 'bg-danger-600 text-white hover:bg-danger-500',
  }

  return mapping[props.variant]
})

const isDisabled = computed(() => props.disabled || props.loading)
</script>

<template>
  <button
    v-bind="attrs"
    :disabled="isDisabled"
    class="inline-flex items-center justify-center gap-2 rounded-lg font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent-500/60 focus-visible:ring-offset-2 focus-visible:ring-offset-surface-950"
    :class="[
      sizeClass,
      variantClass,
      isDisabled && 'cursor-not-allowed opacity-60',
    ]"
  >
    <span
      v-if="props.loading"
      class="inline-flex h-4 w-4 animate-spin items-center justify-center rounded-full border-2 border-current border-b-transparent"
      aria-hidden="true"
    />
    <slot />
  </button>
</template>
