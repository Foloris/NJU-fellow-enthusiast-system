<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  LogOut,
  Menu,
  MessageCircle,
  Settings,
  UserRound,
  X,
} from "lucide-vue-next";
import Avatar from "./Avatar.vue";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";

const route = useRoute();
const router = useRouter();
const mobileOpen = ref(false);

const nav = computed(() => [
  { to: "/", label: "发现" },
  { to: "/exchange", label: "同好交流" },
  { to: "/mates", label: "找搭子" },
  ...(auth.state.user
    ? [{ to: "/messages", label: "私信" }]
    : []),
]);

async function logout() {
  try {
    await auth.logout();
    mobileOpen.value = false;
    notify("已安全退出");
    router.push("/");
  } catch (error) {
    notify(error instanceof Error ? error.message : "退出失败", "error");
  }
}
</script>

<template>
  <header class="site-header">
    <div class="header-inner">
      <RouterLink class="brand" to="/" @click="mobileOpen = false">
        <span class="brand-mark">
          <span></span><span></span>
        </span>
        <span class="brand-copy">
          <strong>搭个伴</strong>
          <small>校园同好社区</small>
        </span>
      </RouterLink>

      <nav class="desktop-nav">
        <RouterLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          :class="{ active: item.to === '/' ? route.path === '/' : route.path.startsWith(item.to) }"
        >
          {{ item.label }}
        </RouterLink>
      </nav>

      <div class="header-actions">
        <template v-if="auth.state.user">
          <RouterLink
            v-if="auth.state.user.role === 'admin'"
            to="/admin"
            class="header-icon admin-icon"
            title="管理后台"
          >
            <Settings :size="19" />
          </RouterLink>
          <RouterLink class="user-chip" to="/profile">
            <Avatar :user="auth.state.user" :size="32" />
            <span>{{ auth.state.user.nickname }}</span>
          </RouterLink>
          <button class="header-icon logout-button" title="退出登录" @click="logout">
            <LogOut :size="18" />
          </button>
        </template>
        <template v-else>
          <RouterLink class="text-link" to="/login">登录</RouterLink>
          <RouterLink class="button button-small" to="/login?tab=register">
            加入校园
          </RouterLink>
        </template>
        <button
          class="mobile-menu-button"
          :aria-label="mobileOpen ? '关闭菜单' : '打开菜单'"
          @click="mobileOpen = !mobileOpen"
        >
          <X v-if="mobileOpen" :size="22" />
          <Menu v-else :size="22" />
        </button>
      </div>
    </div>

    <Transition name="menu">
      <nav v-if="mobileOpen" class="mobile-nav">
        <RouterLink
          v-for="item in nav"
          :key="item.to"
          :to="item.to"
          @click="mobileOpen = false"
        >
          {{ item.label }}
        </RouterLink>
        <RouterLink v-if="auth.state.user" to="/profile" @click="mobileOpen = false">
          <UserRound :size="18" /> 个人中心
        </RouterLink>
        <RouterLink
          v-if="auth.state.user?.role === 'admin'"
          to="/admin"
          @click="mobileOpen = false"
        >
          <Settings :size="18" /> 管理后台
        </RouterLink>
      </nav>
    </Transition>
  </header>
</template>
