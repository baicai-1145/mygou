<script setup>
import Icon from "./Icon.vue";

const props = defineProps({
  tracks: { type: Array, default: () => [] },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["play", "artist", "album"]);

function cover(track, size = 100) {
  const c = track.cover || "";
  return c ? c.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}

function fav(t) {
  window.dispatchEvent(new CustomEvent("mygou-fav", { detail: t }));
}

function collect(t) {
  window.dispatchEvent(new CustomEvent("mygou-collect", { detail: t }));
}

function time(sec) {
  sec = Math.max(0, Math.floor(sec || 0));
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}
</script>

<template>
  <div class="table">
    <div class="head">
      <div class="col-idx">#</div>
      <div class="col-cover"></div>
      <div class="col-name">标题</div>
      <div class="col-singer">歌手</div>
      <div class="col-album">专辑</div>
      <div class="col-time">时长</div>
    </div>
    <div
      v-for="(t, i) in tracks"
      :key="t.hash + i"
      class="row"
      :class="{ current: t.hash === currentHash }"
      @dblclick="emit('play', i)"
    >
      <div class="col-idx">
        <span class="num">{{ String(i + 1).padStart(2, "0") }}</span>
        <button class="play-btn" title="播放" @click="emit('play', i)">
          <Icon :name="t.hash === currentHash && playing ? 'pause' : 'play'" />
        </button>
      </div>
      <div class="col-cover">
        <img v-if="cover(t)" :src="cover(t)" loading="lazy" alt="" />
        <div v-else class="cover-empty"><Icon name="music" /></div>
      </div>
      <div class="col-name" :title="t.name">
        {{ t.name }}
        <button class="fav" title="添加到「我喜欢」" @click.stop="fav(t)" @dblclick.stop>♥</button>
        <button class="col-action" title="收藏到歌单" @click.stop="collect(t)" @dblclick.stop>＋</button>
      </div>
      <div class="col-singer" :title="t.singer">
        <button v-if="t.singer" class="link" @click.stop="emit('artist', t)" @dblclick.stop>
          {{ t.singer }}
        </button>
        <span v-else>-</span>
      </div>
      <div class="col-album" :title="t.album">
        <button
          v-if="t.albumID"
          class="link"
          @click.stop="emit('album', t)"
          @dblclick.stop
        >
          {{ t.album || "未知专辑" }}
        </button>
        <span v-else>{{ t.album || "-" }}</span>
      </div>
      <div class="col-time">{{ time(t.duration) }}</div>
    </div>
    <div v-if="tracks.length === 0" class="empty">没有结果</div>
  </div>
</template>

<style scoped>
.table {
  display: flex;
  flex-direction: column;
}
.head,
.row {
  display: grid;
  grid-template-columns: 44px 56px minmax(180px, 1.4fr) minmax(120px, 1fr) minmax(120px, 1fr) 64px;
  align-items: center;
  gap: 8px;
}
.head {
  height: 36px;
  font-size: 12px;
  color: var(--color-secondary);
  border-bottom: 1px solid var(--color-secondary-bg-for-transparent);
  margin-bottom: 4px;
}
.row {
  height: 56px;
  border-radius: 8px;
  font-size: 13px;
  padding: 0 8px;
  cursor: pointer;
}
.row:hover {
  background: var(--color-secondary-bg-for-transparent);
}
.row.current .col-name {
  color: var(--color-primary);
  font-weight: 600;
}
.fav {
  margin-left: 6px;
  border: 0;
  background: none;
  color: var(--color-secondary);
  font-size: 13px;
  opacity: 0;
  cursor: pointer;
  transition: opacity 0.15s ease, color 0.15s ease;
  vertical-align: middle;
}
.row:hover .fav {
  opacity: 1;
}
.fav:hover {
  color: #f56c6c;
}
.col-action {
  margin-left: 6px;
  border: 0;
  background: none;
  color: var(--color-secondary);
  font-size: 13px;
  opacity: 0;
  cursor: pointer;
  transition: opacity 0.15s ease, color 0.15s ease;
  vertical-align: middle;
}
.row:hover .col-action {
  opacity: 1;
}
.col-action:hover {
  color: var(--color-primary);
}
.col-idx {
  position: relative;
  display: grid;
  place-items: center;
  color: var(--color-secondary);
  font-size: 12px;
}
.play-btn {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  opacity: 0;
  color: var(--color-text);
}
.row:hover .play-btn {
  opacity: 1;
}
.row:hover .num {
  opacity: 0;
}
.col-cover img,
.cover-empty {
  width: 40px;
  height: 40px;
  border-radius: 6px;
  object-fit: cover;
  display: block;
}
.cover-empty {
  display: grid;
  place-items: center;
  background: var(--color-secondary-bg);
  color: var(--color-secondary);
}
.link {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: inherit;
  font-size: inherit;
  text-align: left;
}
.link:hover {
  color: var(--color-primary);
  text-decoration: underline;
}
.col-name,
.col-singer,
.col-album {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  padding-right: 8px;
}
.col-singer,
.col-album {
  color: var(--color-secondary);
}
.col-time {
  color: var(--color-secondary);
  font-size: 12px;
  text-align: right;
  padding-right: 6px;
}
.empty {
  padding: 48px 0;
  text-align: center;
  color: var(--color-secondary);
}
</style>
