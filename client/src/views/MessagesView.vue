<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  ref,
  watch,
} from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  ArrowLeft,
  Flag,
  MessageCircle,
  Send,
  ShieldCheck,
} from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { ChatMessage, Conversation } from "@/types";
import Avatar from "@/components/Avatar.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import EmptyState from "@/components/EmptyState.vue";
import ReportDialog from "@/components/ReportDialog.vue";
import { formatDateTime, relativeTime } from "@/utils/date";

type RealtimePayload = {
  type: "message" | "read" | "connected";
  conversationId?: number;
  message?: ChatMessage;
  readerId?: number;
};

const route = useRoute();
const router = useRouter();
const conversations = ref<Conversation[]>([]);
const messages = ref<ChatMessage[]>([]);
const loading = ref(true);
const loadingMessages = ref(false);
const draft = ref("");
const sending = ref(false);
const chatBody = ref<HTMLElement | null>(null);
const selectedId = ref<number | null>(
  route.query.c ? Number(route.query.c) : null,
);
const socket = ref<WebSocket | null>(null);
const reconnectTimer = ref<number | null>(null);
const reportOpen = ref(false);
const reportMessageId = ref(0);

const selected = computed(() =>
  conversations.value.find((item) => item.id === selectedId.value),
);

function scrollToBottom() {
  nextTick(() => {
    if (chatBody.value) chatBody.value.scrollTop = chatBody.value.scrollHeight;
  });
}

async function loadConversations(keepSelection = true) {
  const data = await api<{ items: Conversation[] }>("/api/messages");
  conversations.value = data.items;
  if (!keepSelection || !selectedId.value) {
    selectedId.value = data.items[0]?.id ?? null;
  }
  if (
    selectedId.value &&
    !data.items.some((item) => item.id === selectedId.value)
  ) {
    selectedId.value = data.items[0]?.id ?? null;
  }
}

async function openConversation(id: number) {
  selectedId.value = id;
  router.replace({ query: { c: id } });
  loadingMessages.value = true;
  try {
    const data = await api<{ items: ChatMessage[] }>(
      `/api/messages/${id}/messages`,
    );
    messages.value = data.items;
    const conversation = conversations.value.find((item) => item.id === id);
    if (conversation) conversation.unreadCount = 0;
    await api(`/api/messages/${id}/read`, { method: "POST" });
    scrollToBottom();
  } catch (error) {
    notify(error instanceof Error ? error.message : "消息加载失败", "error");
  } finally {
    loadingMessages.value = false;
  }
}

async function initialize() {
  loading.value = true;
  try {
    await loadConversations();
    if (selectedId.value) await openConversation(selectedId.value);
  } catch (error) {
    notify(error instanceof Error ? error.message : "会话加载失败", "error");
  } finally {
    loading.value = false;
  }
  connectSocket();
}

function connectSocket() {
  if (socket.value?.readyState === WebSocket.OPEN) return;
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
  socket.value = ws;
  ws.onmessage = async (event) => {
    const payload = JSON.parse(event.data) as RealtimePayload;
    if (payload.type === "message" && payload.message && payload.conversationId) {
      if (payload.conversationId === selectedId.value) {
        if (!messages.value.some((item) => item.id === payload.message!.id)) {
          messages.value.push(payload.message);
          scrollToBottom();
        }
        if (payload.message.sender.id !== auth.state.user?.id) {
          await api(`/api/messages/${payload.conversationId}/read`, {
            method: "POST",
          }).catch(() => undefined);
        }
      }
      await loadConversations(true).catch(() => undefined);
    }
    if (payload.type === "read") {
      messages.value.forEach((message) => {
        if (message.sender.id === auth.state.user?.id) {
          message.readAt ||= new Date().toISOString();
        }
      });
    }
  };
  ws.onclose = () => {
    socket.value = null;
    if (reconnectTimer.value) window.clearTimeout(reconnectTimer.value);
    reconnectTimer.value = window.setTimeout(connectSocket, 2500);
  };
}

async function sendMessage() {
  if (!selectedId.value || !draft.value.trim() || sending.value) return;
  const content = draft.value.trim();
  sending.value = true;
  try {
    const data = await api<{ message: ChatMessage }>(
      `/api/messages/${selectedId.value}/messages`,
      {
        method: "POST",
        body: JSON.stringify({ content }),
      },
    );
    if (!messages.value.some((item) => item.id === data.message.id)) {
      messages.value.push(data.message);
    }
    draft.value = "";
    scrollToBottom();
    await loadConversations(true);
  } catch (error) {
    notify(error instanceof Error ? error.message : "消息发送失败", "error");
  } finally {
    sending.value = false;
  }
}

