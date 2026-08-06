import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import './style.css'
import BreedQuery from './views/BreedQuery.vue'
import SpeciesView from './views/SpeciesView.vue'
import PetsView from './views/PetsView.vue'
import LinesView from './views/LinesView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', name: 'breed', component: BreedQuery },
    { path: '/lines', name: 'lines', component: LinesView },
    { path: '/pets', name: 'pets', component: PetsView },
    { path: '/species', name: 'species', component: SpeciesView },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

createApp(App).use(router).mount('#app')
