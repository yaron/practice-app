import { defineConfig, loadEnv } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import fs from 'node:fs'
import path from 'node:path'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')

  return {
    plugins: [
      svelte(),
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
