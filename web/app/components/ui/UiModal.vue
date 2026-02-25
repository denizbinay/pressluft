<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'

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

const isOpen = computed(() => props.open)

const handleClose = () => {
  emit('close')
}

const handleEscape = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && isOpen.value) {
    handleClose()
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleEscape)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', handleEscape)
})
</script>

<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-150 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-100 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center">
        <div
          class="absolute inset-0 bg-surface-950/70 backdrop-blur-sm"
          @click="handleClose"
        />
        <div
          class="relative z-10 w-full max-w-lg rounded-xl border border-surface-800/60 bg-surface-900/80 shadow-2xl"
        >
          <div class="flex items-center justify-between border-b border-surface-800/40 px-6 py-4">
            <h3 class="text-base font-semibold text-surface-100">{{ props.title }}</h3>
            <button
              type="button"
              class="inline-flex h-8 w-8 items-center justify-center rounded-md text-surface-300 transition hover:bg-surface-800/60 hover:text-surface-100"
              aria-label="Close modal"
              @click="handleClose"
            >
              <span aria-hidden="true">×</span>
            </button>
          </div>

          <div class="px-6 py-5">
            <slot />
          </div>

          <div v-if="$slots.footer" class="border-t border-surface-800/40 px-6 py-4">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
