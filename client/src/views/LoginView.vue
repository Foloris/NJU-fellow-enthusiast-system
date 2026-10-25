<script setup lang="ts">
import { computed, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowRight, Leaf, ShieldCheck, Users } from "lucide-vue-next";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";

const route = useRoute();
const router = useRouter();
const tab = ref(route.query.tab === "register" ? "register" : "login");
const submitting = ref(false);
const form = reactive({
  username: "",
  password: "",
  nickname: "",
});
const title = computed(() =>
  tab.value === "login" ? "欢迎回来，继续遇见同好" : "从一个校园身份开始",
);

watch(
  () => route.query.tab,
  (value) => {
    tab.value = value === "register" ? "register" : "login";
  },
);

function switchTab(value: string) {
  tab.value = value;
  router.replace({ query: { ...route.query, tab: value } });
}

async function submit() {
  submitting.value = true;
  try {
    if (tab.value === "login") {
      await auth.login(form.username, form.password);
      notify("欢迎回来");
    } else {
      await auth.register(form.username, form.password, form.nickname);
      notify("账号创建成功，欢迎加入");
    }
    const redirect =
      typeof route.query.redirect === "string" ? route.query.redirect : "/";
    router.push(redirect);
  } catch (error) {
    notify(error instanceof Error ? error.message : "操作失败", "error");
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <div class="auth-page">
    <section class="auth-story">
      <div class="auth-story-inner">
        <span class="hero-kicker"><Leaf :size="15" /> 搭个伴 · 校园同好社区</span>
        <h1>让每一个<br />“要不要一起？”<br /><em>都有回应。</em></h1>
        <p>同好交流、兴趣搭子、安全的私聊空间。从一个真诚的邀请开始。</p>
        <div class="auth-points">
          <span><Users :size="18" /> 申请通过后开启私聊</span>
          <span><ShieldCheck :size="18" /> 举报与社区管理机制</span>
        </div>
        <div class="auth-quote">
          “我们不是因为完完全全一样才相遇，而是愿意一起走一段。”
        </div>
      </div>
    </section>

    <section class="auth-form-wrap">
      <div class="auth-card">
        <div class="segmented-tabs auth-tabs">
          <button :class="{ active: tab === 'login' }" @click="switchTab('login')">登录</button>
          <button :class="{ active: tab === 'register' }" @click="switchTab('register')">注册</button>
        </div>
        <h2>{{ title }}</h2>
        <p class="muted">
          {{ tab === "login" ? "输入账号密码，继续上次的对话。" : "创建账号即可分享和寻找搭子。" }}
        </p>
        <form class="stack-form" @submit.prevent="submit">
          <label v-if="tab === 'register'">
            <span>昵称</span>
            <input v-model="form.nickname" maxlength="30" required placeholder="大家怎么称呼你？" />
          </label>
          <label>
            <span>账号</span>
            <input
              v-model="form.username"
              minlength="3"
              maxlength="24"
              required
              autocomplete="username"
              placeholder="中英文、数字或下划线"
            />
          </label>
          <label>
            <span>密码</span>
            <input
              v-model="form.password"
              type="password"
              minlength="8"
              maxlength="72"
              required
              :autocomplete="tab === 'login' ? 'current-password' : 'new-password'"
              placeholder="至少 8 位"
            />
          </label>
          <button class="button auth-submit" :disabled="submitting">
            {{ submitting ? "请稍候…" : tab === "login" ? "登录" : "创建账号" }}
            <ArrowRight :size="18" />
          </button>
        </form>
        <p v-if="tab === 'login'" class="demo-hint">
          本地演示账号：<code>linzhi / Demo123!</code>
        </p>
      </div>
    </section>
  </div>
</template>

