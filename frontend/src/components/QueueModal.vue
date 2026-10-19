<script setup>
import Icon from "./Icon.vue";

const props = defineProps({
  queue: { type: Object, default: null },
});
const emit = defineEmits(["close", "play"]);

function cover(t, size = 80) {
  const c = t?.cover || "";
  return c ? c.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <section class="panel">
      <header>
        <div>
          <h2>播放队列</h2>
          <p>{{ queue?.tracks?.length || 0 }} 首</p>
        </div>
        <button class="close" title="关闭" @click="emit('close')">✕</button>
      </header>
      <div class="list">
        <button
          v-for="(t, i) in queue?.tracks || []"
          :key="t.hash + i"
          class="row"
          :class="{ current: i === queue.index }"
          @click="emit('play', i)"
        >
          <span class="idx">{{ String(i + 1).padStart(2, "0") }}</span>
          <span class="art">
            <img v-if="cover(t)" :src="cover(t)" alt="" loading="lazy" />
            <Icon v-else name="music" />
          </span>
          <span class="name">{{ t.name }}</span>
          <span class="singer">{{ t.singer }}</span>
          <span class="state">
            <Icon v-if="i === queue.index" name="play" />
          </span>
        </button>
        <div v-if="!queue?.tracks?.length" class="hint">队列是空的</div>
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
  width: min(640px, 92vw);
  max-height: min(76vh, 720px);
  display: flex;
  flex-direction: column;
  border-radius: 18px;
  padding: 20px 22px 10px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.45);
}
header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding-bottom: 12px;
}
h2 {
  margin: 0;
  font-size: 19px;
}
header p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--color-secondary);
}
.close {
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.close:hover {
  color: var(--color-text);
}
.list {
  overflow-y: auto;
  padding-bottom: 10px;
}
.row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 7px 8px;
  border-radius: 8px;
  text-align: left;
  color: inherit;
}
.row:hover {
  background: var(--color-secondary-bg);
}
.row.current .name,
.row.current .idx {
  color: var(--color-primary);
}
.idx {
  width: 22px;
  flex: none;
  font-size: 11px;
  color: var(--color-secondary);
}
.art {
  width: 32px;
  height: 32px;
  flex: none;
  border-radius: 6px;
  overflow: hidden;
  display: grid;
  place-items: center;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.singer {
  width: 120px;
  flex: none;
  font-size: 11px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.state {
  width: 18px;
  flex: none;
  display: grid;
  place-items: center;
  color: var(--color-primary);
}
.state .icon {
  width: 12px;
  height: 12px;
}
.hint {
  padding: 40px 0;
  text-align: center;
  font-size: 13px;
  color: var(--color-secondary);
}
</style>
