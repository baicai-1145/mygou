<script setup>
import { computed, onMounted, onUnmounted, ref } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({
  state: { type: Object, default: null },
  position: { type: Number, default: 0 },
  mode: { type: String, default: "list" },
  liked: { type: Boolean, default: false },
});
const emit = defineEmits(["toggle", "prev", "next", "seek", "volume", "lyrics", "favorite", "collect", "quality", "comments", "mode", "download", "queue"]);
const modeLabel = computed(() =>
  props.mode === "one" ? "单曲" : props.mode === "shuffle" ? "随机" : "循环",
);

const qParam = new URLSearchParams(location.search).get("quality");
const pref = ref(
  qParam || (localStorage.getItem("quality") === "128" ? "128" : "320"),
);
function syncQuality(e) {
  if (e?.detail) pref.value = e.detail;
}
onMounted(() => window.addEventListener("mygou-quality", syncQuality));
onUnmounted(() => window.removeEventListener("mygou-quality", syncQuality));

function pickQuality(e) {
  pref.value = e.target.value;
  localStorage.setItem("quality", pref.value);
  emit("quality", pref.value);
}

const track = computed(() => props.state?.track || null);
const duration = computed(() => props.state?.duration || 0);
const actual = computed(() => {
  const s = props.state;
  if (!s?.track) return "";
  if (s.duration > 0 && s.track.duration > 30 && s.duration < s.track.duration - 5) return "试听";
  return s.quality === "flac" || s.quality === "high"
    ? "无损"
    : s.quality === "320"
      ? "高品质"
      : s.quality === "128"
        ? "标准"
        : "";
});
const scrubRatio = ref(null);
const progress = computed(() => {
  if (scrubRatio.value !== null) return scrubRatio.value;
  return duration.value > 0 ? Math.min(1, props.position / duration.value) : 0;
});

function cover(size) {
  const c = track.value?.cover || "";
  return c ? c.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}

function time(sec) {
  sec = Math.max(0, Math.floor(sec || 0));
  const m = Math.floor(sec / 60);
  const s = sec % 60;
  return `${m}:${String(s).padStart(2, "0")}`;
}

const bar = ref(null);
function getRatio(e) {
  if (!bar.value || duration.value <= 0) return 0;
  const r = bar.value.getBoundingClientRect();
  return Math.min(1, Math.max(0, (e.clientX - r.left) / r.width));
}
function down(e) {
  if (duration.value <= 0) return;
  scrubRatio.value = getRatio(e);
  window.addEventListener("pointermove", move);
  window.addEventListener("pointerup", up);
}
function move(e) {
  if (scrubRatio.value !== null) {
    scrubRatio.value = getRatio(e);
  }
}
function up(e) {
  if (scrubRatio.value !== null) {
    const finalRatio = getRatio(e);
    emit("seek", finalRatio * duration.value);
    scrubRatio.value = null;
  }
  window.removeEventListener("pointermove", move);
  window.removeEventListener("pointerup", up);
}

const vol = ref(null);
const isVolDrag = ref(false);
const localVol = ref(null);
let lastVol = 1;
let volWheelTimer = null;

const currentVol = computed(() => {
  if (localVol.value !== null) return localVol.value;
  return props.state?.volume ?? 1;
});
const volPercent = computed(() => Math.round(currentVol.value * 100));

function updateVol(e) {
  if (!vol.value) return;
  const r = vol.value.getBoundingClientRect();
  const val = Math.min(1, Math.max(0, (e.clientX - r.left) / r.width));
  localVol.value = val;
  emit("volume", val);
}

function vdown(e) {
  isVolDrag.value = true;
  if (e.currentTarget?.setPointerCapture) {
    try {
      e.currentTarget.setPointerCapture(e.pointerId);
    } catch {}
  }
  updateVol(e);
}
function vmove(e) {
  if (isVolDrag.value) updateVol(e);
}
function vup(e) {
  if (isVolDrag.value) {
    if (e.currentTarget?.releasePointerCapture) {
      try {
        e.currentTarget.releasePointerCapture(e.pointerId);
      } catch {}
    }
    updateVol(e);
    isVolDrag.value = false;
    localVol.value = null;
  }
}

