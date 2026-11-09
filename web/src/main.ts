import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import { startVitals } from './api/vitals'
import './assets/fonts.css'
import './assets/tokens.css'

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

startVitals()
