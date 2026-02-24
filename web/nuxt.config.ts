import tailwindcss from '@tailwindcss/vite'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  css: ['~/assets/css/main.css'],

  modules: ['@nuxtjs/google-fonts', '@nuxt/ui'],

  googleFonts: {
    families: {
      'Inter': '100..900',
      'JetBrains+Mono': '100..800',
    },
    display: 'swap',
    download: true,
    inject: true,
    subsets: ['latin'],
  },

  ui: {
    primary: 'cyan',
    gray: 'neutral',
  },

  vite: {
    plugins: [tailwindcss()],
  },

  nitro: {
    devProxy: {
      '/api': {
        target: process.env.NUXT_API_BASE || 'http://localhost:8081/api',
        changeOrigin: true,
      },
    },
  },
})