function onVolWheel(e) {
  e.preventDefault();
  const step = e.deltaMode === 1 ? 0.05 : 0.02;
  const delta = e.deltaY > 0 ? -step : step;
  const next = Math.min(1, Math.max(0, currentVol.value + delta));
  localVol.value = next;
  emit("volume", next);
  clearTimeout(volWheelTimer);
  volWheelTimer = setTimeout(() => {
    localVol.value = null;
  }, 300);
}

function toggleMute() {
  const v = currentVol.value;
  if (v > 0) {
    lastVol = v;
    emit("volume", 0);
  } else {
    emit("volume", lastVol || 0.8);
  }
}
</script>

<template>
  <footer class="player">
    <div ref="bar" class="progress" @pointerdown="down">
      <div class="progress-fill" :style="{ width: progress * 100 + '%' }"></div>
      <div class="thumb" :style="{ left: progress * 100 + '%' }"></div>
    </div>

    <div
      class="now"
      :class="{ clickable: !!track }"
      :title="track ? '打开播放页' : ''"
      @click="track && emit('lyrics')"
    >
      <img
        v-if="track && cover(200)"
        :src="cover(200)"
        alt=""
        draggable="false"
      />
      <div v-else class="cover-empty"><Icon name="music" /></div>
      <div class="meta">
        <div class="name" :title="track?.name">{{ track?.name || "未在播放" }}</div>
        <div class="singer" :title="track?.singer">
          {{ track?.singer || "搜索并播放一首歌" }}
        </div>
      </div>
    </div>

    <div class="controls">
      <button class="side" title="上一首" @click="emit('prev')">
        <Icon name="prev" />
      </button>
      <button class="main" :disabled="!track" @click="emit('toggle')">
        <Icon :name="state?.playing ? 'pause' : 'play'" />
      </button>
      <button class="side" title="下一首" @click="emit('next')">
        <Icon name="next" />
      </button>
    </div>

    <div class="right">
      <span class="time">{{ time(position) }} / {{ time(duration) }}</span>
      <span v-if="state?.loading" class="quality">缓冲中</span>
      <select class="quality" :value="pref" title="音质" @change="pickQuality">
        <option value="128">标准</option>
        <option value="320">高品质</option>
        <option value="flac">无损</option>
      </select>
      <button class="side mode" :title="'播放模式：' + modeLabel" @click="emit('mode')">
        {{ modeLabel }}
      </button>
      <button class="side" title="歌词" @click="emit('lyrics')">
        <Icon name="lyrics" />
      </button>
      <button class="side comment" title="播放队列" @click="emit('queue')">队列</button>
      <button class="side comment" title="评论" @click="emit('comments')">评论</button>
      <button
        class="side"
        :class="{ liked }"
        :title="liked ? '从「我喜欢」移除' : '添加到「我喜欢」'"
        @click="emit('favorite')"
      >
        <Icon name="heart" />
      </button>
      <button
        class="side"
        title="收藏到歌单"
        @click="emit('collect', track)"
      >
        <Icon name="folderPlus" />
      </button>
      <button class="side" title="下载到 Music/mygou" @click="emit('download')">↓</button>
      <div class="vol-wrap" @wheel.passive="onVolWheel">
        <button
          class="side vol-btn"
          :title="currentVol === 0 ? '取消静音' : `静音 (${volPercent}%)`"
          @click="toggleMute"
        >
          <Icon :name="currentVol === 0 ? 'mute' : 'volume'" />
        </button>
        <div
          ref="vol"
          class="volume"
          :class="{ dragging: isVolDrag }"
          :title="`音量: ${volPercent}%`"
          @pointerdown="vdown"
          @pointermove="vmove"
          @pointerup="vup"
          @pointercancel="vup"
        >
          <div class="volume-track">
            <div class="volume-fill" :style="{ width: volPercent + '%' }"></div>
            <div class="volume-dot" :style="{ left: volPercent + '%' }"></div>
          </div>
        </div>
      </div>
    </div>

    <div v-if="state?.error" class="error">{{ state.error }}</div>
  </footer>
