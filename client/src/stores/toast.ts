import { reactive } from "vue";

export type Toast = {
  id: number;
  message: string;
  type: "success" | "error" | "info";
};

export const toasts = reactive<Toast[]>([]);
let nextId = 1;

export function notify(
  message: string,
  type: Toast["type"] = "success",
) {
  const item = { id: nextId++, message, type };
  toasts.push(item);
  window.setTimeout(() => {
    const index = toasts.findIndex((toast) => toast.id === item.id);
    if (index >= 0) toasts.splice(index, 1);
  }, 3500);
}

