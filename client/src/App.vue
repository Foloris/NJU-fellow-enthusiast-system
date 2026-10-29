<script setup lang="ts">
import { ref } from "vue";
import { Github, Home, MessageCircle, UserRound, Users } from "lucide-vue-next";
import AppHeader from "@/components/AppHeader.vue";
import FeedbackDialog from "@/components/FeedbackDialog.vue";
import HelpDialog from "@/components/HelpDialog.vue";
import ToastHost from "@/components/ToastHost.vue";
import { auth } from "@/stores/auth";

const repoUrl = "https://github.com/awencat/NJU-fellow-enthusiast-system";
const helpTopic = ref<"" | "faq" | "guide">("");
const feedbackType = ref<"problem" | "suggestion" | null>(null);

function openFeedbackFromHelp() {
  helpTopic.value = "";
  feedbackType.value = "problem";
}
</script>

<template>
  <AppHeader />
  <main class="app-main">
    <RouterView />
  </main>
  <footer class="site-footer">
    <div class="footer-inner">
      <div class="footer-brand">
        <span class="brand-mark" aria-hidden="true"><span /><span /></span>
        <span class="footer-brand-name">搭个伴</span>
      </div>
      <p class="footer-tagline">志趣相投 · 结伴同行</p>

      <div class="footer-main">
        <section class="footer-col">
          <h2 class="footer-heading">帮助</h2>
          <p class="footer-links">
            <button type="button" class="link-button" @click="helpTopic = 'faq'">常见问题</button>
            <button type="button" class="link-button" @click="helpTopic = 'guide'">使用说明</button>
            <button type="button" class="link-button" @click="feedbackType = 'problem'">问题反馈</button>
          </p>
        </section>
        <section class="footer-col">
          <h2 class="footer-heading">项目</h2>
          <p class="footer-links">
            <a :href="repoUrl">
              <Github :size="14" aria-hidden="true" />GitHub 仓库
            </a>
            <button type="button" class="link-button" @click="feedbackType = 'suggestion'">
              意见反馈
            </button>
          </p>
        </section>
      </div>

      <p class="footer-quote">同频的人，终会相遇</p>
      <p class="footer-copy">© 2026 同频项目组</p>
    </div>
  </footer>

  <HelpDialog :topic="helpTopic" @close="helpTopic = ''" @feedback="openFeedbackFromHelp" />
  <FeedbackDialog
    v-if="feedbackType"
    :open="!!feedbackType"
    :type="feedbackType"
    @close="feedbackType = null"
  />

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

