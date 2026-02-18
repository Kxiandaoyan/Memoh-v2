import {
  createRouter,
  createWebHistory,
  type RouteLocationNormalized,
} from 'vue-router'
import { h } from 'vue'
import { RouterView } from 'vue-router'
import { i18nRef } from './i18n'

const routes = [
  {
    path: '/',
    redirect: '/login',
    component: () => import('@/pages/main-section/index.vue'),
    children: [
      {
        name: 'chat',
        path: '/chat',
        component: () => import('@/pages/chat/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.chat'),
        },
      },
      {
        path: '/bots',
        component: { render: () => h(RouterView) },
        meta: {
          breadcrumb: i18nRef('sidebar.bots'),
        },
        children: [
          {
            name: 'bots',
            path: '',
            component: () => import('@/pages/bots/index.vue'),
          },
          {
            name: 'bot-detail',
            path: ':botId',
            component: () => import('@/pages/bots/detail.vue'),
            meta: {
              breadcrumb: (route: RouteLocationNormalized) => route.params.botId,
            },
          },
        ],
      },
      {
        name: 'shared-workspace',
        path: '/shared-workspace',
        component: () => import('@/pages/shared-workspace/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.sharedWorkspace'),
        },
      },
      {
        name: 'schedules',
        path: '/schedules',
        component: () => import('@/pages/schedules/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.schedules'),
        },
      },
      {
        name: 'models',
        path: '/models',
        component: () => import('@/pages/models/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.models'),
        },
      },
      {
        name: 'search-providers',
        path: '/search-providers',
        component: () => import('@/pages/search-providers/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.searchProvider'),
        },
      },
      {
        name: 'token-usage',
        path: '/token-usage',
        component: () => import('@/pages/token-usage/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.tokenUsage'),
        },
      },
      {
        name: 'logs',
        path: '/logs',
        component: () => import('@/pages/logs/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.logs'),
        },
      },
      {
        name: 'settings',
        path: '/settings',
        component: () => import('@/pages/settings/index.vue'),
        meta: {
          breadcrumb: i18nRef('sidebar.settings'),
        },
      },
    ],
  },
  {
    name: 'Login',
    path: '/login',
    component: () => import('@/pages/login/index.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})
router.beforeEach((to) => {
  const token = localStorage.getItem('token')

  if (to.fullPath !== '/login') {
    return token ? true : { name: 'Login' }
  } else {
    return token ? { path: '/chat' } : true
  }
})

export default router
