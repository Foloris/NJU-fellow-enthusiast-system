export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.status = status;
  }
}

export async function api<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !(options.body instanceof FormData)) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(path, {
    ...options,
    headers,
    credentials: "include",
  });
  const data = (await response.json().catch(() => ({}))) as {
    message?: string;
  };
  if (!response.ok) {
    throw new ApiError(data.message || "请求失败，请稍后重试", response.status);
  }
  return data as T;
}

export function uploadImage(file: File) {
  const form = new FormData();
  form.append("image", file);
  return api<{ url: string }>("/api/uploads", {
    method: "POST",
    body: form,
  });
}

