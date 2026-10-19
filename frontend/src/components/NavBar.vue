<script setup>
import { onMounted, onUnmounted, ref, watch } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({
  theme: { type: String, default: "dark" },
  loggedIn: { type: Boolean, default: false },
  avatar: { type: String, default: "" },
  view: { type: String, default: "home" },
});
const emit = defineEmits(["toggle-theme", "search", "login", "navigate", "settings"]);

const keyword = ref("");
const focused = ref(false);
const isFs = ref(false);

async function checkFs() {
  try {
    isFs.value = await window.mygo?.call("App.IsFullScreen");
  } catch {}
}
async function exitFs() {
  try {
    await window.mygo?.call("App.ExitFullScreen");
    isFs.value = false;
  } catch {}
}

function onFsEvent(e) {
  if (e?.detail && typeof e.detail.fullscreen === "boolean") isFs.value = e.detail.fullscreen;
  else checkFs();
}

onMounted(() => {
  checkFs();
  window.addEventListener("mygo:fullscreen", onFsEvent);
  window.addEventListener("resize", checkFs);
});
onUnmounted(() => {
  window.removeEventListener("mygo:fullscreen", onFsEvent);
  window.removeEventListener("resize", checkFs);
});

function submit() {
  const kw = keyword.value.trim();
  if (kw) emit("search", kw);
}
watch(
  () => props.theme,
  () => {},
);
</script>

<template>
  <header class="navbar">
    <div class="left">
      <div class="brand">mygou</div>
      <nav class="links">
        <span
          :class="{ active: view === 'home' || view === 'search' || view === 'playlist' }"
          @click="emit('navigate', 'home')"
          >发现音乐</span
        >
        <span :class="{ active: view === 'plaza' }" @click="emit('navigate', 'plaza')"
          >歌单广场</span
        >
        <span :class="{ active: view === 'rank' }" @click="emit('navigate', 'rank')"
          >排行榜</span
        >
        <span :class="{ active: view === 'mine' }" @click="emit('navigate', 'mine')"
          >我的音乐</span
        >
      </nav>
    </div>
    <div class="right">
      <label class="search" :class="{ active: focused }">
        <Icon name="search" />
        <input
          v-model="keyword"
          type="search"
          placeholder="搜索音乐"
          @focus="focused = true"
          @blur="focused = false"
          @keydown.enter="submit"
        />
      </label>
      <button v-if="isFs" class="round" title="退出全屏 (Esc)" @click="exitFs">
        <Icon name="fullscreenExit" />
      </button>
      <button class="round" title="设置" @click="emit('settings')">⚙</button>
      <button class="round" title="切换主题" @click="emit('toggle-theme')">
        <Icon :name="theme === 'dark' ? 'sun' : 'moon'" />
      </button>
      <button v-if="!loggedIn" class="login" @click="emit('login')">
        登录
      </button>
      <div v-else class="avatar" title="我的音乐" @click="emit('navigate', 'mine')">
        <img v-if="avatar" :src="avatar" alt="" />
      </div>
    </div>
  </header>
</template>

<style scoped>
.brand {
  margin-right: 8px;
}
.navbar {
  position: fixed;
  inset: 0 0 auto 0;
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 20px 0 92px;
  background: var(--color-navbar-bg);
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-bottom: 1px solid var(--color-secondary-bg-for-transparent);
  z-index: 20;
  --app-region: drag;
  -webkit-app-region: drag;
}
.left,
.right {
  display: flex;
  align-items: center;
  gap: 18px;
}
.links,
.search,
.round,
.login,
.avatar {
  --app-region: no-drag;
  -webkit-app-region: no-drag;
}
.brand {
  font-size: 20px;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--color-primary);
}
.links {
  display: flex;
  gap: 16px;
  font-size: 14px;
  font-weight: 600;
}
.links span {
  cursor: pointer;
  transition: color 0.15s ease;
}
.links span:hover {
  color: var(--color-text);
}
.links .active {
  color: var(--color-text);
}
.links .muted {
  color: var(--color-secondary);
}
.search {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 220px;
  height: 34px;
  padding: 0 12px;
  border-radius: 17px;
  background: var(--color-secondary-bg-for-transparent);
  color: var(--color-secondary);
  transition:
    width 0.2s ease,
    background 0.2s ease;
}
.search.active {
  width: 280px;
  background: var(--color-primary-bg-for-transparent);
  color: var(--color-text);
}
.search input {
  flex: 1;
  min-width: 0;
  font-size: 13px;
}
.search input::placeholder {
  color: var(--color-secondary);
}
.round {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  color: var(--color-secondary);
  background: var(--color-secondary-bg-for-transparent);
}
.round:hover {
  color: var(--color-text);
}
.login {
  height: 32px;
  padding: 0 16px;
  border-radius: 16px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary);
  background: var(--color-primary-bg-for-transparent);
}
.avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  overflow: hidden;
  cursor: pointer;
  background: linear-gradient(135deg, #ff9966, #ff5e62);
}
.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
