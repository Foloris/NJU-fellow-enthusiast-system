<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { ImagePlus, X } from "lucide-vue-next";
import Modal from "./Modal.vue";
import { api } from "@/api";
import { notify } from "@/stores/toast";

const maxShots = 3;
const maxShotSize = 5 * 1024 * 1024;
const shotTypes = ["image/png", "image/jpeg", "image/webp", "image/gif"];

type Shot = { id: number; name: string; blob: Blob; url: string };

const props = defineProps<{
  open: boolean;
  type: "problem" | "suggestion";
}>();
const emit = defineEmits<{ close: [] }>();

const content = ref("");
const error = ref("");
const submitting = ref(false);
const shots = ref<Shot[]>([]);
let nextShotId = 1;

const title = computed(() => (props.type === "problem" ? "问题反馈" : "意见反馈"));

function clearShots() {
  shots.value.forEach((shot) => URL.revokeObjectURL(shot.url));
  shots.value = [];
}

function addShots(files: File[] | null) {
  if (!files || files.length === 0) return;
  const accepted: Shot[] = [];
  let problem = "";
  for (const file of files) {
    if (shots.value.length + accepted.length >= maxShots) {
      problem = `最多只能添加 ${maxShots} 张截图`;
      break;
    }
    if (!shotTypes.includes(file.type)) {
      problem = "只支持 JPG、PNG、WebP、GIF 格式的截图";
      continue;
    }
    if (file.size > maxShotSize) {
      problem = "单张截图不能超过 5MB";
      continue;
    }
    accepted.push({
      id: nextShotId++,
      name: file.name,
      blob: file,
      url: URL.createObjectURL(file),
    });
  }
  shots.value.push(...accepted);
  error.value = problem;
}

function onPick(event: Event) {
  const input = event.target as HTMLInputElement;
  addShots(Array.from(input.files ?? []));
  input.value = "";
}

function onDrop(event: DragEvent) {
  addShots(Array.from(event.dataTransfer?.files ?? []));
}

function removeShot(id: number) {
  const shot = shots.value.find((item) => item.id === id);
  if (shot) URL.revokeObjectURL(shot.url);
  shots.value = shots.value.filter((item) => item.id !== id);
}

function onPaste(event: ClipboardEvent) {
  const images = Array.from(event.clipboardData?.files ?? []).filter((file) =>
    file.type.startsWith("image/"),
  );
  if (images.length === 0) return;
  event.preventDefault();
  addShots(images);
}

watch(
  () => props.open,
  (open) => {
    window.removeEventListener("paste", onPaste);
    if (open) {
      content.value = "";
      error.value = "";
      submitting.value = false;
      clearShots();
      window.addEventListener("paste", onPaste);
    }
  },
);

onBeforeUnmount(() => {
  window.removeEventListener("paste", onPaste);
  clearShots();
});

async function submit() {
  const text = content.value.trim();
  if (text.length < 5) {
    error.value = "请至少写 5 个字，说明你遇到的情况或想法";
    return;
  }
  error.value = "";
  submitting.value = true;
  try {
    const form = new FormData();
    form.append("type", props.type);
    form.append("content", text);
    shots.value.forEach((shot, index) => form.append("images", shot.blob, `截图${index + 1}`));
    await api("/api/feedback", { method: "POST", body: form });
    notify("反馈已发送，我们会尽快处理");
    emit("close");
  } catch (failure) {
    error.value = failure instanceof Error ? failure.message : "反馈发送失败，请稍后重试";
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <Modal :open="open" :title="title" @close="emit('close')">
    <form class="stack-form" @submit.prevent="submit">
      <label>
        <span>反馈内容</span>
        <textarea
          v-model="content"
          rows="6"
          maxlength="1000"
          :placeholder="
            type === 'problem'
              ? '描述你遇到的问题，例如在哪个页面、做了什么操作、出现了什么提示'
              : '说说你希望怎么改进，或者期待增加什么功能'
          "
        />
      </label>

      <div v-if="type === 'problem'" class="shot-field">
        <span class="shot-label">附带截图 <small>选填，最多 {{ maxShots }} 张</small></span>
        <div class="shot-list" @dragover.prevent @drop.prevent="onDrop">
          <figure v-for="shot in shots" :key="shot.id" class="shot-item">
            <img :src="shot.url" :alt="shot.name" />
            <button
              type="button"
              class="shot-remove"
              aria-label="移除截图"
              @click="removeShot(shot.id)"
            >
              <X :size="13" />
            </button>
          </figure>
          <label v-if="shots.length < maxShots" class="shot-add">
            <input
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              multiple
              hidden
              @change="onPick"
            />
            <ImagePlus :size="18" aria-hidden="true" />
            <span>添加图片</span>
          </label>
        </div>
        <p class="shot-hint">可点击选择、拖入或按 Ctrl+V 粘贴，单张不超过 5MB。</p>
      </div>

      <p v-if="error" class="form-error">{{ error }}</p>
      <button class="button" :disabled="submitting || !content.trim()">
        {{ submitting ? "正在发送…" : "发送反馈" }}
      </button>
    </form>
  </Modal>
</template>
