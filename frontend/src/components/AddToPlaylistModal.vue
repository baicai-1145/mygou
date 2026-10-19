<script setup>
import { onMounted, ref } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({
  track: { type: Object, default: null },
});
const emit = defineEmits(["close", "added"]);

const call = (method, ...args) => window.mygo.call(method, ...args);

const playlists = ref([]);
const loading = ref(true);
const error = ref("");
const newName = ref("");
const creating = ref(false);

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const list = await call("App.UserPlaylists");
    playlists.value = list || [];
  } catch (e) {
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

onMounted(load);

async function addTo(pl) {
  try {
    await call("App.AddTrackToPlaylist", pl.listid || 0, props.track || {});
    emit("added", pl.name);
    emit("close");
  } catch (e) {
    alert(e?.message || "添加失败");
  }
}

async function createAndAdd() {
  const name = newName.value.trim();
  if (!name) return;
  creating.value = true;
  try {
    const lists = await call("App.CreatePlaylistAndAdd", name, props.track || {});
    playlists.value = lists || [];
    emit("added", name);
    emit("close");
  } catch (e) {
    alert(e?.message || "创建失败");
  } finally {
    creating.value = false;
  }
}

function coverUrl(c) {
  return c ? c.replace("{size}", "120").replace(/^http:/, "https:") : "";
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <div class="panel">
      <header class="phead">
        <h3>收藏到歌单</h3>
        <button class="close-btn" title="关闭" @click="emit('close')">✕</button>
      </header>

      <div v-if="track" class="song-card">
        <div class="song-info">
          <div class="name">{{ track.name }}</div>
          <div class="singer">{{ track.singer }}</div>
        </div>
      </div>

      <div class="create-bar">
        <input
          v-model="newName"
          type="text"
          placeholder="新建歌单名称…"
          maxlength="30"
          @keydown.enter="createAndAdd"
        />
        <button class="create-btn" :disabled="!newName.trim() || creating" @click="createAndAdd">
          新建并收藏
        </button>
      </div>

      <div class="list-box">
        <div v-if="loading" class="empty">载入歌单中…</div>
        <div v-else-if="error" class="empty error">{{ error }}</div>
        <div v-else-if="!playlists.length" class="empty">暂无可用歌单</div>
        <div
          v-for="p in playlists"
          :key="p.id"
          class="pl-item"
          @click="addTo(p)"
        >
          <div class="pl-cover">
            <img v-if="coverUrl(p.cover)" :src="coverUrl(p.cover)" alt="" />
            <Icon v-else name="music" />
          </div>
          <div class="pl-meta">
            <div class="pl-name">{{ p.name }}</div>
            <div class="pl-count">{{ p.count }} 首</div>
          </div>
          <span class="pl-add">+</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 70;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.55);
  backdrop-filter: blur(8px);
}
.panel {
  width: min(420px, 92vw);
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  border-radius: 18px;
  padding: 20px 22px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.45);
}
.phead {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.phead h3 {
  margin: 0;
  font-size: 17px;
  font-weight: 700;
}
.close-btn {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
  cursor: pointer;
}
.close-btn:hover {
  color: var(--color-text);
}
.song-card {
  padding: 10px 14px;
  border-radius: 10px;
  background: var(--color-secondary-bg-for-transparent);
  margin-bottom: 14px;
}
.song-card .name {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.song-card .singer {
  margin-top: 3px;
  font-size: 11px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.create-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}
.create-bar input {
  flex: 1;
  height: 34px;
  padding: 0 12px;
  border-radius: 8px;
  background: var(--color-secondary-bg);
  border: 1px solid var(--color-border);
  font-size: 12px;
  color: var(--color-text);
}
.create-btn {
  flex: none;
  height: 34px;
  padding: 0 14px;
  border-radius: 8px;
  background: var(--color-primary);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}
.create-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.list-box {
  flex: 1;
  overflow-y: auto;
  margin-top: 4px;
  max-height: 320px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.pl-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 10px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.15s ease;
}
.pl-item:hover {
  background: var(--color-secondary-bg-for-transparent);
}
.pl-cover {
  width: 44px;
  height: 44px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-secondary-bg);
  display: grid;
  place-items: center;
  flex: none;
}
.pl-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.pl-meta {
  flex: 1;
  min-width: 0;
}
.pl-name {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pl-count {
  margin-top: 3px;
  font-size: 11px;
  color: var(--color-secondary);
}
.pl-add {
  font-size: 18px;
  color: var(--color-secondary);
  margin-right: 4px;
}
.pl-item:hover .pl-add {
  color: var(--color-primary);
}
.empty {
  padding: 24px 0;
  text-align: center;
  font-size: 12px;
  color: var(--color-secondary);
}
.empty.error {
  color: #ff7a7a;
}
</style>
