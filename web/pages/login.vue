<template>
  <main class="login-page">
    <section class="login-card">
      <p class="eyebrow">Pressluft</p>
      <h1>Control Plane Login</h1>
      <p class="subtitle">Sign in with your operator account.</p>

      <form class="login-form" @submit.prevent="submitLogin">
        <label for="email">Email</label>
        <input
          id="email"
          v-model="email"
          name="email"
          type="email"
          autocomplete="username"
          required
          :disabled="isSubmitting"
        />

        <label for="password">Password</label>
        <input
          id="password"
          v-model="password"
          name="password"
          type="password"
          autocomplete="current-password"
          required
          :disabled="isSubmitting"
        />

        <p v-if="errorMessage" class="error" role="alert" aria-live="polite">{{ errorMessage }}</p>

        <button type="submit" :disabled="isSubmitting">
          {{ isSubmitting ? "Signing in..." : "Sign In" }}
        </button>
      </form>
    </section>
  </main>
</template>

<script setup lang="ts">
import { ApiClientError } from "~/lib/api/client";

const route = useRoute();
const auth = useAuthSession();

const email = ref("");
const password = ref("");
const errorMessage = ref("");
const isSubmitting = ref(false);

const redirectPath = computed(() => {
  const value = route.query.redirect;
  if (typeof value === "string" && value.startsWith("/") && !value.startsWith("//")) {
    return value;
  }
  return "/app";
});

const submitLogin = async (): Promise<void> => {
  isSubmitting.value = true;
  errorMessage.value = "";

  try {
    await auth.login({
      email: email.value,
      password: password.value,
    });
    await navigateTo(redirectPath.value, { replace: true });
  } catch (error) {
    if (error instanceof ApiClientError && error.status === 401) {
      errorMessage.value = "Invalid email or password.";
      return;
    }
    errorMessage.value = "Could not sign in. Please try again.";
  } finally {
    isSubmitting.value = false;
  }
};
</script>

<style scoped>
.login-page {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  padding: 2rem;
  background:
    radial-gradient(circle at top right, #dbeafe 0%, rgba(219, 234, 254, 0) 45%),
    radial-gradient(circle at bottom left, #fce7f3 0%, rgba(252, 231, 243, 0) 35%),
    #f8fafc;
}

.login-card {
  width: min(100%, 28rem);
  background: #ffffff;
  border: 1px solid #e2e8f0;
  border-radius: 1rem;
  padding: 2rem;
  box-shadow: 0 20px 40px -30px rgba(15, 23, 42, 0.5);
}

.eyebrow {
  margin: 0;
  font-size: 0.75rem;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: #475569;
}

h1 {
  margin: 0.4rem 0;
  color: #0f172a;
}

.subtitle {
  margin-top: 0;
  color: #475569;
}

.login-form {
  display: grid;
  gap: 0.75rem;
}

label {
  font-size: 0.9rem;
  color: #334155;
}

input {
  width: 100%;
  border: 1px solid #cbd5e1;
  border-radius: 0.625rem;
  padding: 0.65rem 0.75rem;
  font-size: 1rem;
}

input:focus-visible {
  outline: 2px solid #2563eb;
  outline-offset: 1px;
  border-color: #2563eb;
}

button {
  margin-top: 0.5rem;
  border: 0;
  border-radius: 0.625rem;
  padding: 0.7rem 0.95rem;
  font-size: 1rem;
  font-weight: 600;
  color: #ffffff;
  background: #1d4ed8;
  cursor: pointer;
}

button:disabled {
  opacity: 0.7;
  cursor: wait;
}

.error {
  margin: 0;
  font-size: 0.9rem;
  color: #b91c1c;
}

@media (max-width: 640px) {
  .login-card {
    padding: 1.25rem;
  }
}
</style>