function openReport(message: ChatMessage) {
  reportMessageId.value = message.id;
  reportOpen.value = true;
}

function backToList() {
  selectedId.value = null;
  messages.value = [];
  router.replace({ query: {} });
}

watch(
  () => route.query.c,
  (value) => {
    const id = Number(value);
    if (id && id !== selectedId.value) openConversation(id);
  },
);

onMounted(initialize);
onBeforeUnmount(() => {
  if (reconnectTimer.value) window.clearTimeout(reconnectTimer.value);
  socket.value?.close();
});
</script>

<template>
  <div class="messages-page page-shell wide-shell">
    <header class="page-heading compact-heading">
      <span class="eyebrow">私信</span>
      <h1>把同行的约定，继续聊下去</h1>
    </header>

    <LoadingBlock v-if="loading" />
    <EmptyState
      v-else-if="!conversations.length"
      title="还没有私信会话"
      text="申请搭子并通过后，聊天会出现在这里。"
    >
      <RouterLink class="button button-small" to="/mates">去找搭子</RouterLink>
    </EmptyState>
    <div v-else class="chat-shell" :class="{ 'chat-has-selection': selectedId }">
      <aside class="conversation-list">
        <div class="conversation-head">
          <div>
            <strong>全部会话</strong>
            <span>{{ conversations.length }} 个</span>
          </div>
          <ShieldCheck :size="20" title="仅有已通过的搭子可以私聊" />
        </div>
        <button
          v-for="conversation in conversations"
          :key="conversation.id"
          class="conversation-item"
          :class="{ active: conversation.id === selectedId }"
          @click="openConversation(conversation.id)"
        >
          <Avatar :user="conversation.otherUser" :size="44" />
          <div class="conversation-copy">
            <div>
              <strong>{{ conversation.otherUser.nickname }}</strong>
              <span>{{ relativeTime(conversation.lastMessageAt) }}</span>
            </div>
            <p>{{ conversation.lastMessage }}</p>
            <small>来自：{{ conversation.postTitle }}</small>
          </div>
          <span v-if="conversation.unreadCount" class="unread-badge">
            {{ conversation.unreadCount > 99 ? "99+" : conversation.unreadCount }}
          </span>
        </button>
      </aside>

      <section v-if="selected" class="chat-panel">
        <header class="chat-header">
          <button class="chat-back" @click="backToList"><ArrowLeft :size="20" /></button>
          <Avatar :user="selected.otherUser" :size="38" />
          <div>
            <strong>{{ selected.otherUser.nickname }}</strong>
            <span>{{ selected.postTitle }}</span>
          </div>
          <span class="online-dot">实时消息</span>
        </header>

        <div ref="chatBody" class="chat-body">
          <LoadingBlock v-if="loadingMessages" text="正在加载消息…" />
          <div v-else class="chat-intro">
            <MessageCircle :size="22" />
            <p>你们的申请已通过。友善沟通，尊重彼此的时间与边界。</p>
          </div>
          <div
            v-for="message in messages"
            :key="message.id"
            class="message-row"
            :class="{ mine: message.sender.id === auth.state.user?.id }"
          >
            <Avatar
              v-if="message.sender.id !== auth.state.user?.id"
              :user="message.sender"
              :size="30"
            />
            <div class="message-bubble-wrap">
              <div class="message-bubble">{{ message.content }}</div>
              <div class="message-time">
                <span>{{ formatDateTime(message.createdAt) }}</span>
                <button
                  v-if="message.sender.id !== auth.state.user?.id"
                  title="举报这条消息"
                  @click="openReport(message)"
                >
                  <Flag :size="13" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <form class="chat-composer" @submit.prevent="sendMessage">
          <textarea
            v-model="draft"
            rows="1"
            maxlength="2000"
            placeholder="写一条消息…"
            @keydown.enter.exact.prevent="sendMessage"
          />
          <button :disabled="sending || !draft.trim()" aria-label="发送消息">
            <Send :size="19" />
          </button>
        </form>
      </section>

      <ReportDialog
        :open="reportOpen"
        target-type="message"
        :target-id="reportMessageId"
        @close="reportOpen = false"
      />
    </div>
  </div>
</template>

