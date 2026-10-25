<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import {
  ArrowRight,
  CalendarDays,
  MessageCircleHeart,
  Search,
  Sparkles,
  Users,
} from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import type { ExchangePost, MatePost, Paginated } from "@/types";
import PostCard from "@/components/PostCard.vue";
import MateCard from "@/components/MateCard.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";
import EmptyState from "@/components/EmptyState.vue";

const router = useRouter();
const loading = ref(true);
const posts = ref<ExchangePost[]>([]);
const mates = ref<MatePost[]>([]);
const search = ref("");

async function loadHome() {
  loading.value = true;
  try {
    const [exchange, mate] = await Promise.all([
      api<Paginated<ExchangePost>>("/api/exchange/posts?pageSize=3"),
      api<Paginated<MatePost>>("/api/mates?pageSize=3"),
    ]);
    posts.value = exchange.items;
    mates.value = mate.items;
  } finally {
    loading.value = false;
  }
}

function searchMates() {
  router.push({ path: "/mates", query: search.value ? { q: search.value } : {} });
}

onMounted(loadHome);
</script>

<template>
  <div class="home-page">
    <section class="hero">
      <div class="hero-copy">
        <span class="hero-kicker">
          <Sparkles :size="15" />
          校园里的兴趣，在这里遇见回应
        </span>
        <h1>今天，想和谁<br /><em>一起做点什么？</em></h1>
        <p>
          分享校园里的新鲜与热爱，也找到愿意同路的人。一次散步、一场球，或是一段长期陪伴，都可以从一句“搭个伴”开始。
        </p>
        <form class="hero-search" @submit.prevent="searchMates">
          <Search :size="19" />
          <input
            v-model="search"
            type="search"
            placeholder="想找什么搭子？比如：周末爬山、英语口语"
            aria-label="搜索搭子"
          />
          <button type="submit">开始寻找</button>
        </form>
        <div class="hero-quick-links">
          <span>大家都在找</span>
          <button @click="router.push('/mates?tag=周末')">周末出行</button>
          <button @click="router.push('/mates?tag=学习')">学习搭子</button>
          <button @click="router.push('/mates?tag=运动')">一起运动</button>
        </div>
      </div>

      <div class="hero-collage" aria-hidden="true">
        <div class="hero-photo hero-photo-main">
          <img src="/demo/autumn-walk.svg" alt="" />
          <span>一起去看秋天</span>
        </div>
        <div class="hero-note">
          <span class="hero-note-pin"></span>
          <strong>新朋友，从同好开始。</strong>
          <small>—— 校园同好站</small>
        </div>
        <div class="hero-photo hero-photo-small">
          <img src="/demo/badminton.svg" alt="" />
        </div>
        <div class="hero-float-card">
          <div><Users :size="16" /> 正在招募</div>
          <strong>周末去看展</strong>
          <span>2 位同学已申请</span>
        </div>
      </div>
    </section>

    <section class="home-stats">
      <div>
        <strong>同好交流</strong>
        <span>发现日常里的小小共鸣</span>
      </div>
      <div>
        <strong>兴趣搭子</strong>
        <span>让想做的事不再落单</span>
      </div>
      <div>
        <strong>安全私聊</strong>
        <span>申请通过后开启对话</span>
      </div>
      <div>
        <strong>校园社区</strong>
        <span>共建友善、可信的空间</span>
      </div>
    </section>

    <LoadingBlock v-if="loading" />

    <template v-else>
      <section class="section-block">
        <div class="section-heading">
          <div>
            <span class="eyebrow">同好正在发生</span>
            <h2>校园里，值得停一停的分享</h2>
          </div>
          <RouterLink class="text-link arrow-link" to="/exchange">
            查看全部 <ArrowRight :size="17" />
          </RouterLink>
        </div>
        <div v-if="posts.length" class="post-grid">
          <PostCard
            v-for="(post, index) in posts"
            :key="post.id"
            :post="post"
            :featured="index === 0"
          />
        </div>
        <EmptyState v-else title="交流区还是空的" />
      </section>

      <section class="mate-callout">
        <div class="mate-callout-copy">
          <span class="eyebrow"><MessageCircleHeart :size="14" /> 找人一起</span>
          <h2>不必等所有条件都刚刚好，<br />先发出邀请吧。</h2>
          <p>
            发布你的时间、地点和期待。感兴趣的同学会提交申请，你同意后再开始私聊。
          </p>
          <RouterLink
            class="button"
            :to="auth.state.user ? '/mates/new' : '/login?redirect=/mates/new'"
          >
            发布一个搭子邀请 <ArrowRight :size="17" />
          </RouterLink>
        </div>
        <div class="mate-callout-list">
          <article>
            <CalendarDays :size="20" />
            <div><strong>说清楚时间</strong><span>让彼此更好安排</span></div>
          </article>
          <article>
            <Users :size="20" />
            <div><strong>友善地回应</strong><span>尊重每一次申请</span></div>
          </article>
          <article>
            <MessageCircleHeart :size="20" />
            <div><strong>通过后私聊</strong><span>把线上默契带到线下</span></div>
          </article>
        </div>
      </section>

      <section class="section-block">
        <div class="section-heading">
          <div>
            <span class="eyebrow">等待同行的人</span>
            <h2>最近发布的搭子邀请</h2>
          </div>
          <RouterLink class="text-link arrow-link" to="/mates">
            全部搭子 <ArrowRight :size="17" />
          </RouterLink>
        </div>
        <div v-if="mates.length" class="mate-grid">
          <MateCard v-for="mate in mates" :key="mate.id" :mate="mate" />
        </div>
        <EmptyState v-else title="还没有搭子邀请" text="成为第一个发出邀请的人吧。" />
      </section>
    </template>
  </div>
</template>

