<script setup lang="ts">
definePageMeta({ layout: false })

const route = useRoute()
const router = useRouter()
const { login } = useAuth()

const email = ref('')
const password = ref('')
const error = ref('')
const loading = ref(false)

const redirectTarget = computed(() => {
  const raw = route.query.redirect
  return typeof raw === 'string' && raw.startsWith('/') ? raw : '/'
})

const submit = async () => {
  loading.value = true
  error.value = ''
  try {
    await login(email.value, password.value)
    await router.push(redirectTarget.value)
  } catch (err: any) {
    error.value = err?.data?.error || err?.message || 'Login failed'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="flex min-h-screen items-center justify-center bg-background px-6">
    <div class="w-full max-w-md rounded-2xl border border-border/60 bg-card/80 p-8 shadow-sm backdrop-blur">
      <div class="mb-8 space-y-2">
        <h1 class="text-2xl font-semibold text-foreground">Sign in</h1>
        <p class="text-sm text-muted-foreground">
          Access the Pressluft control plane.
        </p>
      </div>

      <form class="space-y-4" @submit.prevent="submit">
        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground" for="email">Email</label>
          <input
            id="email"
            v-model="email"
            type="email"
            autocomplete="username"
            class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            required
          >
        </div>

        <div class="space-y-2">
          <label class="text-sm font-medium text-foreground" for="password">Password</label>
          <input
            id="password"
            v-model="password"
            type="password"
            autocomplete="current-password"
            class="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
            required
          >
        </div>

        <p v-if="error" class="text-sm text-destructive">{{ error }}</p>

        <button
          type="submit"
          class="inline-flex h-10 w-full items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-60"
          :disabled="loading"
        >
          {{ loading ? 'Signing in...' : 'Sign in' }}
        </button>
      </form>
    </div>
  </div>
</template>
