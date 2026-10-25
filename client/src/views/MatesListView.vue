<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { Plus, Search, Users } from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import type { MatePost, Paginated } from "@/types";
import MateCard from "@/components/MateCard.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import EmptyState from "@/components/EmptyState.vue";
import PaginationBar from "@/components/PaginationBar.vue";

const route = useRoute();
const router = useRouter();
const loading = ref(true);
const mates = ref<MatePost[]>([]);
const q = ref(String(route.query.q ?? ""));
const tag = ref(String(route.query.tag ?? ""));
const page = ref(Number(route.query.page) || 1);
const total = ref(0);
const pageSize = 9;
const popularTags = ["周末", "运动", "学习", "摄影", "户外", "长期", "今晚"];
const activeTag = computed(() => tag.value);

async function load() {
  loading.value = true;
  try {
    const params = new URLSearchParams({
      page: String(page.value),
      pageSize: String(pageSize),
    });
    if (q.value) params.set("q", q.value);
    if (tag.value) params.set("tag", tag.value);
    const data = await api<Paginated<MatePost>>(`/api/mates?${params}`);
    mates.value = data.items;
    total.value = data.total;
  } finally {
    loading.value = false;
  }
}

function updateTags(nextTag: string) {
  tag.value = nextTag;
  page.value = 1;
  router.replace({
    query: {
      ...(q.value ? { q: q.value } : {}),
      ...(tag.value ? { tag: tag.value } : {}),
    },
  });
  load();
}

function applySearch() {
  page.value = 1;
  router.replace({
    query: {
      ...(q.value ? { q: q.value } : {}),
      ...(tag.value ? { tag: tag.value } : {}),
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
    q.value = String(query.q ?? "");
    tag.value = String(query.tag ?? "");
    page.value = Number(query.page) || 1;
    load();
  },
);

onMounted(load);
</script>

<template>
  <div class="page-shell mates-page">
    <header class="page-heading split-heading mates-heading">
      <div>
        <span class="eyebrow"><Users :size="14" /> 搭子广场</span>
        <h1>想做的事，<em>一起去做吧。</em></h1>
        <p>找到时间合拍、兴趣相近的同学。提交申请，经过发布者同意后即可私聊。</p>
      </div>
      <RouterLink
        class="button warm-button"
        :to="auth.state.user ? '/mates/new' : '/login?redirect=/mates/new'"
      >
        <Plus :size="18" /> 发布搭子邀请
      </RouterLink>
    </header>

    <section class="mate-search-panel">
      <form class="search-field large" @submit.prevent="applySearch">
        <Search :size="20" />
        <input v-model="q" type="search" placeholder="搜索兴趣、活动或地点" />
        <button type="submit">搜索</button>
      </form>
      <div class="popular-tags">
        <button :class="{ active: !activeTag }" @click="updateTags('')">全部</button>
        <button
          v-for="item in popularTags"
          :key="item"
          :class="{ active: activeTag === item }"
          @click="updateTags(item)"
        >
          {{ item }}
        </button>
      </div>
    </section>

    <div class="list-summary">
      <strong>{{ total }} 个正在寻找同伴的邀请</strong>
      <span>请尊重彼此边界，线下见面优先选择公共场所</span>
    </div>

    <LoadingBlock v-if="loading" />
    <template v-else>
      <div v-if="mates.length" class="mate-grid mates-grid">
        <MateCard v-for="mate in mates" :key="mate.id" :mate="mate" />
      </div>
      <EmptyState
        v-else
        title="暂时没有匹配的搭子"
        text="换个关键词，或者发布一个属于你的邀请。"
      >
        <RouterLink class="button button-small" to="/mates/new">发布邀请</RouterLink>
      </EmptyState>
      <PaginationBar
        :page="page"
        :total="total"
        :page-size="pageSize"
        @change="changePage"
      />
    </template>
  </div>
</template>

