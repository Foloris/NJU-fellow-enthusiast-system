import { computed, reactive } from "vue";
import { api } from "@/api";
import type { User } from "@/types";

const state = reactive<{
  user: User | null;
  initialized: boolean;
}>({
  user: null,
  initialized: false,
});

let loading: Promise<void> | null = null;

async function loadMe(force = false) {
  if (state.initialized && !force) return;
  if (loading && !force) return loading;
  loading = api<{ user: User | null }>("/api/auth/me")
    .then((data) => {
      state.user = data.user;
      state.initialized = true;
    })
    .finally(() => {
      loading = null;
    });
  return loading;
}

async function login(username: string, password: string) {
  const data = await api<{ user: User }>("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ username, password }),
  });
  state.user = data.user;
  state.initialized = true;
}

async function register(username: string, password: string, nickname: string) {
  const data = await api<{ user: User }>("/api/auth/register", {
    method: "POST",
    body: JSON.stringify({ username, password, nickname }),
  });
  state.user = data.user;
  state.initialized = true;
}

async function logout() {
  await api("/api/auth/logout", { method: "POST" });
  state.user = null;
}

function updateUser(user: User) {
  state.user = user;
}

export const auth = {
  state,
  isLoggedIn: computed(() => Boolean(state.user)),
  isAdmin: computed(() => state.user?.role === "admin"),
  loadMe,
  login,
  register,
  logout,
  updateUser,
};

