<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowLeft,
  Bookmark,
  Flag,
  Heart,
  MessageCircle,
  Pencil,
  Send,
  Share2,
  Trash2,
} from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { Comment, ExchangePost } from "@/types";
import Avatar from "@/components/Avatar.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import ReportDialog from "@/components/ReportDialog.vue";
import { formatDateTime } from "@/utils/date";

const route = useRoute();
const router = useRouter();
const loading = ref(true);
const post = ref<ExchangePost | null>(null);
const comments = ref<Comment[]>([]);
const commentText = ref("");
const reportOpen = ref(false);
const sendingComment = ref(false);

const canManage = computed(
  () =>
    post.value?.author.id === auth.state.user?.id ||
    auth.state.user?.role === "admin",
);

async function load() {
  loading.value = true;
  try {
    const data = await api<{ post: ExchangePost; comments: Comment[] }>(
      `/api/exchange/posts/${route.params.id}`,
    );
    post.value = data.post;
    comments.value = data.comments;
  } catch (error) {
    notify(error instanceof Error ? error.message : "帖子加载失败", "error");
    router.replace("/exchange");
  } finally {
    loading.value = false;
  }
}

function requireLogin() {
  if (auth.state.user) return true;
  router.push({ name: "login", query: { redirect: route.fullPath } });
  return false;
}

async function toggleLike() {
  if (!requireLogin() || !post.value) return;
  try {
    const data = await api<{ active: boolean; count: number }>(
      `/api/exchange/posts/${post.value.id}/like`,
      { method: "POST" },
    );
    post.value.liked = data.active;
    post.value.likeCount = data.count;
  } catch (error) {
    notify(error instanceof Error ? error.message : "操作失败", "error");
  }
}

async function toggleFavorite() {
  if (!requireLogin() || !post.value) return;
  try {
    const data = await api<{ active: boolean; count: number }>(
      `/api/exchange/posts/${post.value.id}/favorite`,
      { method: "POST" },
    );
    post.value.favorited = data.active;
    post.value.favoriteCount = data.count;
    notify(data.active ? "已收藏这篇分享" : "已取消收藏", "info");
  } catch (error) {
    notify(error instanceof Error ? error.message : "操作失败", "error");
  }
}

async function addComment() {
  if (!requireLogin() || !post.value || !commentText.value.trim()) return;
  sendingComment.value = true;
  try {
    const data = await api<{ comment: Comment }>(
      `/api/exchange/posts/${post.value.id}/comments`,
      {
        method: "POST",
        body: JSON.stringify({ content: commentText.value }),
      },
    );
    comments.value.push(data.comment);
    post.value.commentCount += 1;
    commentText.value = "";
    notify("评论已发布");
  } catch (error) {
    notify(error instanceof Error ? error.message : "评论失败", "error");
  } finally {
    sendingComment.value = false;
  }
}

async function removeComment(comment: Comment) {
  if (!window.confirm("确定删除这条评论吗？")) return;
  try {
    await api(`/api/exchange/comments/${comment.id}`, { method: "DELETE" });
    comments.value = comments.value.filter((item) => item.id !== comment.id);
    if (post.value) post.value.commentCount -= 1;
    notify("评论已删除");
  } catch (error) {
    notify(error instanceof Error ? error.message : "删除失败", "error");
  }
}

async function removePost() {
  if (!post.value || !window.confirm("删除后无法恢复，确定删除这篇内容吗？")) return;
  try {
    await api(`/api/exchange/posts/${post.value.id}`, { method: "DELETE" });
    notify("内容已删除");
    router.push("/exchange");
  } catch (error) {
    notify(error instanceof Error ? error.message : "删除失败", "error");
  }
}

async function sharePost() {
  try {
    await navigator.clipboard.writeText(window.location.href);
    notify("页面链接已复制");
  } catch {
    notify("浏览器未允许复制链接", "error");
  }
}

