<script setup lang="ts">
import { ChevronLeft, ChevronRight } from "lucide-vue-next";

const props = defineProps<{
  page: number;
  total: number;
  pageSize: number;
}>();
const emit = defineEmits<{ change: [page: number] }>();

const pages = () => Math.max(1, Math.ceil(props.total / props.pageSize));
</script>

<template>
  <div v-if="pages() > 1" class="pagination">
    <button
      class="icon-button"
      :disabled="page <= 1"
      aria-label="上一页"
      @click="emit('change', page - 1)"
    >
      <ChevronLeft :size="18" />
    </button>
    <span>第 {{ page }} / {{ pages() }} 页</span>
    <button
      class="icon-button"
      :disabled="page >= pages()"
      aria-label="下一页"
      @click="emit('change', page + 1)"
    >
      <ChevronRight :size="18" />
    </button>
  </div>
</template>

