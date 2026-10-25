<script setup lang="ts">
import { computed } from "vue";
import type { Author } from "@/types";

const props = withDefaults(
  defineProps<{
    user?: Pick<Author, "nickname" | "avatar"> | null;
    size?: number;
  }>(),
  { size: 40 },
);

const initial = computed(() => props.user?.nickname?.slice(0, 1) || "伴");
const colors = [
  ["#d3e5da", "#426b59"],
  ["#f1d9be", "#8d5b2e"],
  ["#d7deef", "#4b5c86"],
  ["#ead5d5", "#8b4d50"],
  ["#e7dfc7", "#7a6735"],
];
const color = computed(() => {
  const code = props.user?.nickname?.charCodeAt(0) || 0;
  return colors[code % colors.length];
});
</script>

<template>
  <img
    v-if="user?.avatar"
    class="avatar"
    :src="user.avatar"
    :alt="`${user.nickname}的头像`"
    :style="{ width: `${size}px`, height: `${size}px` }"
  />
  <span
    v-else
    class="avatar avatar-fallback"
    :style="{
      width: `${size}px`,
      height: `${size}px`,
      background: color[0],
      color: color[1],
      fontSize: `${Math.max(12, size * 0.38)}px`,
    }"
  >
    {{ initial }}
  </span>
</template>

