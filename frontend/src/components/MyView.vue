<script setup>
import Icon from "./Icon.vue";

const props = defineProps({
  playlists: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  loggedIn: { type: Boolean, default: false },
  account: { type: Object, default: () => ({}) },
});
const emit = defineEmits(["open", "login", "create"]);

function cover(p, size = 320) {
  return p.cover ? p.cover.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}
</script>

<template>
  <section class="mine">
    <div v-if="!loggedIn" class="signin">
      <Icon name="music" />
      <h2>登录后看到你的歌单</h2>
      <p>收藏、歌单和最近播放都会同步到酷狗概念版账号。</p>
      <button class="primary" @click="emit('login')">扫码登录</button>
    </div>
    <template v-else>
      <div class="who">
        <div class="avatar">
          <img v-if="account.avatar" :src="account.avatar" alt="" />
          <Icon v-else name="music" />
        </div>
        <div>
          <h2>{{ account.name || "酷狗用户" }}</h2>
          <p>共 {{ playlists.length }} 个歌单</p>
        </div>
        <button class="new" @click="emit('create')">＋ 新建歌单</button>
      </div>
      <div v-if="loading" class="hint">正在载入歌单…</div>
      <div v-else-if="error" class="hint error">{{ error }}</div>
      <div v-else class="grid">
        <button
          v-for="p in playlists"
          :key="p.id"
          class="card"
          @click="emit('open', p)"
        >
          <div class="art">
            <img v-if="cover(p)" :src="cover(p)" alt="" loading="lazy" />
            <Icon v-else name="music" />
            <span class="count">{{ p.count }} 首</span>
          </div>
          <div class="name">{{ p.name }}</div>
          <div class="creator">{{ p.creator }}</div>
        </button>
      </div>
    </template>
  </section>
</template>

<style scoped>
.mine {
  padding: 28px 0 40px;
}
.signin {
  max-width: 420px;
  margin: 10vh auto;
  text-align: center;
  color: var(--color-secondary);
}
.signin .icon {
  width: 46px;
  height: 46px;
  opacity: 0.5;
}
.signin h2 {
  margin: 14px 0 6px;
  color: var(--color-text);
  font-size: 20px;
}
.signin p {
  font-size: 13px;
  margin: 0 0 20px;
}
.primary {
  padding: 10px 22px;
  border-radius: 22px;
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
}
.new {
  margin-left: auto;
  padding: 7px 14px;
  border-radius: 999px;
  border: 1px solid var(--color-border);
  background: var(--color-secondary-bg);
  color: var(--color-text);
  font-size: 13px;
  cursor: pointer;
}
.who {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 26px;
}
.avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  overflow: hidden;
  display: grid;
  place-items: center;
  background: var(--color-secondary-bg);
  color: var(--color-secondary);
}
.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.who h2 {
  margin: 0 0 4px;
  font-size: 22px;
}
.who p {
  margin: 0;
  font-size: 13px;
  color: var(--color-secondary);
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(168px, 1fr));
  gap: 22px;
}
.card {
  text-align: left;
  color: inherit;
}
.art {
  position: relative;
  aspect-ratio: 1;
  border-radius: 12px;
  overflow: hidden;
  background: linear-gradient(
    135deg,
    var(--color-primary-bg-for-transparent),
    var(--color-secondary-bg)
  );
  display: grid;
  place-items: center;
  color: var(--color-secondary);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.35s ease;
}
.card:hover .art img {
  transform: scale(1.04);
}
.count {
  position: absolute;
  right: 8px;
  top: 8px;
  padding: 2px 8px;
  border-radius: 10px;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  font-size: 11px;
  backdrop-filter: blur(6px);
}
.name {
  margin-top: 10px;
  font-size: 14px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.creator {
  margin-top: 2px;
  font-size: 12px;
  color: var(--color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.hint {
  padding: 40px 0;
  color: var(--color-secondary);
  font-size: 13px;
}
.hint.error {
  color: #ff7a7a;
}
</style>
