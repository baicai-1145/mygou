<script setup>
import Icon from "./Icon.vue";

const props = defineProps({
  track: { type: Object, default: null },
  comments: { type: Array, default: () => [] },
  total: { type: Number, default: 0 },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  hasMore: { type: Boolean, default: false },
});
const emit = defineEmits(["close", "more"]);

function avatar(src) {
  if (!src) return "";
  return src.replace("{size}", "120").replace(/^http:/, "https:");
}
function when(t) {
  return (t || "").slice(0, 16);
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <section class="panel">
      <header>
        <div class="titles">
          <h2>评论</h2>
          <p v-if="track" class="sub">
            {{ track.name }} — {{ track.singer }}
            <span v-if="total"> · {{ total.toLocaleString() }} 条</span>
          </p>
        </div>
        <button class="close" title="关闭" @click="emit('close')">✕</button>
      </header>

      <div v-if="loading" class="hint">正在载入评论…</div>
      <div v-else-if="error" class="hint error">{{ error }}</div>
      <div v-else-if="!comments.length" class="hint">还没有评论</div>

      <div v-else class="list">
        <article v-for="c in comments" :key="c.id" class="row">
          <img v-if="c.avatar" class="avatar" :src="avatar(c.avatar)" alt="" />
          <div v-else class="avatar empty"><Icon name="music" /></div>
          <div class="body">
            <div class="meta">
              <span class="user">{{ c.user }}</span>
              <span v-if="c.location" class="dot">·</span>
              <span v-if="c.location" class="loc">{{ c.location }}</span>
              <span class="time">{{ when(c.time) }}</span>
            </div>
            <p class="content">{{ c.content }}</p>
            <div class="stats">
              <span v-if="c.likes">♥ {{ c.likes.toLocaleString() }}</span>
              <span v-if="c.replies">💬 {{ c.replies.toLocaleString() }}</span>
            </div>
          </div>
        </article>
        <div v-if="hasMore" class="more">
          <button @click="emit('more')">加载更多评论</button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 60;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.45);
  backdrop-filter: blur(6px);
}
.panel {
  width: min(620px, 92vw);
  max-height: min(78vh, 760px);
  display: flex;
  flex-direction: column;
  border-radius: 18px;
  padding: 22px 24px 8px;
  background: var(--color-bg);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.45);
  border: 1px solid var(--color-border);
}
header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 14px;
}
h2 {
  margin: 0;
  font-size: 20px;
  letter-spacing: -0.01em;
}
.sub {
  margin: 5px 0 0;
  font-size: 12px;
  color: var(--color-secondary);
}
.close {
  width: 28px;
  height: 28px;
  flex: none;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.close:hover {
  color: var(--color-text);
}
.close .icon {
  width: 14px;
  height: 14px;
}
.list {
  overflow-y: auto;
  padding-bottom: 12px;
}
.row {
  display: flex;
  gap: 12px;
  padding: 12px 0;
  border-top: 1px solid var(--color-border);
}
.avatar {
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 50%;
  object-fit: cover;
  background: var(--color-secondary-bg);
}
.avatar.empty {
  display: grid;
  place-items: center;
  color: var(--color-secondary);
}
.body {
  min-width: 0;
}
.meta {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  color: var(--color-secondary);
}
.user {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-primary);
}
.loc {
  color: var(--color-secondary);
}
.time {
  margin-left: auto;
}
.content {
  margin: 6px 0 0;
  font-size: 13px;
  line-height: 1.55;
  word-break: break-word;
}
.stats {
  display: flex;
  gap: 14px;
  margin-top: 6px;
  font-size: 11px;
  color: var(--color-secondary);
}
.more {
  padding: 14px 0 18px;
  text-align: center;
  border-top: 1px solid var(--color-border);
}
.more button {
  height: 30px;
  padding: 0 18px;
  border-radius: 15px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.more button:hover {
  color: var(--color-text);
}
.hint {
  padding: 36px 0;
  text-align: center;
  font-size: 13px;
  color: var(--color-secondary);
}
.hint.error {
  color: #e0533f;
}
</style>
