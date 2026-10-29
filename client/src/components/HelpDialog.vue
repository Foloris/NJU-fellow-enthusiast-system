<script setup lang="ts">
import { computed } from "vue";
import {
  LifeBuoy,
  Lightbulb,
  MessagesSquare,
  PenLine,
  ShieldAlert,
  UserRound,
} from "lucide-vue-next";
import Modal from "./Modal.vue";

const props = defineProps<{
  topic: "" | "faq" | "guide";
}>();

const emit = defineEmits<{
  close: [];
  feedback: [];
}>();

const title = computed(() => (props.topic === "faq" ? "常见问题" : "使用说明"));

const guideSections = [
  {
    icon: UserRound,
    title: "注册账号",
    text: "用 @smail.nju.edu.cn 邮箱注册，输入邮件中的 6 位验证码即可完成绑定。",
  },
  {
    icon: PenLine,
    title: "发布帖子",
    text: "进入「同好交流」板块，填写标题与正文，支持上传一张封面图。亮眼的标题和内容更容易吸引大家来留言互动～",
  },
  {
    icon: MessagesSquare,
    title: "搭子申请与私信",
    text: "在「找搭子」的帖子页发起申请，对方通过后就可以在「私信」中实时沟通了。",
  },
  {
    icon: ShieldAlert,
    title: "举报处理",
    text: "遇到不良内容可对帖子或用户提交举报，管理员核实后撤销内容或封禁账号。",
  },
  {
    icon: LifeBuoy,
    title: "问题反馈",
    text: "出现任何使用问题都可以在「问题反馈」中反馈给我们，写上详细步骤并附上截图可以帮助更快定位。",
  },
  {
    icon: Lightbulb,
    title: "意见反馈",
    text: "有任何改进意见都欢迎在「意见反馈」告诉我们 ~",
  },
];
</script>

<template>
  <Modal :open="topic !== ''" :title="title" width="620px" @close="emit('close')">
    <div v-if="topic === 'faq'" class="help-block">
      <div class="help-accordion">
        <details class="help-item">
          <summary>注册需要校园邮箱吗？</summary>
          <p>需要，目前只接受 @smail.nju.edu.cn 邮箱。</p>
        </details>
        <details class="help-item">
          <summary>收不到验证码怎么办？</summary>
          <p>
            验证码在 60 秒内有效，倒计时结束后可以重新发送。仍收不到时，建议先查看邮箱的垃圾邮件目录；如果一直没有来信，可通过「问题反馈」告诉我们。
          </p>
        </details>
        <details class="help-item">
          <summary>为什么只能和部分同学私信？</summary>
          <p>私信在搭子申请通过后才开放，避免陌生消息打扰。</p>
        </details>
        <details class="help-item">
          <summary>发现了不合适的内容怎么办？</summary>
          <p>可以对帖子或用户进行举报，管理员审核确认后进行撤销帖子或者封禁用户。</p>
        </details>
        <details class="help-item">
          <summary>账号被封禁后还能使用吗？</summary>
          <p>
            封禁后无法登录、发布内容和发送私信。如果认为处理有误，可以通过「问题反馈」提交申诉，附上账号与具体情况。
          </p>
        </details>
        <details class="help-item">
          <summary>图片上传有什么限制？</summary>
          <p>支持 JPG、PNG、WebP 和 GIF 四种格式，单张图片不超过 5MB。</p>
        </details>
      </div>

      <button type="button" class="help-more" @click="emit('feedback')">
        都不是你遇到的问题？点这里试试。
      </button>
    </div>

    <div v-else class="help-block">
      <div class="help-sections">
        <section v-for="item in guideSections" :key="item.title" class="help-section">
          <span class="help-section-icon">
            <component :is="item.icon" :size="15" aria-hidden="true" />
          </span>
          <div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.text }}</p>
          </div>
        </section>
      </div>
    </div>
  </Modal>
</template>
