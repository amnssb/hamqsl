import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  // 服务器公开门户：访客默认从这里开始
  {
    path: '/',
    name: 'PublicHome',
    component: () => import('../views/public/Home.vue'),
    meta: { public: true, title: '公开服务' }
  },
  // 公开页面（无需登录）
  {
    path: '/apply',
    name: 'PublicApply',
    component: () => import('../views/public/ApplyForm.vue'),
    meta: { public: true, title: 'QSL 换卡申请' }
  },
  {
    path: '/confirm',
    name: 'PublicConfirmHome',
    component: () => import('../views/public/ConfirmReceipt.vue'),
    meta: { public: true, title: '确认收件' }
  },
  {
    path: '/confirm/:code',
    name: 'PublicConfirm',
    component: () => import('../views/public/ConfirmReceipt.vue'),
    meta: { public: true, title: '确认收件' }
  },
  {
    path: '/status',
    name: 'PublicRequestStatusHome',
    component: () => import('../views/public/RequestStatus.vue'),
    meta: { public: true, title: '申请进度查询' }
  },
  {
    path: '/status/:code',
    name: 'PublicRequestStatus',
    component: () => import('../views/public/RequestStatus.vue'),
    meta: { public: true, title: '申请进度查询' }
  },
  {
    path: '/track',
    name: 'PublicTrackHome',
    component: () => import('../views/public/TrackCard.vue'),
    meta: { public: true, title: '快递追踪' }
  },
  {
    path: '/track/:code',
    name: 'PublicTrack',
    component: () => import('../views/public/TrackCard.vue'),
    meta: { public: true, title: '快递追踪' }
  },

  // 兼容旧版后台链接，避免服务器升级后已有书签失效
  { path: '/dashboard', redirect: '/admin/dashboard' },
  { path: '/qso', redirect: '/admin/qso' },
  { path: '/cards', redirect: '/admin/cards' },
  { path: '/exchange/online', redirect: '/admin/exchange/online' },
  { path: '/exchange/offline', redirect: '/admin/exchange/offline' },
  { path: '/receive', redirect: '/admin/receive' },
  { path: '/address', redirect: '/admin/address' },
  { path: '/bureau', redirect: '/admin/bureau' },
  { path: '/station', redirect: '/admin/station' },

  // 登录
  {
    path: '/login',
    name: 'Login',
    component: () => import('../views/Login.vue'),
  },

  // 后台管理（需要登录）
  {
    path: '/admin',
    component: () => import('../views/Layout.vue'),
    redirect: '/admin/dashboard',
    children: [
      { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '总览' } },
      { path: 'qso', name: 'QsoList', component: () => import('../views/QsoList.vue'), meta: { title: '通联日志' } },
      { path: 'cards', name: 'CardList', component: () => import('../views/CardList.vue'), meta: { title: '卡片记录' } },
      { path: 'exchange/online', name: 'ExchangeOnline', component: () => import('../views/ExchangeOnline.vue'), meta: { title: '线上换卡' } },
      { path: 'exchange/offline', name: 'ExchangeOffline', component: () => import('../views/ExchangeOffline.vue'), meta: { title: '线下换卡' } },
      { path: 'receive', name: 'ReceiveList', component: () => import('../views/ReceiveList.vue'), meta: { title: '收卡记录' } },
      { path: 'address', name: 'AddressBook', component: () => import('../views/AddressBook.vue'), meta: { title: '地址管理' } },
      { path: 'bureau', name: 'BureauList', component: () => import('../views/BureauList.vue'), meta: { title: '卡片局' } },
      { path: 'station', name: 'StationConfig', component: () => import('../views/StationConfig.vue'), meta: { title: '卡片版本' } },
      { path: 'settings', name: 'Settings', component: () => import('../views/Settings.vue'), meta: { title: '设置' } },
    ],
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  // 公开页面不需要登录
  if (to.meta.public) {
    next()
    return
  }
  // 登录页面不需要登录
  if (to.path === '/login') {
    next()
    return
  }
  // 其他页面需要登录
  if (!localStorage.getItem('token')) {
    next('/login')
  } else {
    next()
  }
})

export default router
