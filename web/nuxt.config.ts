// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },
  nitro: {
    devProxy: {
      '/api': {
        target: process.env.NUXT_API_BASE || 'http://localhost:8081/api',
        changeOrigin: true,
      },
    },
  },
})
