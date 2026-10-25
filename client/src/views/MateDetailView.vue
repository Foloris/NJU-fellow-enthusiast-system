<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowLeft,
  Check,
  Clock3,
  Flag,
  LockKeyhole,
  MapPin,
  MessageCircle,
  Pencil,
  Send,
  Trash2,
  UserCheck,
  Users,
  X,
} from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { Application, MatePost } from "@/types";
import Avatar from "@/components/Avatar.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import Modal from "@/components/Modal.vue";
import ReportDialog from "@/components/ReportDialog.vue";
import { formatDateTime } from "@/utils/date";

const route = useRoute();
const router = useRouter();
const loading = ref(true);
const mate = ref<MatePost | null>(null);
const applications = ref<Application[]>([]);
const applyOpen = ref(false);
const reportOpen = ref(false);
const applicationMessage = ref("");
const submitting = ref(false);
const processingId = ref<number | null>(null);

const canManage = computed(
  () =>
    mate.value?.author.id === auth.state.user?.id ||
    auth.state.user?.role === "admin",
);
const canApply = computed(
  () =>
    auth.state.user &&
    mate.value &&
    !mate.value.isOwner &&
    mate.value.status === "published" &&
    !mate.value.myApplicationStatus,
);

async function load() {
  loading.value = true;
  try {
    const data = await api<{ post: MatePost }>(`/api/mates/${route.params.id}`);
    mate.value = data.post;
    if (
      data.post.author.id === auth.state.user?.id ||
      auth.state.user?.role === "admin"
    ) {
      const applicationData = await api<{ items: Application[] }>(
        `/api/mates/${data.post.id}/applications`,
      );
      applications.value = applicationData.items;
    }
  } catch (error) {
    notify(error instanceof Error ? error.message : "搭子加载失败", "error");
    router.replace("/mates");
  } finally {
    loading.value = false;
  }
}

function openApply() {
  if (!auth.state.user) {
    router.push({ name: "login", query: { redirect: route.fullPath } });
    return;
  }
  applyOpen.value = true;
}

async function apply() {
  if (!mate.value || !applicationMessage.value.trim()) return;
  submitting.value = true;
  try {
    await api(`/api/mates/${mate.value.id}/apply`, {
      method: "POST",
      body: JSON.stringify({ message: applicationMessage.value }),
    });
    mate.value.myApplicationStatus = "pending";
    mate.value.applicationCount += 1;
    applyOpen.value = false;
    applicationMessage.value = "";
    notify("申请已发送，等待发布者回应");
  } catch (error) {
    notify(error instanceof Error ? error.message : "申请失败", "error");
  } finally {
    submitting.value = false;
  }
}

async function processApplication(application: Application, action: "accept" | "reject") {
  processingId.value = application.id;
  try {
    const data = await api<{ conversationId: number | null }>(
      `/api/mates/applications/${application.id}`,
      {
        method: "PATCH",
        body: JSON.stringify({ action }),
      },
    );
    application.status = action === "accept" ? "accepted" : "rejected";
    application.conversationId = data.conversationId;
    notify(action === "accept" ? "已同意申请，可以开始私聊" : "已拒绝这条申请");
  } catch (error) {
    notify(error instanceof Error ? error.message : "处理失败", "error");
  } finally {
    processingId.value = null;
  }
}

async function removeMate() {
  if (!mate.value || !window.confirm("删除后无法恢复，确定删除这个搭子邀请吗？")) return;
  try {
    await api(`/api/mates/${mate.value.id}`, { method: "DELETE" });
    notify("搭子邀请已删除");
    router.push("/mates");
  } catch (error) {
    notify(error instanceof Error ? error.message : "删除失败", "error");
  }
}

function openReport() {
  if (!auth.state.user) {
    router.push({ name: "login", query: { redirect: route.fullPath } });
    return;
  }
  reportOpen.value = true;
}

watch(() => route.params.id, load);
onMounted(load);
</script>

