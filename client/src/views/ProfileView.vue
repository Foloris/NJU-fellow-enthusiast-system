<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import {
  CalendarDays,
  ChevronRight,
  FileText,
  LogOut,
  MessageCircle,
  Save,
  Settings,
  UserRound,
} from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { ExchangePost, MatePost, Paginated } from "@/types";
import Avatar from "@/components/Avatar.vue";
import ImageUpload from "@/components/ImageUpload.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import EmptyState from "@/components/EmptyState.vue";
import PostCard from "@/components/PostCard.vue";
import MateCard from "@/components/MateCard.vue";

type ApplicationSummary = {
  id: number;
  message: string;
  status: "pending" | "accepted" | "rejected";
  createdAt: string;
  post: { id: number; title: string; status: string };
  applicantNickname: string;
  ownerNickname: string;
  conversationId: number | null;
};

const router = useRouter();
const activeTab = ref<"posts" | "mates" | "applications">("posts");
const editing = ref(false);
const loading = ref(true);
const saving = ref(false);
const posts = ref<ExchangePost[]>([]);
const mates = ref<MatePost[]>([]);
const sentApplications = ref<ApplicationSummary[]>([]);
const receivedApplications = ref<ApplicationSummary[]>([]);
const profileForm = reactive({
  nickname: auth.state.user?.nickname ?? "",
  bio: auth.state.user?.bio ?? "",
  avatar: auth.state.user?.avatar ?? null,
});
const allApplications = computed(() =>
  [...receivedApplications.value, ...sentApplications.value].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
  ),
);

async function load() {
  loading.value = true;
  try {
    const [myPosts, myMates, sent, received] = await Promise.all([
      api<Paginated<ExchangePost>>("/api/exchange/posts?mine=true&pageSize=50"),
      api<Paginated<MatePost>>("/api/mates?mine=true&pageSize=50"),
      api<{ items: ApplicationSummary[] }>("/api/mates/applications/mine?scope=sent"),
      api<{ items: ApplicationSummary[] }>("/api/mates/applications/mine?scope=received"),
    ]);
    posts.value = myPosts.items;
    mates.value = myMates.items;
    sentApplications.value = sent.items;
    receivedApplications.value = received.items;
  } catch (error) {
    notify(error instanceof Error ? error.message : "个人内容加载失败", "error");
  } finally {
    loading.value = false;
  }
}

async function saveProfile() {
  saving.value = true;
  try {
    const data = await api<{ user: typeof auth.state.user }>("/api/auth/me", {
      method: "PATCH",
      body: JSON.stringify(profileForm),
    });
    if (data.user) auth.updateUser(data.user);
    editing.value = false;
    notify("个人信息已更新");
  } catch (error) {
    notify(error instanceof Error ? error.message : "保存失败", "error");
  } finally {
    saving.value = false;
  }
}

async function logout() {
  await auth.logout();
  await router.replace({ name: "login" });
  notify("已安全退出");
}

onMounted(load);
</script>

