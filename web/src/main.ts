import { createApp } from 'vue'
import App from './App.vue'
import './styles/base.css'
import { syncDocumentLang } from './i18n'
import { installNav } from './nav'
import { settleSessionLink } from './store/session'

syncDocumentLang()
installNav(settleSessionLink)

createApp(App).mount('#app')
