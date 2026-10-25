<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowLeft, Save } from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { ExchangePost } from "@/types";
import ImageUpload from "@/components/ImageUpload.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";

const route = useRoute();
const router = useRouter();
const editId = computed(() =>
  route.name === "exchange-edit" ? Number(route.params.id) : null,
);
const loading = ref(Boolean(editId.value));
const submitting = ref(false);
const form = reactive({
  kind:
    route.query.kind === "news" && auth.state.user?.role === "admin"
      ? ("news" as const)
      : ("share" as const),
  title: "",
  content: "",
  imageUrl: null as string | null,
});

async function load() {
  if (!editId.value) return;
  try {
    const data = await api<{ post: ExchangePost }>(
      `/api/exchange/posts/${editId.value}`,
    );
    if (
      data.post.author.id !== auth.state.user?.id &&
      auth.state.user?.role !== "admin"
    ) {
      notify("只能编辑自己发布的内容", "error");
      router.replace("/exchange");
      return;
    }
    form.kind = data.post.kind;
    form.title = data.post.title;
    form.content = data.post.content;
    form.imageUrl = data.post.imageUrl;
  } catch (error) {
    notify(error instanceof Error ? error.message : "内容加载失败", "error");
    router.replace("/exchange");
  } finally {
    loading.value = false;
  }
}

async function submit() {
  submitting.value = true;
  try {
    const result = await api<{ id?: number }>(
      editId.value ? `/api/exchange/posts/${editId.value}` : "/api/exchange/posts",
      {
        method: editId.value ? "PATCH" : "POST",
        body: JSON.stringify(form),
      },
    );
    notify(editId.value ? "分享已更新" : "分享已发布");
    router.push(`/exchange/${editId.value || result.id}`);
  } catch (error) {
    notify(error instanceof Error ? error.message : "发布失败", "error");
  } finally {
    submitting.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="editor-page page-shell narrow">
    <button class="back-link" @click="router.back()">
      <ArrowLeft :size="17" /> 返回
    </button>
    <LoadingBlock v-if="loading" />
    <template v-else>
      <header class="page-heading compact-heading">
        <span class="eyebrow">{{ editId ? "编辑内容" : "分享此刻" }}</span>
        <h1>{{ editId ? "让这段文字更接近你想表达的" : "把值得分享的事写下来" }}</h1>
      </header>
      <form class="editor-card stack-form" @submit.prevent="submit">
        <label v-if="auth.state.user?.role === 'admin'">
          <span>内容类型</span>
          <select v-model="form.kind">
            <option value="share">同好分享</option>
            <option value="news">校园新闻</option>
          </select>
          <small class="field-help">校园新闻仅管理员可以发布。</small>
        </label>
        <label>
          <span>标题</span>
          <input
            v-model="form.title"
            maxlength="100"
            required
            placeholder="用一句让人想点开的话"
          />
        </label>
        <label>
          <span>正文</span>
          <textarea
            v-model="form.content"
            class="large-textarea"
            maxlength="10000"
            required
            placeholder="记录故事、感受，或者给读者的一个小邀请…"
          />
          <small class="field-help">{{ form.content.length }} / 10000</small>
        </label>
        <ImageUpload v-model="form.imageUrl" />
        <div class="form-actions">
          <button type="button" class="button secondary" @click="router.back()">
            取消
          </button>
          <button class="button" :disabled="submitting">
            <Save :size="17" />
            {{ submitting ? "正在保存…" : editId ? "保存修改" : "发布分享" }}
          </button>
        </div>
      </form>
    </template>
  </div>
</template>