<template>
  <div class="profile-page page-shell">
    <section v-if="auth.state.user" class="profile-hero">
      <div class="profile-identity">
        <Avatar :user="auth.state.user" :size="82" />
        <div>
          <span class="eyebrow">{{ auth.state.user.role === "admin" ? "社区管理员" : "校园同好" }}</span>
          <h1>{{ auth.state.user.nickname }}</h1>
          <p>@{{ auth.state.user.username }} · {{ auth.state.user.bio || "还没有填写个人简介" }}</p>
        </div>
      </div>
      <div class="profile-actions">
        <RouterLink v-if="auth.state.user.role === 'admin'" class="button secondary button-small" to="/admin">
          <Settings :size="15" /> 管理后台
        </RouterLink>
        <button class="button secondary button-small" @click="editing = !editing">
          {{ editing ? "收起编辑" : "编辑资料" }}
        </button>
        <button class="icon-button" title="退出登录" @click="logout"><LogOut :size="18" /></button>
      </div>
    </section>

    <form v-if="editing && auth.state.user" class="profile-edit-card stack-form" @submit.prevent="saveProfile">
      <div class="profile-edit-heading">
        <div><strong>编辑个人信息</strong><span>让你的校园名片更完整</span></div>
        <ImageUpload v-model="profileForm.avatar" />
      </div>
      <label>
        <span>昵称</span>
        <input v-model="profileForm.nickname" maxlength="30" required />
      </label>
      <label>
        <span>个人简介</span>
        <textarea v-model="profileForm.bio" rows="3" maxlength="200" placeholder="介绍一下你的兴趣或最近在做的事" />
      </label>
      <button class="button button-small" :disabled="saving">
        <Save :size="15" /> {{ saving ? "保存中…" : "保存资料" }}
      </button>
    </form>

    <div class="profile-stat-row">
      <div><strong>{{ posts.length }}</strong><span>交流内容</span></div>
      <div><strong>{{ mates.length }}</strong><span>搭子邀请</span></div>
      <div><strong>{{ sentApplications.length }}</strong><span>发出的申请</span></div>
      <div><strong>{{ receivedApplications.length }}</strong><span>收到的申请</span></div>
    </div>

    <div class="profile-tabs segmented-tabs">
      <button :class="{ active: activeTab === 'posts' }" @click="activeTab = 'posts'">
        <FileText :size="16" /> 我的分享
      </button>
      <button :class="{ active: activeTab === 'mates' }" @click="activeTab = 'mates'">
        <UserRound :size="16" /> 我的搭子
      </button>
      <button :class="{ active: activeTab === 'applications' }" @click="activeTab = 'applications'">
        <MessageCircle :size="16" /> 申请记录
      </button>
    </div>

    <LoadingBlock v-if="loading" />
    <template v-else>
      <template v-if="activeTab === 'posts'">
        <div v-if="posts.length" class="post-grid">
          <PostCard v-for="post in posts" :key="post.id" :post="post" />
        </div>
        <EmptyState v-else title="你还没有发布分享" text="记录一个校园片段，也许会遇见想法相近的人。">
          <RouterLink class="button button-small" to="/exchange/new">写一篇分享</RouterLink>
        </EmptyState>
      </template>

      <template v-if="activeTab === 'mates'">
        <div v-if="mates.length" class="mate-grid">
          <MateCard v-for="mate in mates" :key="mate.id" :mate="mate" />
        </div>
        <EmptyState v-else title="你还没有发布搭子邀请" text="把想做的事写下来，邀请同学一起。">
          <RouterLink class="button button-small" to="/mates/new">发布搭子邀请</RouterLink>
        </EmptyState>
      </template>

      <template v-if="activeTab === 'applications'">
        <div v-if="allApplications.length" class="application-summary-list">
          <article v-for="application in allApplications" :key="application.id">
            <div class="application-summary-icon">
              <CalendarDays :size="19" />
            </div>
            <div>
              <RouterLink :to="`/mates/${application.post.id}`">{{ application.post.title }}</RouterLink>
              <p>
                {{
                  receivedApplications.some((item) => item.id === application.id)
                    ? `收到 ${application.applicantNickname} 的申请`
                    : `由你向 ${application.ownerNickname} 发出`
                }}
              </p>
              <span class="status-pill" :class="application.status">
                {{ application.status === "pending" ? "等待回应" : application.status === "accepted" ? "已通过" : "未通过" }}
              </span>
            </div>
            <RouterLink
              v-if="application.conversationId"
              class="text-link"
              :to="`/messages?c=${application.conversationId}`"
            >
              私信 <ChevronRight :size="15" />
            </RouterLink>
          </article>
        </div>
        <EmptyState v-else title="还没有申请记录" text="找到感兴趣的搭子邀请，发出第一次同行申请。" />
      </template>
    </template>
  </div>
</template>

