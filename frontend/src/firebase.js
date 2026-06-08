import { initializeApp } from 'firebase/app'
import { getMessaging } from 'firebase/messaging'

export const VAPID_KEY = import.meta.env.VITE_FIREBASE_VAPID_KEY ?? ''

/** @type {import('firebase/messaging').Messaging | null} */
let messaging = null

if (VAPID_KEY) {
  const app = initializeApp({
    apiKey:            import.meta.env.VITE_FIREBASE_API_KEY,
    projectId:         import.meta.env.VITE_FIREBASE_PROJECT_ID,
    messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
    appId:             import.meta.env.VITE_FIREBASE_APP_ID,
  })
  messaging = getMessaging(app)
}

export { messaging }
