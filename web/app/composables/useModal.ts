import { ref, readonly } from 'vue'

export function useModal() {
  const isOpen = ref(false)

  const open = () => { isOpen.value = true }
  const close = () => { isOpen.value = false }
  const toggle = () => { isOpen.value = !isOpen.value }

  return {
    isOpen: readonly(isOpen),
    open,
    close,
    toggle,
  }
}
