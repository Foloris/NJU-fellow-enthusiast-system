<script setup lang="ts">
import { onMounted, ref } from "vue";
import {
  Ban,
  CheckCircle2,
  Eye,
  EyeOff,
  FileWarning,
  FileText,
  Flag,
  LoaderCircle,
  MessageSquare,
  Plus,
  RotateCcw,
  Search,
  ShieldCheck,
  Trash2,
  UserRound,
  Users,
} from "lucide-vue-next";
import { api } from "@/api";
import { notify } from "@/stores/toast";
import Avatar from "@/components/Avatar.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import EmptyState from "@/components/EmptyState.vue";
import { formatDateTime } from "@/utils/date";

type Summary = {
  users: number;
  bannedUsers: number;
  exchangePosts: number;
  matePosts: number;
  pendingReports: number;
  activeConversations: number;
};

type ReportItem = {
  id: number;
  targetType: "post" | "mate" | "user" | "message";
  targetId: number;
  reason: string;
  details: string;
  snapshot: Record<string, unknown>;
  status: string;
  action: string | null;
  adminNote: string | null;
  createdAt: string;
  reporter: { id: number; nickname: string; username: string };
  messageContext: Array<{
    id: number;
    content: string;
    createdAt: string;
    senderId: number;
    senderNickname: string;
  }>;
};

type AdminUser = {
  id: number;
  username: string;
  nickname: string;
  avatar: string | null;
  bio: string;
  role: string;
  status: "active" | "banned";
  created_at: string;
  postCount: number;
};

type AdminContent = {
  id: number;
  title: string;
  content: string;
  kind?: string;
  status: "published" | "removed";
  createdAt: string;
  authorNickname: string;
  authorUsername: string;
};

const tab = ref<"reports" | "content" | "users">("reports");
const contentTab = ref<"exchange" | "mate">("exchange");
const reportStatus = ref("pending");
const summary = ref<Summary | null>(null);
const reports = ref<ReportItem[]>([]);
const content = ref<AdminContent[]>([]);
const users = ref<AdminUser[]>([]);
const loading = ref(true);
const search = ref("");
const processing = ref<number | null>(null);
const actionChoices = ref<Record<number, string>>({});
const notes = ref<Record<number, string>>({});

async function loadSummary() {
  summary.value = await api<Summary>("/api/admin/summary");
}

async function loadReports() {
  const data = await api<{ items: ReportItem[] }>(
    `/api/reports/admin?status=${reportStatus.value}`,
  );
  reports.value = data.items;
}

async function loadContent() {
  const params = new URLSearchParams({ type: contentTab.value });
  if (search.value) params.set("q", search.value);
  const data = await api<{ items: AdminContent[] }>(
    `/api/admin/content?${params}`,
  );
  content.value = data.items;
}

async function loadUsers() {
  const params = search.value ? `?q=${encodeURIComponent(search.value)}` : "";
  const data = await api<{ items: AdminUser[] }>(`/api/admin/users${params}`);
  users.value = data.items;
}

async function loadTab() {
  loading.value = true;
  try {
    if (tab.value === "reports") await loadReports();
    if (tab.value === "content") await loadContent();
    if (tab.value === "users") await loadUsers();
  } catch (error) {
    notify(error instanceof Error ? error.message : "后台数据加载失败", "error");
  } finally {
    loading.value = false;
  }
}

async function switchTab(next: typeof tab.value) {
  tab.value = next;
  search.value = "";
  await loadTab();
}

async function processReport(item: ReportItem, status: "resolved" | "dismissed") {
  processing.value = item.id;
  try {
    await api(`/api/admin/reports/${item.id}`, {
      method: "PATCH",
      body: JSON.stringify({
        status,
        action: actionChoices.value[item.id] || "none",
        adminNote: notes.value[item.id] || "",
      }),
    });
    notify(status === "resolved" ? "举报已处理" : "举报已驳回");
    await Promise.all([loadReports(), loadSummary()]);
  } catch (error) {
    notify(error instanceof Error ? error.message : "处理失败", "error");
  } finally {
    processing.value = null;
  }
}

async function toggleContent(item: AdminContent) {
  processing.value = item.id;
  try {
    const status = item.status === "published" ? "removed" : "published";
    await api(`/api/admin/content/${contentTab.value}/${item.id}`, {
      method: "PATCH",
      body: JSON.stringify({ status }),
    });
    item.status = status;
    notify(status === "removed" ? "内容已下架" : "内容已恢复");
  } catch (error) {
    notify(error instanceof Error ? error.message : "操作失败", "error");
  } finally {
    processing.value = null;
  }
}

async function deleteContent(item: AdminContent) {
  if (!window.confirm(`确定永久删除“${item.title}”吗？此操作无法恢复。`)) return;
  processing.value = item.id;
  try {
    await api(`/api/admin/content/${contentTab.value}/${item.id}`, {
      method: "DELETE",
    });
    content.value = content.value.filter((entry) => entry.id !== item.id);
    notify("内容已永久删除");
  } catch (error) {
    notify(error instanceof Error ? error.message : "删除失败", "error");
  } finally {
    processing.value = null;
  }
}

