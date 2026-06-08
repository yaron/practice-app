// Firebase Messaging service worker — handles background push notifications.
// Placeholders (__VITE_FIREBASE_*__) are replaced with real values during `npm run build`
// via the inject-sw-firebase-config plugin in vite.config.js.
// In dev mode the SW loads but Firebase init silently fails (notifications require real credentials).

const FIREBASE_VERSION = '12.14.0'

importScripts(`https://www.gstatic.com/firebasejs/${FIREBASE_VERSION}/firebase-app-compat.js`)
importScripts(`https://www.gstatic.com/firebasejs/${FIREBASE_VERSION}/firebase-messaging-compat.js`)

firebase.initializeApp({
  apiKey:            '__VITE_FIREBASE_API_KEY__',
  projectId:         '__VITE_FIREBASE_PROJECT_ID__',
  messagingSenderId: '__VITE_FIREBASE_MESSAGING_SENDER_ID__',
  appId:             '__VITE_FIREBASE_APP_ID__',
})

const messaging = firebase.messaging()

messaging.onBackgroundMessage((payload) => {
  const { title, body } = payload.notification ?? {}
  self.registration.showNotification(title ?? 'Viool Quest', {
    body: body ?? '',
    icon: '/icons/icon-192.png',
  })
})
