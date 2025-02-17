import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import router from './router/router'
import App from './App.vue'
import './styles/style.css'


const app = createApp(App)
const pinia = createPinia()
app.use(pinia)

app.use(router)
app.use(ElementPlus)
app.mount('#app')