async function toggleUser(user: AdminUser) {
  const next = user.status === "active" ? "banned" : "active";
  if (
    next === "banned" &&
    !window.confirm(`封禁 ${user.nickname} 后，其帖子会被下架且无法登录。确定继续吗？`)
  ) {
    return;
  }
  processing.value = user.id;
  try {
    await api(`/api/admin/users/${user.id}`, {
      method: "PATCH",
      body: JSON.stringify({ status: next }),
    });
    user.status = next;
    notify(next === "banned" ? "账号已封禁" : "账号已解除封禁");
    await loadSummary();
  } catch (error) {
    notify(error instanceof Error ? error.message : "操作失败", "error");
  } finally {
    processing.value = null;
  }
}

function snapshotText(item: ReportItem) {
  const snapshot = item.snapshot;
  return String(snapshot.title || snapshot.details || snapshot.content || snapshot.nickname || snapshot.text || "无可展示内容");
}

async function init() {
  loading.value = true;
  try {
    await Promise.all([loadSummary(), loadReports()]);
  } catch (error) {
    notify(error instanceof Error ? error.message : "后台加载失败", "error");
  } finally {
    loading.value = false;
  }
}

onMounted(init);
</script>

<template>
  <div class="admin-page page-shell wide-shell">
    <header class="admin-heading">
      <div>
        <span class="eyebrow"><ShieldCheck :size="14" /> 管理后台</span>
        <h1>社区秩序，由每一次认真判断守护</h1>
        <p>审核举报、管理内容与账号，保持校园交流空间安全友善。</p>
      </div>
      <div class="admin-publish-actions">
        <RouterLink class="button secondary button-small" to="/exchange/new?kind=news">
          <Plus :size="15" /> 发布新闻
        </RouterLink>
        <RouterLink class="button button-small" to="/exchange/new">
          <FileText :size="15" /> 发布分享
        </RouterLink>
      </div>
    </header>

    <section v-if="summary" class="admin-stats">
      <article><Users :size="20" /><div><strong>{{ summary.users }}</strong><span>普通用户</span></div></article>
      <article>
        <FileText :size="20" /><div><strong>{{ summary.exchangePosts + summary.matePosts }}</strong><span>在线内容</span></div>
      </article>
      <article :class="{ alert: summary.pendingReports > 0 }">
        <Flag :size="20" /><div><strong>{{ summary.pendingReports }}</strong><span>待处理举报</span></div>
      </article>
      <article>
        <MessageSquare :size="20" /><div><strong>{{ summary.activeConversations }}</strong><span>私信会话</span></div>
      </article>
      <article>
        <Ban :size="20" /><div><strong>{{ summary.bannedUsers }}</strong><span>封禁账号</span></div>
      </article>
    </section>

    <div class="admin-tabs">
      <button :class="{ active: tab === 'reports' }" @click="switchTab('reports')">
        <Flag :size="17" /> 举报审核
        <span v-if="summary?.pendingReports">{{ summary.pendingReports }}</span>
      </button>
      <button :class="{ active: tab === 'content' }" @click="switchTab('content')">
        <FileWarning :size="17" /> 内容管理
      </button>
      <button :class="{ active: tab === 'users' }" @click="switchTab('users')">
        <UserRound :size="17" /> 用户管理
      </button>
    </div>

    <section class="admin-panel">
      <div v-if="tab === 'reports'" class="admin-panel-head">
        <div class="segmented-tabs">
          <button
            v-for="item in [
              { value: 'pending', label: '待处理' },
              { value: 'resolved', label: '已处理' },
              { value: 'dismissed', label: '已驳回' },
            ]"
            :key="item.value"
            :class="{ active: reportStatus === item.value }"
            @click="reportStatus = item.value; loadTab()"
          >
            {{ item.label }}
          </button>
        </div>
      </div>

      <div v-else-if="tab === 'content'" class="admin-panel-head">
        <div class="segmented-tabs">
          <button :class="{ active: contentTab === 'exchange' }" @click="contentTab = 'exchange'; loadTab()">交流内容</button>
          <button :class="{ active: contentTab === 'mate' }" @click="contentTab = 'mate'; loadTab()">搭子帖子</button>
        </div>
        <form class="search-field compact" @submit.prevent="loadTab">
          <Search :size="16" />
          <input v-model="search" type="search" placeholder="搜索标题或正文" />
          <button type="submit">搜索</button>
        </form>
      </div>

      <div v-else class="admin-panel-head">
        <strong>用户账号</strong>
        <form class="search-field compact" @submit.prevent="loadTab">
          <Search :size="16" />
          <input v-model="search" type="search" placeholder="搜索昵称或账号" />
          <button type="submit">搜索</button>
        </form>
      </div>

      <LoadingBlock v-if="loading" />

      <div v-else-if="tab === 'reports'" class="report-admin-list">
        <article v-for="item in reports" :key="item.id" class="report-admin-card">
          <div class="report-card-head">
            <span class="report-type">
              {{ item.targetType === "post" ? "交流帖" : item.targetType === "mate" ? "搭子帖" : item.targetType === "user" ? "用户" : "私信" }}
            </span>
            <span class="status-pill" :class="item.status">
              {{ item.status === "pending" ? "待审核" : item.status === "resolved" ? "已处理" : "已驳回" }}
            </span>
            <span class="muted">#{{ item.id }} · {{ formatDateTime(item.createdAt) }}</span>
          </div>
          <div class="report-reason">
            <Flag :size="17" />
            <strong>{{ item.reason }}</strong>
            <span>举报人：{{ item.reporter.nickname }}</span>
          </div>
          <p v-if="item.details" class="report-details">{{ item.details }}</p>
          <blockquote>{{ snapshotText(item) }}</blockquote>
          <div v-if="item.messageContext.length" class="message-context">
            <strong>被举报私信及相邻上下文</strong>
            <div
              v-for="message in item.messageContext"
              :key="message.id"
              :class="{ target: message.id === item.targetId }"
            >
              <span>{{ message.senderNickname }}</span>
              <p>{{ message.content }}</p>
            </div>
          </div>
          <div v-if="item.status === 'pending'" class="report-resolution">
            <select v-model="actionChoices[item.id]">
              <option value="none">不处理内容，仅记录</option>
              <option v-if="item.targetType === 'post' || item.targetType === 'mate'" value="remove_content">下架内容</option>
              <option v-if="item.targetType === 'post' || item.targetType === 'mate'" value="delete_content">删除内容</option>
              <option v-if="item.targetType === 'message'" value="delete_message">删除被举报私信</option>
              <option v-if="item.targetType !== 'message'" value="ban_user">封禁发布者/用户</option>
            </select>
            <input v-model="notes[item.id]" maxlength="1000" placeholder="处理备注（选填）" />
            <button class="button button-small" :disabled="processing === item.id" @click="processReport(item, 'resolved')">
              <CheckCircle2 :size="15" /> 确认处理
            </button>
            <button class="button secondary button-small" :disabled="processing === item.id" @click="processReport(item, 'dismissed')">
              驳回举报
            </button>
          </div>
          <p v-else-if="item.adminNote" class="admin-note">处理备注：{{ item.adminNote }}</p>
        </article>
        <EmptyState v-if="!reports.length" title="没有相关举报" text="当前分类下没有需要处理的记录。" />
      </div>

      <div v-else-if="tab === 'content'" class="admin-table">
        <div class="admin-table-head">
          <span>内容</span><span>作者</span><span>状态</span><span>操作</span>
        </div>
        <article v-for="item in content" :key="item.id">
          <div>
            <strong>{{ item.title }}</strong>
            <span>{{ item.content }}</span>
            <small>{{ formatDateTime(item.createdAt) }}</small>
          </div>
          <div>{{ item.authorNickname }}<small>@{{ item.authorUsername }}</small></div>
          <span class="status-pill" :class="item.status">{{ item.status === "published" ? "展示中" : "已下架" }}</span>
          <div class="table-actions">
            <button class="icon-button" :title="item.status === 'published' ? '下架' : '恢复'" @click="toggleContent(item)">
              <EyeOff v-if="item.status === 'published'" :size="17" />
              <Eye v-else :size="17" />
            </button>
            <button class="icon-button danger-text" title="永久删除" @click="deleteContent(item)">
              <Trash2 :size="17" />
            </button>
          </div>
        </article>
        <EmptyState v-if="!content.length" title="没有匹配的内容" />
      </div>

      <div v-else class="admin-table user-table">
        <div class="admin-table-head">
          <span>用户</span><span>账号状态</span><span>发布内容</span><span>操作</span>
        </div>
        <article v-for="user in users" :key="user.id">
          <div class="inline-user">
            <Avatar :user="user" :size="36" />
            <strong>{{ user.nickname }}<small>@{{ user.username }}</small></strong>
          </div>
          <span class="status-pill" :class="user.status">{{ user.status === "active" ? "正常" : "已封禁" }}</span>
          <span>{{ user.postCount }} 篇</span>
          <div class="table-actions">
            <button
              class="button button-small"
              :class="{ secondary: user.status === 'banned' }"
              :disabled="processing === user.id"
              @click="toggleUser(user)"
            >
              <Ban v-if="user.status === 'active'" :size="14" />
              <RotateCcw v-else :size="14" />
              {{ user.status === "active" ? "封禁" : "解除封禁" }}
            </button>
          </div>
        </article>
        <EmptyState v-if="!users.length" title="没有匹配的用户" />
      </div>
    </section>
  </div>
</template>

