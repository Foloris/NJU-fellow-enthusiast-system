<script setup lang="ts">
import { Home, MessageCircle, UserRound, Users } from "lucide-vue-next";
import AppHeader from "@/components/AppHeader.vue";
import ToastHost from "@/components/ToastHost.vue";
import { auth } from "@/stores/auth";
</script>

<template>
  <AppHeader />
  <main class="app-main">
    <RouterView />
  </main>
  <footer class="site-footer">
    <div>
      <strong>搭个伴</strong>
      <span>让兴趣相遇，让同行自然发生。</span>
    </div>
    <span>校园社区 · 理性交流 · 注意安全</span>
  </footer>

  <nav v-if="auth.state.user" class="bottom-nav">
    <RouterLink to="/">
      <Home :size="20" /><span>发现</span>
    </RouterLink>
    <RouterLink to="/mates">
      <Users :size="20" /><span>找搭子</span>
    </RouterLink>
    <RouterLink to="/messages">
      <MessageCircle :size="20" /><span>私信</span>
    </RouterLink>
    <RouterLink to="/profile">
      <UserRound :size="20" /><span>我的</span>
    </RouterLink>
  </nav>
  <ToastHost />
</template>

