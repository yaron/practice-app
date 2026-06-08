import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

if ('serviceWorker' in navigator) {
  navigator.serviceWorker.register('/firebase-messaging-sw.js')
    .then(reg => reg.update())  // always check for a new SW version on page load
    .catch(() => {})
}

const app = mount(App, {
  target: document.getElementById('app'),
})

export default app
