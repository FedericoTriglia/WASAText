import {createRouter, createWebHashHistory} from 'vue-router'
import LoginView from '../views/LoginView.vue'
import ChatView from '../views/ChatView.vue'

const router = createRouter({
	history: createWebHashHistory(import.meta.env.BASE_URL),
	routes: [
		{path: '/', redirect: '/login'},
		{path: '/login', component: LoginView},
		{path: '/conversations', component: ChatView},
	]
})

// Redirect to login if no token is present
router.beforeEach((to) => {
	const token = localStorage.getItem('token');
	if (to.path !== '/login' && !token) {
		return '/login';
	}
})

export default router
