<script setup lang="ts">
import { SearchX } from "lucide-vue-next";

withDefaults(
  defineProps<{
    title: string;
    text?: string;
  }>(),
  { text: "换个关键词试试，或成为第一个发布的人。" },
);
</script>

<template>
  <div class="empty-state">
    <span><SearchX :size="26" /></span>
    <h3>{{ title }}</h3>
    <p>{{ text }}</p>
    <slot />
  </div>
</template>

