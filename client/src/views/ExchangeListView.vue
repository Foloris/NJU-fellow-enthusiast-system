<script setup lang="ts">
import { onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Newspaper, PenLine, Search, Sparkles } from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import type { ExchangePost, Paginated } from "@/types";
import PostCard from "@/components/PostCard.vue";
import EmptyState from "@/components/EmptyState.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import PaginationBar from "@/components/PaginationBar.vue";

const route = useRoute();
const router = useRouter();
const posts = ref<ExchangePost[]>([]);
const loading = ref(true);
const q = ref(String(route.query.q ?? ""));
const kind = ref(String(route.query.kind ?? ""));
const page = ref(Number(route.query.page) || 1);
const total = ref(0);
const pageSize = 9;

async function load() {
  loading.value = true;
  try {
    const params = new URLSearchParams({
      page: String(page.value),
      pageSize: String(pageSize),
    });
    if (q.value) params.set("q", q.value);
    if (kind.value) params.set("kind", kind.value);
    const data = await api<Paginated<ExchangePost>>(
      `/api/exchange/posts?${params}`,
    );
    posts.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

function applyFilters() {
  page.value = 1;
  router.replace({
    query: {
      ...(q.value ? { q: q.value } : {}),
      ...(kind.value ? { kind: kind.value } : {}),
    },
  });
  load();
}

function changePage(value: number) {
  page.value = value;
  router.replace({ query: { ...route.query, page: value } });
  load();
  window.scrollTo({ top: 0, behavior: "smooth" });
}

watch(
  () => route.query,
  (query) => {
    if (String(query.q ?? "") === q.value && String(query.kind ?? "") === kind.value) {
      return;
    }
    q.value = String(query.q ?? "");
    kind.value = String(query.kind ?? "");
    page.value = Number(query.page) || 1;
    load();
  },
);

onMounted(load);
</script>

<template>
  <div class="page-shell">
    <header class="page-heading split-heading">
      <div>
        <span class="eyebrow"><Sparkles :size="14" /> 同好交流</span>
        <h1>发现共鸣，也分享你的日常</h1>
        <p>一篇随手记录、一场校园活动，都可能成为认识新朋友的起点。</p>
      </div>
      <RouterLink
        class="button"
        :to="auth.state.user ? '/exchange/new' : '/login?redirect=/exchange/new'"
      >
        <PenLine :size="17" /> 写一篇分享
      </RouterLink>
    </header>

    <div class="filter-bar">
      <div class="segmented-tabs">
        <button :class="{ active: !kind }" @click="kind = ''; applyFilters()">
          全部
        </button>
        <button :class="{ active: kind === 'share' }" @click="kind = 'share'; applyFilters()">
          同好分享
        </button>
        <button :class="{ active: kind === 'news' }" @click="kind = 'news'; applyFilters()">
          <Newspaper :size="15" /> 校园新闻
        </button>
      </div>
      <form class="search-field compact" @submit.prevent="applyFilters">
        <Search :size="17" />
        <input v-model="q" type="search" placeholder="搜索标题或正文" />
        <button type="submit">搜索</button>
      </form>
    </div>

    <LoadingBlock v-if="loading" />
    <template v-else>
      <div v-if="posts.length" class="post-grid exchange-grid">
        <PostCard v-for="post in posts" :key="post.id" :post="post" />
      </div>
      <EmptyState v-else title="没有找到相关内容" text="换一个关键词，或者分享你的第一篇校园故事。" />
      <PaginationBar
        :page="page"
        :total="total"
        :page-size="pageSize"
        @change="changePage"
      />
    </template>
  </div>
</template>

