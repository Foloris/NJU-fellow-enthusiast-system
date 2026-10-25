<script setup lang="ts">
import { ArrowUpRight, Clock3, MapPin, Users } from "lucide-vue-next";
import Avatar from "./Avatar.vue";
import type { MatePost } from "@/types";
import { relativeTime } from "@/utils/date";

defineProps<{ mate: MatePost }>();
</script>

<template>
  <article class="mate-card">
    <RouterLink :to="`/mates/${mate.id}`" class="mate-card-main">
      <div class="mate-card-top">
        <div class="tag-row">
          <span v-for="tag in mate.tags" :key="tag" class="tag">{{ tag }}</span>
        </div>
        <ArrowUpRight class="mate-arrow" :size="20" />
      </div>
      <h3>{{ mate.title }}</h3>
      <p class="mate-summary">{{ mate.details }}</p>
      <div class="mate-meta">
        <span v-if="mate.location">
          <MapPin :size="15" /> {{ mate.location }}
        </span>
        <span><Clock3 :size="15" /> {{ mate.duration }}</span>
      </div>
    </RouterLink>
    <div class="mate-card-footer">
      <div class="inline-user">
        <Avatar :user="mate.author" :size="30" />
        <div>
          <strong>{{ mate.author.nickname }}</strong>
          <small>{{ relativeTime(mate.createdAt) }}</small>
        </div>
      </div>
      <span class="application-count">
        <Users :size="15" /> {{ mate.applicationCount }} 人申请
      </span>
    </div>
  </article>
</template>

