<script setup lang="ts">
import { ArrowLeft } from "lucide-vue-next";
</script>

<template>
  <div class="not-found page-shell">
    <span>404</span>
    <h1>这条路上暂时没有同伴</h1>
    <p>页面可能已经被删除，或者地址写错了。</p>
    <RouterLink class="button" to="/"><ArrowLeft :size="17" /> 回到发现页</RouterLink>
  </div>
</template>

