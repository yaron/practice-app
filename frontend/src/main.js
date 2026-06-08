import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/firebase-messaging-sw.js').catch(() => {
    // SW registration failure is non-fatal; FCM background notifications won't work
  })
}

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