</template>

<style scoped>
.player {
  position: fixed;
  inset: auto 0 0 0;
  height: var(--player-height);
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 0 20px;
  background: var(--color-navbar-bg);
  backdrop-filter: saturate(180%) blur(20px);
  -webkit-backdrop-filter: saturate(180%) blur(20px);
  border-top: 1px solid var(--color-secondary-bg-for-transparent);
  z-index: 30;
}
.progress {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--color-secondary-bg-for-transparent);
  cursor: pointer;
}
.progress-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 0 2px 2px 0;
}
.thumb {
  position: absolute;
  top: 50%;
  width: 11px;
  height: 11px;
  margin: -5.5px 0 0 -5.5px;
  border-radius: 50%;
  background: var(--color-primary);
  opacity: 0;
  transition: opacity 0.15s ease;
}
.player:hover .thumb {
  opacity: 1;
}
.now {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 260px;
  min-width: 0;
}
.now img {
  cursor: pointer;
}
.now.clickable {
  cursor: pointer;
}
.now.clickable:hover .name {
  color: var(--color-primary);
}
.now img,
.cover-empty {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  object-fit: cover;
  flex: none;
}
.cover-empty {
  display: grid;
  place-items: center;
  background: var(--color-secondary-bg);
  color: var(--color-secondary);
}
.meta {
  min-width: 0;
}
.name,
.singer {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.name {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 2px;
}
.singer {
  font-size: 12px;
  color: var(--color-secondary);
}
.controls {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 18px;
}
.side {
  white-space: nowrap;
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-secondary);
  cursor: pointer;
}
.side.liked {
  color: var(--color-primary);
}
.side.liked :deep(.icon) {
  fill: currentColor;
}
.side:hover {
  color: var(--color-text);
}
.main {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-text);
  color: var(--color-body-bg);
  cursor: pointer;
}
.main:disabled {
  opacity: 0.4;
}
.right {
  display: flex;
  align-items: center;
  gap: 10px;
  width: auto;
  justify-content: flex-end;
  color: var(--color-secondary);
}
.vol-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
}
.time {
  font-size: 12px;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.side.comment,
.side.mode {
  width: auto;
  padding: 0 6px;
  font-size: 12px;
}
.quality {
  appearance: none;
  -webkit-appearance: none;
  font: inherit;
  font-size: 11px;
  color: var(--color-secondary);
  background: transparent;
  padding: 3px 6px;
  border-radius: 5px;
  border: 1px solid var(--color-secondary-bg-for-transparent);
  cursor: pointer;
}
.quality:hover {
  color: var(--color-text);
}
.actual {
  font-size: 11px;
  padding: 3px 7px;
  border-radius: 5px;
  color: var(--color-primary);
  background: var(--color-primary-bg-for-transparent);
}
.volume {
  position: relative;
  width: 84px;
  height: 24px;
  display: flex;
  align-items: center;
  cursor: pointer;
  touch-action: none;
  user-select: none;
}
.volume-track {
  position: relative;
  width: 100%;
  height: 3px;
  border-radius: 2px;
  background: var(--color-secondary-bg-for-transparent);
}
.volume-fill {
  height: 100%;
  border-radius: 2px;
  background: var(--color-text);
  position: relative;
}
.volume-dot {
  position: absolute;
  top: 50%;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-text);
  transform: translate(-50%, -50%);
  opacity: 0;
  transition: opacity 0.15s ease, transform 0.1s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.25);
  pointer-events: none;
}
.volume:hover .volume-dot,
.volume.dragging .volume-dot {
  opacity: 1;
}
.volume.dragging .volume-dot {
  transform: translate(-50%, -50%) scale(1.15);
}
.error {
  position: absolute;
  top: -28px;
  left: 20px;
  font-size: 12px;
  color: #ff6b6b;
  background: var(--color-secondary-bg);
  padding: 4px 8px;
  border-radius: 6px;
}
</style>
