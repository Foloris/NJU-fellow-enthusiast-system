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
    <section class="home-hero">
      <div class="home-hero-copy">
        <span class="home-hero-kicker">
          <Sparkles :size="15" />
          校园兴趣与活动
        </span>
        <h1>今天，想和谁<br /><em>一起做点什么？</em></h1>
        <p>
          发布活动计划，浏览校园动态，也可按时间和地点筛选同行邀请。
        </p>
        <form class="home-search" @submit.prevent="searchMates">
          <Search :size="19" />
          <input
            v-model="search"
            type="search"
            placeholder="想找什么搭子？比如：周末爬山、英语口语"
            aria-label="搜索搭子"
          />
          <button type="submit">开始寻找</button>
        </form>
        <div class="home-quick-links">
        <span>热门兴趣</span>
          <button @click="router.push('/mates?tag=周末')">周末出行</button>
          <button @click="router.push('/mates?tag=学习')">学习搭子</button>
          <button @click="router.push('/mates?tag=运动')">一起运动</button>
        </div>
      </div>

      <div class="home-hero-visual" aria-hidden="true">
        <div class="home-hero-photo">
          <img src="https://images.unsplash.com/photo-1470252649378-9c29740c9fa8?auto=format&fit=crop&w=1400&q=90" alt="" />
          <div class="home-hero-caption"><span>01</span><strong>秋日散步</strong><small>仙林校区 · 周末路线</small></div>
        </div>
        <div class="home-hero-stamp">FIND<br />YOUR<br />FELLOW</div>
        <div class="home-hero-mini">
          <img src="https://images.unsplash.com/photo-1546519638-68e109498ffc?auto=format&fit=crop&w=500&q=85" alt="" />
        </div>
        <div class="home-hero-float">
          <div><Users :size="16" /> 正在招募</div>
          <strong>周末去看展</strong>
          <span>2 位同学已申请</span>
        </div>
      </div>
    </section>

    <section class="home-index-list">
      <div>
        <strong>同好交流</strong>
        <span>发布动态，参与讨论</span>
      </div>
      <div>
        <strong>兴趣搭子</strong>
        <span>按兴趣和时间寻找活动</span>
      </div>
      <div>
        <strong>安全私聊</strong>
        <span>申请通过后开始聊天</span>
      </div>
      <div>
        <strong>校园社区</strong>
        <span>面向南大学生的兴趣社区</span>
      </div>
    </section>

    <LoadingBlock v-if="loading" />

    <template v-else>
      <section class="home-section home-section-featured">
        <div class="home-section-heading">
          <div>
            <span class="eyebrow">CAMPUS FEED / 01</span>
            <h2>校园动态</h2>
          </div>
          <RouterLink class="text-link arrow-link" to="/exchange">
            查看全部 <ArrowRight :size="17" />
          </RouterLink>
        </div>
        <div v-if="posts.length" class="home-post-grid">
          <PostCard
            v-for="(post, index) in posts"
            :key="post.id"
            :post="post"
            :featured="index === 0"
          />
        </div>
        <EmptyState v-else title="交流区还是空的" />
      </section>

      <section class="home-invite">
        <div class="home-invite-copy">
          <span class="eyebrow"><MessageCircleHeart :size="14" /> FIND A PARTNER / 02</span>
          <h2>发布一项活动计划</h2>
          <p>
            填写时间、地点和活动内容，等待感兴趣的同学申请。
          </p>
          <RouterLink
            class="button"
            :to="auth.state.user ? '/mates/new' : '/login?redirect=/mates/new'"
          >
            发布我的计划 <ArrowRight :size="17" />
          </RouterLink>
        </div>
        <div class="home-invite-list">
          <article>
            <CalendarDays :size="20" />
            <div><strong>周六 · 09:30</strong><span>紫金山慢走，缺 1 人</span></div>
          </article>
          <article>
            <Users :size="20" />
            <div><strong>周日 · 14:00</strong><span>图书馆口语练习，长期</span></div>
          </article>
          <article>
            <MessageCircleHeart :size="20" />
            <div><strong>今晚 · 19:00</strong><span>东区球馆双打，缺 1 人</span></div>
          </article>
        </div>
      </section>

      <section class="home-section home-section-mates">
        <div class="home-section-heading">
          <div>
            <span class="eyebrow">FIND A PARTNER / 03</span>
          <h2><span class="title-lead-char">搭</span>子广场</h2>
          </div>
          <RouterLink class="text-link arrow-link" to="/mates">
            全部搭子 <ArrowRight :size="17" />
          </RouterLink>
        </div>
        <div v-if="mates.length" class="home-mate-grid">
          <MateCard v-for="mate in mates" :key="mate.id" :mate="mate" />
        </div>
        <EmptyState v-else title="还没有搭子邀请" text="成为第一个发出邀请的人吧。" />
      </section>
    </template>
  </div>
</template>