function openReport() {
  if (!requireLogin()) return;
  reportOpen.value = true;
}

watch(() => route.params.id, load);
onMounted(load);
</script>

<template>
  <div class="detail-page page-shell">
    <LoadingBlock v-if="loading" />
    <template v-else-if="post">
      <button class="back-link" @click="router.back()"><ArrowLeft :size="17" /> 返回</button>
      <article class="exchange-article">
        <header class="article-header">
          <div class="article-meta">
            <span class="eyebrow">{{ post.kind === "news" ? "校园新闻" : "同好分享" }}</span>
            <span>{{ formatDateTime(post.createdAt) }}</span>
            <span v-if="post.status === 'removed'" class="status-pill removed">已下架</span>
          </div>
          <h1>{{ post.title }}</h1>
          <div class="article-author">
            <Avatar :user="post.author" :size="44" />
            <div>
              <strong>{{ post.author.nickname }}</strong>
              <small>@{{ post.author.username }}</small>
            </div>
          </div>
        </header>

        <img v-if="post.imageUrl" class="article-cover" :src="post.imageUrl" :alt="post.title" />
        <div class="article-content">{{ post.content }}</div>

        <footer class="article-actions">
          <button :class="{ active: post.liked }" @click="toggleLike">
            <Heart :size="19" :fill="post.liked ? 'currentColor' : 'none'" />
            {{ post.likeCount }} 喜欢
          </button>
          <button :class="{ active: post.favorited }" @click="toggleFavorite">
            <Bookmark :size="19" :fill="post.favorited ? 'currentColor' : 'none'" />
            {{ post.favorited ? "已收藏" : "收藏" }}
          </button>
          <button @click="sharePost"><Share2 :size="19" /> 分享</button>
          <button class="action-report" @click="openReport"><Flag :size="18" /> 举报</button>
        </footer>
      </article>

      <aside v-if="canManage" class="owner-actions">
        <span>作者管理</span>
        <RouterLink class="button secondary button-small" :to="`/exchange/${post.id}/edit`">
          <Pencil :size="15" /> 编辑
        </RouterLink>
        <button class="button secondary danger-text button-small" @click="removePost">
          <Trash2 :size="15" /> 删除
        </button>
      </aside>

      <section class="comments-section">
        <div class="section-heading small-heading">
          <div>
            <span class="eyebrow">交流区</span>
            <h2><MessageCircle :size="20" /> {{ comments.length }} 条评论</h2>
          </div>
        </div>
        <form v-if="auth.state.user" class="comment-composer" @submit.prevent="addComment">
          <Avatar :user="auth.state.user" :size="36" />
          <textarea
            v-model="commentText"
            rows="2"
            maxlength="500"
            placeholder="留下你的回应…"
          />
          <button class="button button-small" :disabled="sendingComment || !commentText.trim()">
            <Send :size="15" /> 发送
          </button>
        </form>
        <RouterLink v-else class="login-inline" :to="{ name: 'login', query: { redirect: route.fullPath } }">
          登录后参与评论
        </RouterLink>

        <div v-if="comments.length" class="comment-list">
          <article v-for="comment in comments" :key="comment.id" class="comment-item">
            <Avatar :user="comment.author" :size="36" />
            <div>
              <div class="comment-head">
                <strong>{{ comment.author.nickname }}</strong>
                <span>{{ formatDateTime(comment.createdAt) }}</span>
                <button
                  v-if="comment.author.id === auth.state.user?.id || auth.state.user?.role === 'admin'"
                  class="text-danger"
                  @click="removeComment(comment)"
                >
                  删除
                </button>
              </div>
              <p>{{ comment.content }}</p>
            </div>
          </article>
        </div>
        <p v-else class="empty-inline">还没有评论，来留下第一句回应吧。</p>
      </section>

      <ReportDialog
        :open="reportOpen"
        target-type="post"
        :target-id="post.id"
        @close="reportOpen = false"
      />
    </template>
  </div>
</template>

