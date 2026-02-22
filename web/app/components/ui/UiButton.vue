<script setup lang="ts">
type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'ghost' | 'danger'
type ButtonSize = 'sm' | 'md' | 'lg'

interface Props {
  variant?: ButtonVariant
  size?: ButtonSize
  disabled?: boolean
  loading?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  disabled: false,
  loading: false,
})

const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-accent-500 text-surface-950 hover:bg-accent-400 active:bg-accent-600 font-semibold',
  secondary:
    'bg-surface-800 text-surface-100 hover:bg-surface-700 active:bg-surface-800 border border-surface-700',
  outline:
    'bg-transparent text-surface-200 border border-surface-600 hover:bg-surface-800/60 hover:border-surface-500 active:bg-surface-800',
  ghost:
    'bg-transparent text-surface-300 hover:bg-surface-800/60 hover:text-surface-100 active:bg-surface-800',
  danger:
    'bg-danger-600 text-surface-50 hover:bg-danger-500 active:bg-danger-600 font-semibold',
}

const sizeClasses: Record<ButtonSize, string> = {
  sm: 'h-8 px-3 text-xs rounded-md gap-1.5',
  md: 'h-9 px-4 text-sm rounded-lg gap-2',
  lg: 'h-11 px-6 text-sm rounded-lg gap-2',
}
</script>

<template>
  <button
    :class="[
      'inline-flex items-center justify-center font-medium transition-all duration-150 cursor-pointer',
      'disabled:opacity-40 disabled:pointer-events-none',
      variantClasses[props.variant],
      sizeClasses[props.size],
    ]"
    :disabled="props.disabled || props.loading"
  >
    <!-- Loading spinner -->
    <svg
      v-if="props.loading"
      class="h-4 w-4 animate-spin"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"
      />
    </svg>
    <slot />
  </button>
</template>
