<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ArrowLeft, Save, X } from "lucide-vue-next";
import { api } from "@/api";
import { auth } from "@/stores/auth";
import { notify } from "@/stores/toast";
import type { MatePost } from "@/types";
import ImageUpload from "@/components/ImageUpload.vue";
import LoadingBlock from "@/components/LoadingBlock.vue";

const route = useRoute();
const router = useRouter();
const editId = computed(() =>
  route.name === "mate-edit" ? Number(route.params.id) : null,
);
const loading = ref(Boolean(editId.value));
const submitting = ref(false);
const newTag = ref("");
const form = reactive({
  title: "",
  tags: ["周末"] as string[],
  details: "",
  imageUrl: null as string | null,
  location: "",
  contact: "",
  duration: "",
  notes: "",
});
const suggestedTags = ["周末", "运动", "学习", "摄影", "户外", "长期", "今晚", "美食", "音乐"];

function toggleTag(tag: string) {
  if (form.tags.includes(tag)) {
    form.tags = form.tags.filter((item) => item !== tag);
  } else if (form.tags.length < 6) {
    form.tags.push(tag);
  }
}

function addCustomTag() {
  const tag = newTag.value.trim();
  if (tag && !form.tags.includes(tag) && form.tags.length < 6) form.tags.push(tag);
  newTag.value = "";
}

async function load() {
  if (!editId.value) return;
  try {
    const data = await api<{ post: MatePost }>(`/api/mates/${editId.value}`);
    if (
      data.post.author.id !== auth.state.user?.id &&
      auth.state.user?.role !== "admin"
    ) {
      notify("只能编辑自己发布的搭子", "error");
      router.replace("/mates");
      return;
    }
    Object.assign(form, {
      title: data.post.title,
      tags: data.post.tags,
      details: data.post.details,
      imageUrl: data.post.imageUrl,
      location: data.post.location ?? "",
      contact: data.post.contact ?? "",
      duration: data.post.duration,
      notes: data.post.notes ?? "",
    });
  } catch (error) {
    notify(error instanceof Error ? error.message : "加载失败", "error");
    router.replace("/mates");
  } finally {
    loading.value = false;
  }
}

async function submit() {
  submitting.value = true;
  try {
    const result = await api<{ id?: number }>(
      editId.value ? `/api/mates/${editId.value}` : "/api/mates",
      {
        method: editId.value ? "PATCH" : "POST",
        body: JSON.stringify(form),
      },
    );
    notify(editId.value ? "搭子邀请已更新" : "搭子邀请已发布");
    router.push(`/mates/${editId.value || result.id}`);
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
    <button class="back-link" @click="router.back()"><ArrowLeft :size="17" /> 返回</button>
    <LoadingBlock v-if="loading" />
    <template v-else>
      <header class="page-heading compact-heading">
        <span class="eyebrow">{{ editId ? "编辑邀请" : "发起邀请" }}</span>
        <h1>{{ editId ? "调整你的搭子邀请" : "说说你想找怎样的同伴" }}</h1>
        <p>清楚的时间、地点和期待，会让彼此更安心。</p>
      </header>
      <form class="editor-card stack-form mate-editor" @submit.prevent="submit">
        <label>
          <span>标题</span>
          <input
            v-model="form.title"
            maxlength="100"
            required
            placeholder="例如：周末去紫金山看秋天，想找一位慢慢走路的同伴"
          />
        </label>

        <fieldset class="tag-picker">
          <legend>标签 <small>最多 6 个</small></legend>
          <div class="tag-options">
            <button
              v-for="tag in suggestedTags"
              :key="tag"
              type="button"
              :class="{ selected: form.tags.includes(tag) }"
              @click="toggleTag(tag)"
            >
              {{ tag }}
            </button>
          </div>
          <div class="custom-tag">
            <input
              v-model="newTag"
              maxlength="16"
              placeholder="自定义标签"
              @keydown.enter.prevent="addCustomTag"
            />
            <button type="button" class="text-button" @click="addCustomTag">添加</button>
          </div>
          <div class="selected-tags">
            <span v-for="tag in form.tags" :key="tag" class="tag">
              {{ tag }}
              <button type="button" @click="toggleTag(tag)"><X :size="12" /></button>
            </span>
          </div>
        </fieldset>

        <label>
          <span>详细说明</span>
          <textarea
            v-model="form.details"
            class="large-textarea"
            maxlength="10000"
            required
            placeholder="活动安排、期待同伴具备的条件、交流方式…"
          />
        </label>

        <div class="form-grid">
          <label>
            <span>持续时间</span>
            <input v-model="form.duration" maxlength="100" required placeholder="例如：周六 08:00—15:00" />
          </label>
          <label>
            <span>地点 <small>选填</small></span>
            <input v-model="form.location" maxlength="100" placeholder="例如：仙林校区图书馆" />
          </label>
        </div>

        <label>
          <span>联系方式</span>
          <input
            v-model="form.contact"
            maxlength="100"
            required
            placeholder="微信号或手机号，仅在申请通过后向申请人展示"
          />
          <small class="field-help">出于隐私保护，联系方式不会出现在公开列表和详情页。</small>
        </label>

        <label>
          <span>备注 <small>选填</small></span>
          <textarea
            v-model="form.notes"
            rows="3"
            maxlength="1000"
            placeholder="例如：雨天顺延、自带球拍、费用 AA"
          />
        </label>

        <ImageUpload v-model="form.imageUrl" />

        <div class="form-actions">
          <button type="button" class="button secondary" @click="router.back()">取消</button>
          <button class="button warm-button" :disabled="submitting">
            <Save :size="17" />
            {{ submitting ? "正在保存…" : editId ? "保存修改" : "发布邀请" }}
          </button>
        </div>
      </form>
    </template>
  </div>
</template>

