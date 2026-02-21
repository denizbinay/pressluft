export default defineNuxtConfig({
  compatibilityDate: "2025-11-01",
  devtools: { enabled: false },
  // Embedded dashboard is served as static assets from the Go control plane.
  ssr: false,
  css: ["~/assets/css/main.css"],
  runtimeConfig: {
    public: {
      apiBase: "/api",
    },
  },
  typescript: {
    strict: true,
    typeCheck: true,
  },
});
