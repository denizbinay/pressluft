import tailwindcss from '@tailwindcss/vite'

const devProxyTarget = process.env.NUXT_DEV_PROXY_TARGET || 'http://localhost:8081/api'

// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  devtools: { enabled: true },

  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_API_BASE || '/api',
    },
  },

  css: ['~/assets/css/main.css'],

  modules: ['@nuxtjs/google-fonts', '@nuxtjs/color-mode', 'shadcn-nuxt'],

  googleFonts: {
    families: {
      'Geist': '100..900',
      'JetBrains+Mono': '100..800',
    },
    display: 'swap',
    download: true,
    inject: true,
    subsets: ['latin'],
  },

  colorMode: {
    classSuffix: '',
    preference: 'system',
    fallback: 'light',
  },

  vite: {
    plugins: [tailwindcss()],
  },

  shadcn: {     
    prefix: 'Ui',
    componentDir: './app/components/ui'
  },

  nitro: {
    routeRules: {
      '/api/**': {
        proxy: `${devProxyTarget}/**`,
      },
    },
  },
})
