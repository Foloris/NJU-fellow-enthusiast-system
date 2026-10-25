<script setup lang="ts">
import { ref } from "vue";
import { ImagePlus, LoaderCircle, X } from "lucide-vue-next";
import { uploadImage } from "@/api";
import { notify } from "@/stores/toast";

const props = defineProps<{ modelValue: string | null }>();
const emit = defineEmits<{ "update:modelValue": [value: string | null] }>();
const uploading = ref(false);
const fileInput = ref<HTMLInputElement | null>(null);

async function onFile(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0];
  if (!file) return;
  uploading.value = true;
  try {
    const { url } = await uploadImage(file);
    emit("update:modelValue", url);
  } catch (error) {
    notify(error instanceof Error ? error.message : "图片上传失败", "error");
  } finally {
    uploading.value = false;
    if (fileInput.value) fileInput.value.value = "";
  }
}
</script>

<template>
  <div class="image-uploader">
    <div v-if="modelValue" class="image-preview">
      <img :src="modelValue" alt="已上传的图片预览" />
      <button
        type="button"
        class="remove-image"
        aria-label="移除图片"
        @click="emit('update:modelValue', null)"
      >
        <X :size="18" />
      </button>
    </div>
    <button
      v-else
      type="button"
      class="upload-placeholder"
      :disabled="uploading"
      @click="fileInput?.click()"
    >
      <LoaderCircle v-if="uploading" class="spin" :size="22" />
      <ImagePlus v-else :size="22" />
      <span>{{ uploading ? "正在上传…" : "添加一张图片" }}</span>
      <small>JPG、PNG、WebP，最大 5MB</small>
    </button>
    <input
      ref="fileInput"
      class="sr-only"
      type="file"
      accept="image/jpeg,image/png,image/webp,image/gif"
      @change="onFile"
    />
  </div>
</template>

