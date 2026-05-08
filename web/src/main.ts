import { createApp } from 'vue'
import { createPinia } from 'pinia'
import piniaPersist from 'pinia-plugin-persistedstate'
import App from './App.vue'
import { router } from './router'
import { i18n } from './locales'
import './styles/tokens.css'
import './styles/tailwind.css'

const app = createApp(App)
const pinia = createPinia()
pinia.use(piniaPersist)
app.use(pinia)
app.use(router)
app.use(i18n)

// UI hydrate (theme + locale) runs inside App.vue setup() — calling
// useStore() here, before app.mount(), can race Pinia's active-
// instance setup and crash the bundle (blank page / no shell).
app.mount('#app')
