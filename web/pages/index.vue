<template>
  <main class="route-loading">
    <p>Loading dashboard...</p>
    <p class="fallback-links">
      <a href="/login">Go to login</a>
      <span aria-hidden="true">·</span>
      <a href="/app">Go to app</a>
    </p>
  </main>
</template>

<style scoped>
.route-loading {
  min-height: 100dvh;
  display: grid;
  place-items: center;
  color: #4b5563;
}

.fallback-links {
  display: flex;
  gap: 0.5rem;
  align-items: center;
  font-size: 0.95rem;
}

.fallback-links a {
  color: #1f2937;
  text-decoration: underline;
}

.fallback-links a:hover {
  text-decoration-thickness: 2px;
}
</style>
