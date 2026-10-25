<script setup lang="ts">
import { LoaderCircle } from "lucide-vue-next";

withDefaults(defineProps<{ text?: string }>(), { text: "正在整理校园里的新动态…" });
</script>

<template>
  <div class="loading-block">
    <LoaderCircle class="spin" :size="25" />
    <span>{{ text }}</span>
  </div>
</template>

