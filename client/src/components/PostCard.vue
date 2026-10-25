<script setup lang="ts">
import { Bookmark, Heart, MessageCircle, Newspaper } from "lucide-vue-next";
import Avatar from "./Avatar.vue";
import type { ExchangePost } from "@/types";
import { relativeTime } from "@/utils/date";

defineProps<{
  post: ExchangePost;
  featured?: boolean;
}>();
</script>

<template>
  <article class="post-card" :class="{ 'post-card-featured': featured }">
    <RouterLink :to="`/exchange/${post.id}`" class="post-card-link">
      <div v-if="post.imageUrl" class="post-cover">
        <img :src="post.imageUrl" :alt="post.title" loading="lazy" />
        <span v-if="post.kind === 'news'" class="floating-badge">
          <Newspaper :size="13" /> 校园新闻
        </span>
      </div>
      <div v-else class="post-cover post-cover-placeholder">
        <span>“</span>
        <p>{{ post.kind === "news" ? "校园新鲜事" : "生活的小片段" }}</p>
      </div>
      <div class="post-card-body">
        <div class="post-kind-row">
          <span class="eyebrow">
            {{ post.kind === "news" ? "校园新闻" : "同好分享" }}
          </span>
          <span>{{ relativeTime(post.createdAt) }}</span>
        </div>
        <h3>{{ post.title }}</h3>
        <p>{{ post.content }}</p>
        <div class="post-author-row">
          <div class="inline-user">
            <Avatar :user="post.author" :size="28" />
            <span>{{ post.author.nickname }}</span>
          </div>
          <div class="post-metrics">
            <span :class="{ active: post.liked }">
              <Heart :size="15" /> {{ post.likeCount }}
            </span>
            <span>
              <MessageCircle :size="15" /> {{ post.commentCount }}
            </span>
            <span>
              <Bookmark :size="15" /> {{ post.favoriteCount }}
            </span>
          </div>
        </div>
      </div>
    </RouterLink>
  </article>
</template>

