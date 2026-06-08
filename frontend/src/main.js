import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

if ('serviceWorker' in navigator) {
  // Register the FCM messaging SW at its dedicated push scope so it doesn't
  // conflict with the workbox SW registered at scope '/'.
  navigator.serviceWorker.register('/firebase-messaging-sw.js', {
    scope: '/firebase-cloud-messaging-push-scope/'
  })
    .then(reg => reg.update()) // always check for a new SW version on page load
    .catch(() => {})
}

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
