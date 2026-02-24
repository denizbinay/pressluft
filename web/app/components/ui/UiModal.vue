<script setup lang="ts">
interface Props {
  open: boolean
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: '',
})

const emit = defineEmits<{
  close: []
}>()

const isOpen = computed({
  get: () => props.open,
  set: (value) => {
    if (!value) {
      emit('close')
    }
  }
})
</script>

<template>
  <UModal v-model:open="isOpen" :title="props.title">
    <slot />
    <template #footer>
      <slot name="footer" />
    </template>
  </UModal>
</template>
