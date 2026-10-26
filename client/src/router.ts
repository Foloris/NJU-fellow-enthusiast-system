import { createRouter, createWebHistory } from "vue-router";
import { auth } from "@/stores/auth";

const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: () => ({ top: 0 }),
  routes: [
    {
      path: "/",
      name: "home",
      component: () => import("@/views/HomeView.vue"),
    },
    {
      path: "/login",
      name: "login",
      component: () => import("@/views/LoginView.vue"),
      meta: { guestOnly: true },
    },
    {
      path: "/exchange",
      name: "exchange",
      component: () => import("@/views/ExchangeListView.vue"),
    },
    {
      path: "/exchange/new",
      name: "exchange-new",
      component: () => import("@/views/ExchangeEditorView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/exchange/:id",
      name: "exchange-detail",
      component: () => import("@/views/ExchangeDetailView.vue"),
    },
    {
      path: "/exchange/:id/edit",
      name: "exchange-edit",
      component: () => import("@/views/ExchangeEditorView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/mates",
      name: "mates",
      component: () => import("@/views/MatesListView.vue"),
    },
    {
      path: "/mates/new",
      name: "mate-new",
      component: () => import("@/views/MateEditorView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/mates/:id",
      name: "mate-detail",
      component: () => import("@/views/MateDetailView.vue"),
    },
    {
      path: "/mates/:id/edit",
      name: "mate-edit",
      component: () => import("@/views/MateEditorView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/messages",
      name: "messages",
      component: () => import("@/views/MessagesView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/profile",
      name: "profile",
      component: () => import("@/views/ProfileView.vue"),
      meta: { requiresAuth: true },
    },
    {
      path: "/admin",
      name: "admin",
      component: () => import("@/views/AdminView.vue"),
      meta: { requiresAuth: true, requiresAdmin: true },
    },
    {
      path: "/:pathMatch(.*)*",
      name: "not-found",
      component: () => import("@/views/NotFoundView.vue"),
    },
  ],
});

router.beforeEach(async (to) => {
  await auth.loadMe();
  // Every application page requires a session; only login/registration is public.
  if (!to.meta.guestOnly && !auth.state.user) {
    return { name: "login", query: { redirect: to.fullPath } };
  }
  if (to.meta.requiresAdmin && !auth.isAdmin.value) {
    return { name: "home" };
  }
  if (to.meta.guestOnly && auth.state.user) {
    return { name: "home" };
  }
});

export default router;

