import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { VitePWA } from 'vite-plugin-pwa'
import fs from 'node:fs'
import path from 'node:path'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [
      svelte(),

      VitePWA({
        registerType: 'autoUpdate',
        // Inject the workbox SW registration script directly into index.html.
        // The firebase-messaging-sw.js is registered separately in main.js at
        // scope /firebase-cloud-messaging-push-scope/ to avoid conflict.
        injectRegister: 'inline',
        manifest: {
          name: 'Violin Quest',
          short_name: 'ViolinQuest',
          description: 'Gamified viooloefen-tracker',
          theme_color: '#6C3CE1',
          background_color: '#1A1A2E',
          display: 'standalone',
          orientation: 'portrait',
          scope: '/',
          start_url: '/',
          icons: [
            {
              src: '/icons/icon-192.png',
              sizes: '192x192',
              type: 'image/png',
            },
            {
              src: '/icons/icon-512.png',
              sizes: '512x512',
              type: 'image/png',
            },
            {
              src: '/icons/icon-512.png',
              sizes: '512x512',
              type: 'image/png',
              purpose: 'maskable',
            },
          ],
        },
        workbox: {
          // Precache all built assets; exclude the firebase messaging SW so it's
          // always fetched fresh (it contains injected Firebase credentials).
          globPatterns: ['**/*.{js,css,html,ico,png,svg}'],
          globIgnores: ['**/firebase-messaging-sw.js'],
          runtimeCaching: [
            {
              // Wheel options load offline (StaleWhileRevalidate: serve from
              // cache immediately, update in background for next visit).
              urlPattern: /^https?.*\/api\/options/,
              handler: 'StaleWhileRevalidate',
              options: {
                cacheName: 'wheel-options',
                expiration: {
                  maxEntries: 10,
                  maxAgeSeconds: 24 * 60 * 60, // 24 hours
                },
              },
            },
          ],
        },
      }),

      // Replaces __VITE_FIREBASE_*__ placeholders in firebase-messaging-sw.js.
      // In dev: serves the file via a middleware with values injected on the fly.
      // In build: rewrites the file in dist/ after bundle.
      // The public/ source uses placeholders because it can't access import.meta.env.
      {
        name: 'inject-sw-firebase-config',
        configureServer(server) {
          server.middlewares.use('/firebase-messaging-sw.js', (_req, res) => {
            const src = path.join(process.cwd(), 'public/firebase-messaging-sw.js')
            let content = fs.readFileSync(src, 'utf-8')
            content = content
              .replace(/__VITE_FIREBASE_API_KEY__/g,             env.VITE_FIREBASE_API_KEY             ?? '')
              .replace(/__VITE_FIREBASE_PROJECT_ID__/g,          env.VITE_FIREBASE_PROJECT_ID          ?? '')
              .replace(/__VITE_FIREBASE_MESSAGING_SENDER_ID__/g,  env.VITE_FIREBASE_MESSAGING_SENDER_ID ?? '')
              .replace(/__VITE_FIREBASE_APP_ID__/g,              env.VITE_FIREBASE_APP_ID              ?? '')
            res.setHeader('Content-Type', 'application/javascript')
            res.end(content)
          })
        },
        closeBundle() {
          const swOut = path.join(process.cwd(), 'dist/firebase-messaging-sw.js')
          if (!fs.existsSync(swOut)) return
          let content = fs.readFileSync(swOut, 'utf-8')
          content = content
            .replace(/__VITE_FIREBASE_API_KEY__/g,             env.VITE_FIREBASE_API_KEY             ?? '')
            .replace(/__VITE_FIREBASE_PROJECT_ID__/g,          env.VITE_FIREBASE_PROJECT_ID          ?? '')
            .replace(/__VITE_FIREBASE_MESSAGING_SENDER_ID__/g,  env.VITE_FIREBASE_MESSAGING_SENDER_ID ?? '')
            .replace(/__VITE_FIREBASE_APP_ID__/g,              env.VITE_FIREBASE_APP_ID              ?? '')
          fs.writeFileSync(swOut, content)
        },
      },
    ],
  }
})
