import { z } from "zod";

export const registerSchema = z.object({
  username: z
    .string()
    .trim()
    .min(3, "账号至少 3 个字符")
    .max(24, "账号最多 24 个字符")
    .regex(/^[A-Za-z0-9_\u4e00-\u9fa5]+$/, "账号只能包含中英文、数字或下划线"),
  password: z.string().min(8, "密码至少 8 位").max(72, "密码最多 72 位"),
  nickname: z.string().trim().min(1, "请填写昵称").max(30, "昵称最多 30 个字符"),
});

export const loginSchema = z.object({
  username: z.string().trim().min(1, "请输入账号"),
  password: z.string().min(1, "请输入密码"),
});

export const profileSchema = z.object({
  nickname: z.string().trim().min(1).max(30),
  bio: z.string().trim().max(200).default(""),
  avatar: z.string().trim().max(500).nullable().optional(),
});

export const exchangePostSchema = z.object({
  kind: z.enum(["share", "news"]).default("share"),
  title: z.string().trim().min(4, "标题至少 4 个字符").max(100),
  content: z.string().trim().min(10, "正文至少 10 个字符").max(10000),
  imageUrl: z.string().trim().max(500).nullable().optional(),
});

export const matePostSchema = z.object({
  title: z.string().trim().min(4, "标题至少 4 个字符").max(100),
  tags: z.array(z.string().trim().min(1).max(16)).min(1, "至少选择一个标签").max(6),
  details: z.string().trim().min(10, "详细说明至少 10 个字符").max(10000),
  imageUrl: z.string().trim().max(500).nullable().optional(),
  location: z.string().trim().max(100).nullable().optional(),
  contact: z.string().trim().min(2, "请填写联系方式").max(100),
  duration: z.string().trim().min(1, "请填写持续时间").max(100),
  notes: z.string().trim().max(1000).nullable().optional(),
});

export const applicationSchema = z.object({
  message: z.string().trim().min(1, "请写一句申请说明").max(500),
});

export const reportSchema = z.object({
  targetType: z.enum(["post", "mate", "user", "message"]),
  targetId: z.number().int().positive(),
  reason: z.string().trim().min(2, "请选择或填写举报原因").max(100),
  details: z.string().trim().max(1000).default(""),
});

export function formatZodError(error: z.ZodError) {
  return error.issues.map((issue) => issue.message).join("；");
}

