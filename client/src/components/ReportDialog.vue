<script setup lang="ts">
import { ref, watch } from "vue";
import Modal from "./Modal.vue";
import { api } from "@/api";
import { notify } from "@/stores/toast";

const props = defineProps<{
  open: boolean;
  targetType: "post" | "mate" | "user" | "message";
  targetId: number;
}>();
const emit = defineEmits<{ close: []; submitted: [] }>();

const reason = ref("不友善或骚扰");
const details = ref("");
const submitting = ref(false);

watch(
  () => props.open,
  (open) => {
    if (open) {
      reason.value = "不友善或骚扰";
      details.value = "";
    }
  },
);

async function submit() {
  submitting.value = true;
  try {
    const path =
      props.targetType === "post"
        ? `/api/exchange/posts/${props.targetId}/report`
        : props.targetType === "mate"
          ? `/api/mates/${props.targetId}/report`
          : "/api/reports";
    const payload =
      props.targetType === "post" || props.targetType === "mate"
        ? { reason: reason.value, details: details.value }
        : {
            targetType: props.targetType,
            targetId: props.targetId,
            reason: reason.value,
            details: details.value,
          };
    await api(path, { method: "POST", body: JSON.stringify(payload) });
    notify("举报已提交，管理员会尽快处理");
    emit("submitted");
    emit("close");
  } catch (error) {
    notify(error instanceof Error ? error.message : "举报提交失败", "error");
  } finally {
    submitting.value = false;
  }
}
</script>

<template>
  <Modal :open="open" title="举报内容" @close="emit('close')">
    <form class="stack-form" @submit.prevent="submit">
      <label>
        <span>举报原因</span>
        <select v-model="reason">
          <option>不友善或骚扰</option>
          <option>垃圾广告或诈骗</option>
          <option>违法违规内容</option>
          <option>侵犯隐私或冒用身份</option>
          <option>其他问题</option>
        </select>
      </label>
      <label>
        <span>补充说明 <small>选填</small></span>
        <textarea
          v-model="details"
          rows="4"
          maxlength="1000"
          placeholder="请描述具体情况，便于管理员判断"
        />
      </label>
      <p class="form-note">
        管理员只会查看被举报内容，以及私信举报所需的相邻上下文。
      </p>
      <button class="button danger-button" :disabled="submitting">
        {{ submitting ? "正在提交…" : "提交举报" }}
      </button>
    </form>
  </Modal>
</template>