<template>
  <div class="mate-detail-page page-shell">
    <LoadingBlock v-if="loading" />
    <template v-else-if="mate">
      <button class="back-link" @click="router.back()"><ArrowLeft :size="17" /> 返回搭子广场</button>

      <div class="mate-detail-layout">
        <article class="mate-detail-main">
          <header>
            <div class="tag-row">
              <span v-for="tag in mate.tags" :key="tag" class="tag">{{ tag }}</span>
              <span v-if="mate.status === 'removed'" class="status-pill removed">已下架</span>
            </div>
            <h1>{{ mate.title }}</h1>
            <div class="mate-author-line">
              <Avatar :user="mate.author" :size="42" />
              <div>
                <strong>{{ mate.author.nickname }}</strong>
                <span>@{{ mate.author.username }} · 发布于 {{ formatDateTime(mate.createdAt) }}</span>
              </div>
            </div>
          </header>

          <img v-if="mate.imageUrl" class="mate-detail-cover" :src="mate.imageUrl" :alt="mate.title" />

          <div class="mate-detail-copy">
            <span class="eyebrow">邀请详情</span>
            <p>{{ mate.details }}</p>
          </div>

          <div class="mate-info-grid">
            <div>
              <Clock3 :size="20" />
              <span>持续时间</span>
              <strong>{{ mate.duration }}</strong>
            </div>
            <div>
              <MapPin :size="20" />
              <span>见面地点</span>
              <strong>{{ mate.location || "线上 / 待商议" }}</strong>
            </div>
            <div>
              <Users :size="20" />
              <span>当前申请</span>
              <strong>{{ mate.applicationCount }} 人</strong>
            </div>
          </div>

          <section v-if="mate.notes" class="mate-notes">
            <strong>补充备注</strong>
            <p>{{ mate.notes }}</p>
          </section>

          <section class="contact-card">
            <div>
              <LockKeyhole :size="18" />
              <strong>联系方式</strong>
            </div>
            <p v-if="mate.contact">{{ mate.contact }}</p>
            <p v-else>{{ mate.contactHint }}</p>
          </section>
        </article>

        <aside class="mate-apply-panel">
          <span class="eyebrow">想一起吗？</span>
          <h2>{{ mate.isOwner ? "管理你的搭子邀请" : "发出同行申请" }}</h2>

          <template v-if="mate.isOwner || auth.state.user?.role === 'admin'">
            <p>只有你同意后，申请人才会进入私聊并看到联系方式。</p>
            <RouterLink class="button secondary full-button" :to="`/mates/${mate.id}/edit`">
              <Pencil :size="16" /> 编辑邀请
            </RouterLink>
            <button class="button secondary full-button danger-text" @click="removeMate">
              <Trash2 :size="16" /> 删除邀请
            </button>
          </template>
          <template v-else-if="mate.myApplicationStatus === 'accepted'">
            <span class="status-callout success"><UserCheck :size="20" /> 申请已通过</span>
            <p>你们已经可以私聊，也可以查看上方联系方式。</p>
            <RouterLink
              v-if="applications.length === 0"
              class="button full-button"
              to="/messages"
            >
              <MessageCircle :size="17" /> 前往私信
            </RouterLink>
          </template>
          <template v-else-if="mate.myApplicationStatus === 'pending'">
            <span class="status-callout pending"><Clock3 :size="20" /> 等待发布者回应</span>
            <p>申请已经送达。对方同意后，你的私信页面会自动出现新会话。</p>
          </template>
          <template v-else-if="mate.myApplicationStatus === 'rejected'">
            <span class="status-callout"><X :size="20" /> 这次未能同行</span>
            <p>发布者没有接受这次申请，可以继续看看其他邀请。</p>
          </template>
          <template v-else>
            <p>用一小段话介绍自己，并说明为什么适合这次活动。发布者会收到你的申请。</p>
            <button class="button full-button warm-button" @click="openApply">
              <Send :size="17" /> {{ canApply ? "申请同行" : "登录后申请" }}
            </button>
          </template>
          <button class="text-button report-link" @click="openReport">
            <Flag :size="14" /> 举报这个邀请
          </button>
        </aside>
      </div>

      <section v-if="canManage" class="applications-section">
        <div class="section-heading small-heading">
          <div>
            <span class="eyebrow">待处理</span>
            <h2>收到的申请</h2>
          </div>
          <span class="muted">{{ applications.filter((item) => item.status === "pending").length }} 条待回应</span>
        </div>
        <div v-if="applications.length" class="application-list">
          <article v-for="application in applications" :key="application.id" class="application-card">
            <div class="application-head">
              <Avatar :user="application.applicant" :size="42" />
              <div>
                <strong>{{ application.applicant.nickname }}</strong>
                <span>@{{ application.applicant.username }}</span>
              </div>
              <span class="status-pill" :class="application.status">
                {{ application.status === "pending" ? "待处理" : application.status === "accepted" ? "已同意" : "已拒绝" }}
              </span>
            </div>
            <p>{{ application.message }}</p>
            <div class="application-actions">
              <RouterLink
                v-if="application.conversationId"
                class="button button-small"
                :to="`/messages?c=${application.conversationId}`"
              >
                <MessageCircle :size="15" /> 进入私聊
              </RouterLink>
              <template v-if="application.status === 'pending'">
                <button
                  class="button button-small"
                  :disabled="processingId === application.id"
                  @click="processApplication(application, 'accept')"
                >
                  <Check :size="15" /> 同意
                </button>
                <button
                  class="button secondary button-small"
                  :disabled="processingId === application.id"
                  @click="processApplication(application, 'reject')"
                >
                  拒绝
                </button>
              </template>
            </div>
          </article>
        </div>
        <p v-else class="empty-inline">还没有收到申请，分享出去让更多人看到吧。</p>
      </section>

      <Modal :open="applyOpen" title="申请一起同行" @close="applyOpen = false">
        <form class="stack-form" @submit.prevent="apply">
          <label>
            <span>给发布者的话</span>
            <textarea
              v-model="applicationMessage"
              rows="5"
              maxlength="500"
              required
              placeholder="简单介绍自己，说说你为什么想参加、时间是否合适…"
            />
          </label>
          <p class="form-note">你的申请会直接发送给发布者。对方同意后，才会建立私聊会话。</p>
          <button class="button warm-button" :disabled="submitting || !applicationMessage.trim()">
            {{ submitting ? "正在发送…" : "发送申请" }}
          </button>
        </form>
      </Modal>

      <ReportDialog
        :open="reportOpen"
        target-type="mate"
        :target-id="mate.id"
        @close="reportOpen = false"
      />
    </template>
  </div>
</template>

