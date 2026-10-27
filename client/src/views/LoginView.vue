<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowRight, Leaf, ShieldCheck, Users } from "lucide-vue-next";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { RegistrationConfig } from "@/types";

const route = useRoute();
const router = useRouter();
const tab = ref(route.query.tab === "register" ? "register" : "login");
const submitting = ref(false);
const confirmPasswordInput = ref<HTMLInputElement | null>(null);
const registrationConfig = ref<RegistrationConfig | null>(null);
const configError = ref("");
const emailSending = ref(false);
const resendAt = ref(0);
const codeExpiresAt = ref(0);
const sentEmail = ref("");
const clock = ref(Date.now());
let clockTimer: ReturnType<typeof setInterval> | undefined;
const form = reactive({
  username: "",
  password: "",
  confirmPassword: "",
  nickname: "",
  email: "",
  emailCode: "",
});
const normalizedEmail = computed(() => form.email.trim().toLowerCase());
const emailValid = computed(() => {
  const parts = normalizedEmail.value.split("@");
  return parts.length === 2 && parts[0].length > 0 &&
    !/\s/.test(normalizedEmail.value) && parts[1] === registrationConfig.value?.emailDomain;
});
const resendSeconds = computed(() => Math.max(0, Math.ceil((resendAt.value - clock.value) / 1000)));
const codeExpired = computed(() => codeExpiresAt.value > 0 && clock.value >= codeExpiresAt.value);
const canSendCode = computed(() => emailValid.value && registrationConfig.value?.emailDeliveryAvailable &&
  !emailSending.value && !submitting.value && resendSeconds.value === 0);
const title = computed(() =>
  tab.value === "login" ? "欢迎回来，继续遇见同好" : "从一个校园身份开始",
);
const passwordMismatch = computed(
  () => tab.value === "register" && form.confirmPassword.length > 0 &&
    form.password !== form.confirmPassword,
);

watch(normalizedEmail, () => { form.emailCode = ""; });

function syncClock() { clock.value = Date.now(); }

async function loadRegistrationConfig() {
  configError.value = "";
  try { registrationConfig.value = await auth.registrationConfig(); }
  catch { configError.value = "邮箱配置加载失败，请刷新页面重试。"; }
}

async function sendCode() {
  if (!canSendCode.value) return;
  emailSending.value = true;
  const recipient = normalizedEmail.value;
  try {
    const result = await auth.sendEmailCode(recipient);
    syncClock();
    resendAt.value = clock.value + result.retryAfter * 1000;
    codeExpiresAt.value = clock.value + result.expiresIn * 1000;
    sentEmail.value = recipient;
    form.emailCode = "";
    notify(result.message);
  } catch (error) {
    notify(error instanceof Error ? error.message : "验证码发送失败，请稍后重试", "error");
  } finally { emailSending.value = false; }
}

onMounted(() => {
  void loadRegistrationConfig();
  clockTimer = setInterval(syncClock, 1000);
  document.addEventListener("visibilitychange", syncClock);
});
onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer);
  document.removeEventListener("visibilitychange", syncClock);
});

watch(
  () => route.query.tab,
  (value) => {
    tab.value = value === "register" ? "register" : "login";
  },
);

function switchTab(value: string) {
  form.confirmPassword = "";
  tab.value = value;
  router.replace({ query: { ...route.query, tab: value } });
}

async function submit() {
  if (submitting.value || emailSending.value) return;
  if (tab.value === "register" && !emailValid.value) {
    notify("请填写允许后缀的校园邮箱", "error");
    return;
  }
  if (tab.value === "register" && form.password !== form.confirmPassword) {
    notify("两次输入的密码不一致，请重新确认", "error");
    confirmPasswordInput.value?.focus();
    return;
  }
  submitting.value = true;
  try {
    if (tab.value === "login") {
      await auth.login(form.username, form.password);
      notify("欢迎回来");
    } else {
      await auth.register(form.username, form.password, form.nickname, normalizedEmail.value, form.emailCode.trim());
      notify("账号创建成功，欢迎加入");
    }
    const requestedRedirect = route.query.redirect;
    const redirect = typeof requestedRedirect === "string" &&
      requestedRedirect.startsWith("/") && !requestedRedirect.startsWith("//") &&
      router.resolve(requestedRedirect).name !== "login" ? requestedRedirect : "/";
    await router.replace(redirect);
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
          <div v-if="tab === 'register'" class="registration-email">
            <label for="registration-email">校园邮箱</label>
            <div class="email-code-row">
              <input
                id="registration-email"
                v-model="form.email"
                type="email"
                maxlength="254"
                required
                autocomplete="email"
                :readonly="emailSending"
                :placeholder="registrationConfig ? `请输入 @${registrationConfig.emailDomain} 邮箱` : '请输入校园邮箱'"
                aria-describedby="email-domain-hint"
              />
              <button class="button secondary email-code-button" type="button" :disabled="!canSendCode" @click="sendCode">
                {{ emailSending ? "发送中…" : resendSeconds > 0 ? `${resendSeconds}秒后重发` : "发送验证码" }}
              </button>
            </div>
            <small id="email-domain-hint" class="email-help">
              {{ registrationConfig ? `目前仅支持 @${registrationConfig.emailDomain}，验证码有效期60秒。` : configError || "正在加载邮箱配置…" }}
            </small>
            <small v-if="registrationConfig && !registrationConfig.emailDeliveryAvailable" class="password-error" role="status">
              邮件发送服务尚未配置，请联系管理员。
            </small>
            <small v-if="sentEmail === normalizedEmail && codeExpiresAt" class="email-help" role="status">
              {{ codeExpired ? "验证码已过期，请重新发送。" : `验证码已发送至 ${sentEmail}，请检查收件箱或垃圾邮件。` }}
            </small>
          </div>
          <label v-if="tab === 'register'">
            <span>邮箱验证码</span>
            <input
              v-model="form.emailCode"
              inputmode="numeric"
              autocomplete="one-time-code"
              pattern="[0-9]{6}"
              minlength="6"
              maxlength="6"
              required
              placeholder="请输入6位邮箱验证码"
            />
          </label>
          <label v-if="tab === 'register'">
            <span>确认密码</span>
            <input
              ref="confirmPasswordInput"
              v-model="form.confirmPassword"
              type="password"
              minlength="8"
              maxlength="72"
              required
              autocomplete="new-password"
              placeholder="请再次输入密码"
              :aria-invalid="passwordMismatch ? 'true' : undefined"
              :aria-describedby="passwordMismatch ? 'confirm-password-error' : undefined"
            />
            <small v-if="passwordMismatch" id="confirm-password-error" class="password-error" role="alert">
              两次输入的密码不一致
            </small>
          </label>
          <button class="button auth-submit" :disabled="submitting || emailSending">
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

<style scoped>
.password-error {
  color: #b42318;
}
.registration-email { display: flex; flex-direction: column; gap: 7px; }
.email-code-row { display: flex; align-items: stretch; gap: 8px; }
.email-code-row input { flex: 1; min-width: 0; }
.email-code-button { flex-shrink: 0; padding: 10px 12px; white-space: nowrap; font-size: 12px; }
.email-help { color: var(--muted); font-size: 11px; font-weight: 400; overflow-wrap: anywhere; }
</style>

