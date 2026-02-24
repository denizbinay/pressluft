<script setup lang="ts">
type ButtonColor = 'primary' | 'secondary' | 'success' | 'info' | 'warning' | 'error' | 'neutral'
type ButtonVariant = 'solid' | 'outline' | 'soft' | 'subtle' | 'ghost' | 'link'
type ButtonSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl'

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

// Map old variant to NuxtUI color and variant
const nuxtUiColor = computed<ButtonColor>(() => {
  const mapping: Record<ButtonVariantOld, ButtonColor> = {
    primary: 'primary',
    secondary: 'neutral',
    outline: 'neutral',
    ghost: 'neutral',
    danger: 'error',
  }
  return mapping[props.variant]
})

const nuxtUiVariant = computed<ButtonVariant>(() => {
  const mapping: Record<ButtonVariantOld, ButtonVariant> = {
    primary: 'solid',
    secondary: 'solid',
    outline: 'outline',
    ghost: 'ghost',
    danger: 'solid',
  }
  return mapping[props.variant]
})

const nuxtUiSize = computed<ButtonSize>(() => {
  // NuxtUI supports 'xs' | 'sm' | 'md' | 'lg' | 'xl'
  // We map directly: sm → sm, md → md, lg → lg
  return props.size as ButtonSize
})
</script>

<template>
  <UButton
    :color="nuxtUiColor"
    :variant="nuxtUiVariant"
    :size="nuxtUiSize"
    :disabled="props.disabled"
    :loading="props.loading"
    class="cursor-pointer"
  >
    <slot />
  </UButton>
</template>
