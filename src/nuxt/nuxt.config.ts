// https://nuxt.com/docs/api/configuration/nuxt-config
export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  ssr: false, // SPA mode - ingen server-side rendering
  devtools: { enabled: true },
  future: {
    compatibilityVersion: 4,
  },

  // SPA-konfiguration
  app: {
    baseURL: '/',
    buildAssetsDir: '/_nuxt/',
  },

  // Routing-regler för SPA
  routeRules: {
    // Allt annat än statiska filer = SPA fallback
    '/**': { ssr: false },
  },

  // Router-konfiguration
  router: {
    trailingSlash: false,
  },

  // Miljövariabel för API endpoint
  runtimeConfig: {
    public: {
      apiBase: process.env.NUXT_PUBLIC_API_BASE || 'http://localhost:8080',
    },
  },
})
