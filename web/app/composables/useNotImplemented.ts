const isOpen = ref(false);

export function useNotImplemented() {
  function trigger() {
    isOpen.value = true;
  }

  function close() {
    isOpen.value = false;
  }

  return { isOpen, trigger, close };
}
