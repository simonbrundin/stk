// https://nuxt.com/docs/api/configuration/nuxt-config

// Statisk byggning för GitHub Pages. Sätts av .github/workflows/pages.yaml:
//
//   NUXT_PAGES_BUILD=true   bygger en statisk sajt (SSR + prerender)
//   NUXT_APP_BASE_URL       sökvägen sajten ligger under (t.ex. /stk/)
//
// Utan NUXT_PAGES_BUILD byggs SPA:n som tidigare (ssr: false) och används
// av nginx-imagen i klustret.
const pagesBuild = process.env.NUXT_PAGES_BUILD === 'true'
const baseURL = process.env.NUXT_APP_BASE_URL || '/'

export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  ssr: pagesBuild, // SPA mode - ingen server-side rendering
  devtools: { enabled: true },
  future: {
    compatibilityVersion: 4,
  },

  // SPA-konfiguration
  app: {
    baseURL,
    buildAssetsDir: '/_nuxt/',
    head: {
      link: [
        {
          rel: 'preload',
          href: `${baseURL}fonts/inter-latin-var.woff2`,
          as: 'font',
          type: 'font/woff2',
          crossorigin: '',
        },
      ],
      style: [
        {
          children:
            "@font-face{font-family:'Inter';font-style:normal;font-weight:100 900;" +
            "font-display:swap;src:url('" +
            `${baseURL}fonts/inter-latin-var.woff2` +
            "') format('woff2');" +
            'unicode-range:U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,' +
            'U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,' +
            'U+2212,U+2215,U+FEFF,U+FFFD;}',
        },
        {
          children:
            "@font-face{font-family:'Archivo';font-style:normal;font-weight:100 900;" +
            "font-stretch:62% 125%;font-display:swap;src:url('" +
            `${baseURL}fonts/archivo-latin-var.woff2` +
            "') format('woff2');" +
            'unicode-range:U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,' +
            'U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,' +
            'U+2212,U+2215,U+FEFF,U+FFFD;}',
        },
      ],
    },
  },

  // Routing-regler för SPA
  routeRules: {
    // Allt annat än statiska filer = SPA fallback
    '/**': { ssr: pagesBuild },
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

  // Dev-server: /api/ proxas till Go API:t, samma sökväg som nginx i produktion.
  vite: {
    server: {
      proxy: {
        '/api': {
          target: process.env.API_PROXY_TARGET || 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (path: string) => path.replace(/^\/api/, ''),
        },
      },
    },
  },

  // Sidorna som ska prerenderas i den statiska byggningen.
  nitro: pagesBuild
    ? {
        prerender: {
          crawlLinks: false,
          routes: ['/', '/lunch'],
          failOnError: false,
        },
      }
    : {},
})
