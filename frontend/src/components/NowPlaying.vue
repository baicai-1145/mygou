<script setup>
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({
  state: { type: Object, default: null },
  position: { type: Number, default: 0 },
  mode: { type: String, default: "list" },
  liked: { type: Boolean, default: false },
});
const emit = defineEmits([
  "close",
  "toggle",
  "prev",
  "next",
  "seek",
  "volume",
  "mode",
  "favorite",
  "collect",
]);

const call = (method, ...args) => window.mygo.call(method, ...args);

const track = computed(() => props.state?.track || null);
const cover = computed(() => {
  const c = track.value?.cover || "";
  return c ? c.replace("{size}", "500").replace(/^http:/, "https:") : "";
});
const bgCover = computed(() => {
  const c = track.value?.cover || "";
  return c ? c.replace("{size}", "400").replace(/^http:/, "https:") : "";
});

// Artwork-Driven Ambient Stage: 100% genuine cover artwork, zero canvas leak
const bgStyle = computed(() => ({
  backgroundImage: bgCover.value ? `url("${bgCover.value}")` : "none",
}));

// Fullscreen support
const isFullscreen = ref(false);

async function checkFullscreen() {
  try {
    isFullscreen.value = await call("App.IsFullScreen");
  } catch {
    isFullscreen.value = Boolean(document.fullscreenElement);
  }
}

async function toggleFullscreen() {
  try {
    const next = await call("App.ToggleFullScreen");
    isFullscreen.value = next;
  } catch {
    if (window.mygo?.window?.toggleFullScreen) {
      await window.mygo.window.toggleFullScreen();
      await checkFullscreen();
    } else if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen().catch(() => {});
    } else {
      document.exitFullscreen().catch(() => {});
    }
  }
}

async function onKeydown(e) {
  if (e.key === "Escape") {
    if (isFullscreen.value) {
      e.preventDefault();
      await toggleFullscreen();
    } else {
      emit("close");
    }
  }
}

function onFullscreenEvent(e) {
  if (e?.detail && typeof e.detail.fullscreen === "boolean") {
    isFullscreen.value = e.detail.fullscreen;
  } else {
    checkFullscreen();
  }
}

onMounted(() => {
  checkFullscreen();
  window.addEventListener("mygo:fullscreen", onFullscreenEvent);
  document.addEventListener("fullscreenchange", checkFullscreen);
  window.addEventListener("resize", checkFullscreen);
  window.addEventListener("keydown", onKeydown);
});

onUnmounted(() => {
  window.removeEventListener("mygo:fullscreen", onFullscreenEvent);
  document.removeEventListener("fullscreenchange", checkFullscreen);
  window.removeEventListener("resize", checkFullscreen);
  window.removeEventListener("keydown", onKeydown);
});

// Volume handling
const volRef = ref(null);
const isVolDragging = ref(false);
const localVol = ref(null);
let lastVol = 1;
let volWheelTimer = null;

const currentVol = computed(() => {
  if (localVol.value !== null) return localVol.value;
  return props.state?.volume ?? 1;
});
const isMuted = computed(() => currentVol.value === 0);
const volPercent = computed(() => Math.round(currentVol.value * 100));

function onVolDown(e) {
  isVolDragging.value = true;
  if (e.currentTarget?.setPointerCapture) {
    try {
      e.currentTarget.setPointerCapture(e.pointerId);
    } catch {}
  }
  updateVol(e);
}
function onVolMove(e) {
  if (isVolDragging.value) {
    updateVol(e);
  }
}
function onVolUp(e) {
  if (isVolDragging.value) {
    if (e.currentTarget?.releasePointerCapture) {
      try {
        e.currentTarget.releasePointerCapture(e.pointerId);
      } catch {}
    }
    updateVol(e);
    isVolDragging.value = false;
    localVol.value = null;
  }
}
function updateVol(e) {
  if (!volRef.value) return;
  const rect = volRef.value.getBoundingClientRect();
  const val = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
  localVol.value = val;
  emit("volume", val);
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

// Progress handling
const progressRef = ref(null);
let seeking = false;
const seekPos = ref(0);
const duration = computed(() => props.state?.duration || 0);
const displayPos = computed(() => (seeking ? seekPos.value : props.position));
const progressPercent = computed(() => {
  if (!duration.value) return 0;
  return Math.min(100, Math.max(0, (displayPos.value / duration.value) * 100));
});

function onProgDown(e) {
  seeking = true;
  updateProg(e);
  window.addEventListener("pointermove", onProgMove);
  window.addEventListener("pointerup", onProgUp);
}
function onProgMove(e) {
  if (seeking) updateProg(e);
}
function onProgUp(e) {
  if (seeking) {
    updateProg(e);
    emit("seek", seekPos.value);
    seeking = false;
  }
  window.removeEventListener("pointermove", onProgMove);
  window.removeEventListener("pointerup", onProgUp);
}
function updateProg(e) {
  if (!progressRef.value || !duration.value) return;
  const rect = progressRef.value.getBoundingClientRect();
  const ratio = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width));
  seekPos.value = ratio * duration.value;
}

function time(sec) {
  if (!sec || isNaN(sec) || sec < 0) return "0:00";
  const m = Math.floor(sec / 60);
  const s = Math.floor(sec % 60);
  return `${m}:${s < 10 ? "0" : ""}${s}`;
}

// Lyrics handling
const lines = ref([]);
const loading = ref(false);
const error = ref("");
const box = ref(null);
const current = ref(-1);
const showTrans = ref(true);

const hasTranslation = computed(() =>
  lines.value.some((l) => Boolean(l.trans && l.trans.trim() && l.trans !== l.text)),
);

function parseTranslations(text) {
  for (const raw of String(text).split(/\r?\n/)) {
    const m = raw.match(/^\[language:(.+)\]\s*$/);
    if (!m) continue;
    try {
      const data = JSON.parse(atob(m[1]));
      const list = data.content || [];
      const zh = list.find((c) => c.type === 1) || list[0];
      return (zh?.lyricContent || []).map((parts) => {
        const s = parts.join("").trim();
        if (/享有本翻译作品|著作权|未经许可|翻译[：:]/i.test(s)) return "";
        return s;
      });
    } catch {
      return [];
    }
  }
  return [];
}

function parse(text) {
  const out = [];
  const translations = parseTranslations(text);
  let timed = 0;
  for (const raw of String(text).split(/\r?\n/)) {
    let m = raw.match(/^\[(\d+),(\d+)\](.*)$/); // KRC: [start,duration]text
    if (m) {
      const lyric = m[3].replace(/<[^>]*>/g, "").trim();
      const trans = translations[timed] || "";
      timed++;
      if (lyric) out.push({ t: Number(m[1]) / 1000, text: lyric, trans });
      continue;
    }
    m = raw.match(/^\[(\d+):(\d+)(?:[.:](\d+))?\](.*)$/); // LRC
    if (m) {
      const frac = m[3] ? Number(`0.${m[3]}`) : 0;
      const lyric = m[4].trim();
      if (lyric) out.push({ t: Number(m[1]) * 60 + Number(m[2]) + frac, text: lyric, trans: "" });
    }
  }
  out.sort((a, b) => a.t - b.t);
  return out;
}

async function load() {
  if (!track.value?.hash) {
    lines.value = [];
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    const l = await call("App.Lyric", track.value.hash, track.value.name, track.value.duration || 0);
    lines.value = parse(l.text);
    if (!lines.value.length) error.value = "这首歌还没有歌词";
  } catch (e) {
    lines.value = [];
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

watch(
  () => track.value?.hash,
  () => {
    current.value = -1;
    load();
  },
  { immediate: true },
);

watch(
  () => props.position,
  (pos) => {
    if (!lines.value.length) return;
    let idx = -1;
    for (let i = 0; i < lines.value.length; i++) {
      if (lines.value[i].t <= pos + 0.25) idx = i;
      else break;
    }
    if (idx !== current.value) {
      current.value = idx;
      nextTick(() => {
        const el = box.value?.querySelector(".line.highlight");
        el?.scrollIntoView({ block: "center", behavior: "smooth" });
      });
    }
  },
  { immediate: true },
);

function jump(line) {
  emit("seek", Math.max(0, line.t));
}

const modeIcon = computed(() => {
  if (props.mode === "one") return "repeatOne";
  if (props.mode === "shuffle") return "shuffle";
  return "repeat";
});

const modeTitle = computed(() => {
  if (props.mode === "one") return "单曲循环";
  if (props.mode === "shuffle") return "随机播放";
  return "列表循环";
});
</script>

<template>
  <div class="lyrics-page" :class="{ 'is-fullscreen': isFullscreen }">
    <!-- Artwork-Driven Ambient Stage (Zero Canvas Leak, Authentic Album Colors) -->
    <div class="artwork-stage">
      <div class="artwork-backdrop" :style="bgStyle"></div>
      <div class="artwork-drift" :style="bgStyle"></div>
    </div>
    <div class="veil"></div>

    <!-- Top window controls -->
    <div class="top-bar">
      <div class="left-tools">
        <div class="traffic-light-pad"></div>
      </div>

      <div class="right-tools">
        <button
          class="top-icon-btn"
          :title="isFullscreen ? '退出全屏 (Esc)' : '全屏'"
          @click="toggleFullscreen"
        >
          <Icon :name="isFullscreen ? 'fullscreenExit' : 'fullscreen'" />
        </button>
        <button class="top-icon-btn close-btn" title="收起 (Esc)" @click="emit('close')">
          <svg viewBox="0 0 24 24" class="arrow-down" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M6 9l6 6 6-6" />
          </svg>
        </button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="main-body">
      <!-- Left side: Cover & Player Controls -->
      <div class="left-side">
        <div class="left-side-inner">
          <div class="cover">
            <div class="cover-container">
              <img v-if="cover" :src="cover" alt="" draggable="false" />
              <div v-else class="art-empty">
                <Icon name="music" />
              </div>
            </div>
          </div>

          <div class="controls">
            <div class="top-part">
              <div class="track-info">
                <div class="title" :title="track?.name || '未在播放'">
                  {{ track?.name || "未在播放" }}
                </div>
                <div class="subtitle" :title="(track?.singer || '') + (track?.album ? ' - ' + track.album : '')">
                  <span>{{ track?.singer || "酷狗音乐" }}</span>
                  <span v-if="track?.album"> - {{ track.album }}</span>
                </div>
              </div>

              <div class="top-right-tools">
                <div class="volume-control" @wheel.passive="onVolWheel">
                  <button
                    class="tool-btn vol-btn"
                    :title="isMuted ? '取消静音' : `静音 (${volPercent}%)`"
                    @click="toggleMute"
                  >
                    <Icon :name="isMuted ? 'mute' : 'volume'" />
                  </button>
                  <div
                    ref="volRef"
                    class="volume-bar"
                    :class="{ dragging: isVolDragging }"
                    :title="`音量: ${volPercent}%`"
                    @pointerdown="onVolDown"
                    @pointermove="onVolMove"
                    @pointerup="onVolUp"
                    @pointercancel="onVolUp"
                  >
                    <div class="vol-track">
                      <div class="vol-fill" :style="{ width: volPercent + '%' }"></div>
                      <div class="vol-dot" :style="{ left: volPercent + '%' }"></div>
                    </div>
                  </div>
                </div>

                <div class="buttons">
                  <button
                    class="tool-btn fav-btn"
                    :class="{ liked }"
                    :title="liked ? '从「我喜欢」移除' : '添加到「我喜欢」'"
                    @click="emit('favorite')"
                  >
                    <Icon name="heart" />
                  </button>
                  <button
                    class="tool-btn"
                    title="收藏到歌单"
                    @click="emit('collect', track)"
                  >
                    <Icon name="plus" />
                  </button>
                </div>
              </div>
            </div>

            <!-- Progress Bar -->
            <div class="progress-bar">
              <span class="p-time">{{ time(displayPos) }}</span>
              <div ref="progressRef" class="slider-track" @pointerdown="onProgDown">
                <div class="slider-fill" :style="{ width: progressPercent + '%' }"></div>
                <div class="slider-dot" :style="{ left: progressPercent + '%' }"></div>
              </div>
              <span class="p-time">{{ time(duration) }}</span>
            </div>

            <!-- Media Controls -->
            <div class="media-controls">
              <button
                class="ctrl-icon-btn"
                :title="modeTitle"
                @click="emit('mode')"
              >
                <Icon :name="modeIcon" />
              </button>

              <div class="middle">
                <button
                  class="ctrl-icon-btn"
                  title="上一首"
                  @click="emit('prev')"
                >
                  <Icon name="prev" />
                </button>

                <button
                  class="play-circle-btn"
                  :title="state?.playing ? '暂停' : '播放'"
                  @click="emit('toggle')"
                >
                  <Icon :name="state?.playing ? 'pause' : 'play'" />
                </button>

                <button
                  class="ctrl-icon-btn"
                  title="下一首"
                  @click="emit('next')"
                >
                  <Icon name="next" />
                </button>
              </div>

              <button
                class="trans-toggle-btn"
                :class="{ active: showTrans && hasTranslation, disabled: !hasTranslation }"
                :disabled="!hasTranslation"
                :title="hasTranslation ? '切换歌词翻译' : '当前歌曲暂无翻译'"
                @click="hasTranslation && (showTrans = !showTrans)"
              >
                <span>译</span>
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Right side: Left-aligned Scrolling Lyrics -->
      <div class="right-side">
        <div ref="box" class="lyrics-container">
          <div class="line-pad line-top-pad"></div>

          <div v-if="loading" class="hint">正在载入歌词…</div>
          <div v-else-if="error" class="hint">{{ error }}</div>
          <template v-else>
            <div
              v-for="(line, idx) in lines"
              :id="`line${idx}`"
              :key="idx"
              class="line"
              :class="{ highlight: idx === current }"
              @click="jump(line)"
            >
              <div class="content">
                <span class="main-text">{{ line.text }}</span>
                <span
                  v-if="line.trans && showTrans && line.trans !== line.text"
                  class="trans-text"
                >
                  {{ line.trans }}
                </span>
              </div>
            </div>
          </template>

          <div class="line-pad line-bottom-pad"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.lyrics-page {
  position: fixed;
  inset: 0;
  z-index: 100;
  background: #0c0c0e;
  color: #fff;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  user-select: none;
  -webkit-user-select: none;
  --app-region: drag;
  -webkit-app-region: drag;
}

/* Artwork-Driven Ambient Stage (Apple Music Style, Zero Canvas Leak) */
.artwork-stage {
  position: absolute;
  inset: -60px;
  overflow: hidden;
  pointer-events: none;
  z-index: 0;
  background: #09090d;
}

:root[data-hidden="true"] .artwork-stage * {
  animation-play-state: paused !important;
}

.artwork-backdrop {
  position: absolute;
  inset: -40px;
  background-size: cover;
  background-position: center;
  filter: blur(48px) saturate(240%) brightness(66%);
  -webkit-filter: blur(48px) saturate(240%) brightness(66%);
  transform: scale(1.12);
  animation: artwork-breathe 28s ease-in-out infinite alternate;
  will-change: transform;
}

.artwork-drift {
  position: absolute;
  inset: -60px;
  background-size: cover;
  background-position: center;
  filter: blur(56px) saturate(260%) brightness(62%);
  -webkit-filter: blur(56px) saturate(260%) brightness(62%);
  opacity: 0.75;
  mix-blend-mode: screen;
  transform: scale(1.22);
  animation: artwork-drift 36s ease-in-out infinite alternate;
  will-change: transform;
}

@keyframes artwork-breathe {
  0% { transform: scale(1.1) rotate(0deg); }
  50% { transform: scale(1.18) translate(2%, 1%) rotate(2deg); }
  100% { transform: scale(1.12) translate(-2%, 2%) rotate(-1deg); }
}

@keyframes artwork-drift {
  0% { transform: scale(1.25) translate(-3%, -2%) rotate(0deg); }
  50% { transform: scale(1.16) translate(3%, 3%) rotate(-3deg); }
  100% { transform: scale(1.22) translate(-1%, 4%) rotate(2deg); }
}

.veil {
  position: absolute;
  inset: 0;
  background: rgba(8, 8, 12, 0.18);
  z-index: 1;
  pointer-events: none;
}

/* Top bar with traffic light buffer and close/fullscreen */
.top-bar {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 64px;
  padding: 0 28px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  z-index: 30;
  transition: padding 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
  --app-region: drag;
  -webkit-app-region: drag;
}
.left-tools {
  display: flex;
  align-items: center;
}
.traffic-light-pad {
  width: 70px;
  height: 20px;
  transition: width 0.5s cubic-bezier(0.2, 0.9, 0.3, 1), opacity 0.35s ease;
}
.is-fullscreen .top-bar {
  padding: 0 44px;
}
.is-fullscreen .traffic-light-pad {
  width: 0;
  opacity: 0;
}
.right-tools {
  display: flex;
  align-items: center;
  gap: 10px;
  --app-region: no-drag;
  -webkit-app-region: no-drag;
}
.top-icon-btn {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  display: grid;
  place-items: center;
  background: rgba(255, 255, 255, 0.10);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border: 1px solid rgba(255, 255, 255, 0.16);
  color: rgba(255, 255, 255, 0.85);
  cursor: pointer;
  box-shadow:
    inset 0 1px 1px rgba(255, 255, 255, 0.28),
    0 4px 14px rgba(0, 0, 0, 0.25);
  transition: all 0.2s cubic-bezier(0.2, 0.9, 0.3, 1);
  --app-region: no-drag;
  -webkit-app-region: no-drag;
}
.top-icon-btn:hover {
  background: rgba(255, 255, 255, 0.18);
  border-color: rgba(255, 255, 255, 0.32);
  color: #fff;
  transform: scale(1.06);
  box-shadow:
    inset 0 1px 1.5px rgba(255, 255, 255, 0.4),
    0 6px 20px rgba(0, 0, 0, 0.32);
}
.top-icon-btn:active {
  transform: scale(0.94);
}
.arrow-down {
  width: 20px;
  height: 20px;
}

/* Main body split */
.main-body {
  position: relative;
  z-index: 10;
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  padding: 0;
  box-sizing: border-box;
  --app-region: drag;
  -webkit-app-region: drag;
}

/* Left side */
.left-side {
  flex: 1;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  padding-right: 48px;
  box-sizing: border-box;
  transition: padding 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.left-side-inner {
  width: min(45vh, 390px);
  display: flex;
  flex-direction: column;
  transition: width 0.5s cubic-bezier(0.2, 0.9, 0.3, 1), transform 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.cover {
  position: relative;
  width: 100%;
}
.cover-container {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  border-radius: 20px;
  box-shadow: 0 20px 48px -10px rgba(0, 0, 0, 0.65), 0 10px 24px -8px rgba(0, 0, 0, 0.4);
  transition: border-radius 0.5s cubic-bezier(0.2, 0.9, 0.3, 1), box-shadow 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.cover-container img,
.art-empty {
  width: 100%;
  height: 100%;
  border-radius: 20px;
  object-fit: cover;
  display: block;
  transition: border-radius 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.art-empty {
  background: rgba(255, 255, 255, 0.08);
  display: grid;
  place-items: center;
  color: rgba(255, 255, 255, 0.3);
}

/* Apple Liquid Frosted Glass Floating Controls Card */
.controls {
  margin-top: 22px;
  width: 100%;
  padding: 18px 20px 16px;
  background: rgba(255, 255, 255, 0.08);
  backdrop-filter: blur(28px) saturate(180%);
  -webkit-backdrop-filter: blur(28px) saturate(180%);
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 22px;
  box-shadow:
    inset 0 1px 1.5px 0 rgba(255, 255, 255, 0.32),
    0 20px 48px rgba(0, 0, 0, 0.35);
  transition: all 0.3s cubic-bezier(0.2, 0.9, 0.3, 1);
  --app-region: no-drag;
  -webkit-app-region: no-drag;
}
.controls:hover {
  background: rgba(255, 255, 255, 0.11);
  border-color: rgba(255, 255, 255, 0.24);
}

.top-part {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.track-info {
  min-width: 0;
  flex: 1;
}
.title {
  font-size: 1.35rem;
  font-weight: 700;
  color: #fff;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  letter-spacing: -0.01em;
  transition: font-size 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.subtitle {
  margin-top: 4px;
  font-size: 0.92rem;
  color: rgba(255, 255, 255, 0.6);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  transition: font-size 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.top-right-tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: none;
}
.volume-control {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-right: 4px;
}
.tool-btn {
  width: 32px;
  height: 32px;
  border-radius: 9px;
  display: grid;
  place-items: center;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.7);
  box-shadow: inset 0 0.5px 0.5px rgba(255, 255, 255, 0.2);
  cursor: pointer;
  padding: 0;
  transition: all 0.18s ease;
}
.tool-btn:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.16);
  border-color: rgba(255, 255, 255, 0.22);
  transform: scale(1.06);
}
.fav-btn.liked {
  color: #ff3b30;
  background: rgba(255, 59, 48, 0.15);
  border-color: rgba(255, 59, 48, 0.3);
}
.fav-btn.liked :deep(.icon) {
  fill: currentColor;
}
.volume-bar {
  position: relative;
  width: 72px;
  height: 24px;
  display: flex;
  align-items: center;
  cursor: pointer;
  touch-action: none;
  user-select: none;
}
.vol-track {
  position: relative;
  width: 100%;
  height: 4px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.18);
  transition: height 0.15s ease;
}
.volume-bar:hover .vol-track {
  height: 5px;
}
.vol-fill {
  height: 100%;
  border-radius: 2px;
  background: #ffffff;
  box-shadow: 0 0 8px rgba(255, 255, 255, 0.3);
}
.vol-dot {
  position: absolute;
  top: 50%;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #fff;
  transform: translate(-50%, -50%);
  opacity: 0;
  transition: opacity 0.15s ease, transform 0.1s ease;
  box-shadow: 0 1px 5px rgba(0, 0, 0, 0.4), 0 0 10px rgba(255, 255, 255, 0.45);
  pointer-events: none;
}
.volume-bar:hover .vol-dot,
.volume-bar.dragging .vol-dot {
  opacity: 1;
}
.volume-bar.dragging .vol-dot {
  transform: translate(-50%, -50%) scale(1.15);
}
.buttons {
  display: flex;
  align-items: center;
  gap: 6px;
}

/* Progress bar */
.progress-bar {
  margin-top: 16px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.p-time {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
  color: rgba(255, 255, 255, 0.58);
  min-width: 32px;
}
.slider-track {
  position: relative;
  flex: 1;
  height: 20px;
  display: flex;
  align-items: center;
  cursor: pointer;
  touch-action: none;
}
.slider-track::before {
  content: "";
  position: absolute;
  left: 0;
  right: 0;
  height: 4px;
  border-radius: 2px;
  background: rgba(255, 255, 255, 0.18);
  transition: height 0.15s ease;
}
.slider-track:hover::before {
  height: 5px;
}
.slider-fill {
  height: 4px;
  border-radius: 2px;
  background: #fff;
  box-shadow: 0 0 10px rgba(255, 255, 255, 0.35);
  position: relative;
}
.slider-dot {
  position: absolute;
  top: 50%;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: #fff;
  transform: translate(-50%, -50%);
  box-shadow: 0 1px 6px rgba(0, 0, 0, 0.45), 0 0 12px rgba(255, 255, 255, 0.5);
  opacity: 0;
  transition: opacity 0.15s ease, transform 0.15s ease;
  pointer-events: none;
}
.slider-track:hover .slider-dot {
  opacity: 1;
}

/* Media controls */
.media-controls {
  margin-top: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.ctrl-icon-btn {
  width: 38px;
  height: 38px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: transparent;
  border: none;
  color: rgba(255, 255, 255, 0.65);
  cursor: pointer;
  transition: all 0.18s ease;
}
.ctrl-icon-btn:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.1);
  transform: scale(1.06);
}
.middle {
  display: flex;
  align-items: center;
  gap: 16px;
}
.middle .ctrl-icon-btn {
  color: rgba(255, 255, 255, 0.88);
}
.play-circle-btn {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: #ffffff;
  color: #121214;
  border: none;
  display: grid;
  place-items: center;
  cursor: pointer;
  transition: transform 0.2s cubic-bezier(0.2, 0.9, 0.3, 1), box-shadow 0.2s ease;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.38), 0 0 22px rgba(255, 255, 255, 0.26);
}
.play-circle-btn:hover {
  transform: scale(1.08);
  box-shadow: 0 8px 28px rgba(0, 0, 0, 0.45), 0 0 32px rgba(255, 255, 255, 0.4);
}
.play-circle-btn:active {
  transform: scale(0.94);
}
.play-circle-btn :deep(.icon) {
  width: 22px;
  height: 22px;
}
.trans-toggle-btn {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: rgba(255, 255, 255, 0.55);
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  transition: all 0.18s ease;
}
.trans-toggle-btn.active {
  color: #fff;
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.35);
  box-shadow: inset 0 1px 1px rgba(255, 255, 255, 0.3);
}
.trans-toggle-btn.disabled {
  opacity: 0.22;
  cursor: not-allowed;
}
.trans-toggle-btn:not(.disabled):hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.14);
}

/* Right side: Scrolling lyrics with Optical Depth of Field */
.right-side {
  flex: 1;
  display: flex;
  justify-content: flex-start;
  align-items: center;
  padding-left: 48px;
  height: 82vh;
  box-sizing: border-box;
  transition: padding 0.5s cubic-bezier(0.2, 0.9, 0.3, 1), height 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.lyrics-container {
  width: 100%;
  max-width: 540px;
  height: 100%;
  overflow-y: auto;
  scrollbar-width: none;
  transition: max-width 0.5s cubic-bezier(0.2, 0.9, 0.3, 1);
  --app-region: no-drag;
  -webkit-app-region: no-drag;
}
.lyrics-container::-webkit-scrollbar {
  display: none;
}
.line-top-pad {
  height: 38vh;
}
.line-bottom-pad {
  height: 45vh;
}
.line {
  padding: 14px 18px;
  margin: 3px 0;
  border-radius: 14px;
  cursor: pointer;
  transition: all 0.25s cubic-bezier(0.2, 0.9, 0.3, 1);
  text-align: left;
}
.line:hover {
  background: rgba(255, 255, 255, 0.06);
}
.content {
  display: flex;
  flex-direction: column;
  transform-origin: center left;
  transition: all 0.38s cubic-bezier(0.2, 0.9, 0.3, 1);
}
.main-text {
  font-size: 1.55rem;
  font-weight: 700;
  color: #fff;
  opacity: 0.38;
  transition: all 0.35s cubic-bezier(0.2, 0.9, 0.3, 1);
  line-height: 1.4;
  letter-spacing: -0.01em;
}
.trans-text {
  font-size: 1.08rem;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.85);
  opacity: 0.24;
  margin-top: 6px;
  transition: all 0.35s cubic-bezier(0.2, 0.9, 0.3, 1);
  line-height: 1.4;
}
.line:hover .main-text {
  opacity: 0.85;
}
.line:hover .trans-text {
  opacity: 0.65;
}
.line.highlight .main-text {
  opacity: 1;
  font-size: 1.85rem;
  font-weight: 800;
  transform: scale(1.02);
  text-shadow: 0 0 28px rgba(255, 255, 255, 0.36);
}
.line.highlight .trans-text {
  opacity: 0.88;
  font-size: 1.3rem;
  font-weight: 600;
  margin-top: 6px;
  text-shadow: 0 0 16px rgba(255, 255, 255, 0.22);
}

/* Fullscreen Dynamic Scaling (Apple Music Proportions) */
.is-fullscreen .left-side {
  padding-right: 76px;
}
.is-fullscreen .left-side-inner {
  width: min(56vh, 540px);
}
.is-fullscreen .cover-container {
  border-radius: 24px;
  box-shadow: 0 32px 72px -12px rgba(0, 0, 0, 0.78), 0 16px 36px -8px rgba(0, 0, 0, 0.5);
}
.is-fullscreen .cover-container img,
.is-fullscreen .art-empty {
  border-radius: 24px;
}
.is-fullscreen .controls {
  margin-top: 26px;
  padding: 22px 26px 20px;
  border-radius: 26px;
}
.is-fullscreen .title {
  font-size: 1.55rem;
}
.is-fullscreen .subtitle {
  font-size: 1.05rem;
}
.is-fullscreen .right-side {
  padding-left: 76px;
  height: 86vh;
}
.is-fullscreen .lyrics-container {
  max-width: 680px;
}
.is-fullscreen .main-text {
  font-size: 1.85rem;
}
.is-fullscreen .line.highlight .main-text {
  font-size: 2.3rem;
}
.is-fullscreen .trans-text {
  font-size: 1.25rem;
}
.is-fullscreen .line.highlight .trans-text {
  font-size: 1.5rem;
}
.hint {
  padding-top: 30vh;
  text-align: left;
  padding-left: 20px;
  color: rgba(255, 255, 255, 0.55);
  font-size: 14px;
}

@media (max-width: 900px) {
  .left-side {
    display: none;
  }
  .right-side {
    max-width: 100%;
  }
}

/* Staggered entrance/exit animations when sliding up/down */
.lyrics-page.slide-up-enter-active .left-side-inner {
  transition: transform 0.44s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.38s ease;
}
.lyrics-page.slide-up-enter-from .left-side-inner {
  transform: translateY(28px) scale(0.97);
  opacity: 0.25;
}

.lyrics-page.slide-up-enter-active .lyrics-container {
  transition: transform 0.48s cubic-bezier(0.16, 1, 0.3, 1), opacity 0.4s ease;
}
.lyrics-page.slide-up-enter-from .lyrics-container {
  transform: translateY(42px);
  opacity: 0.25;
}

.lyrics-page.slide-up-enter-active .top-bar {
  transition: opacity 0.35s ease;
}
.lyrics-page.slide-up-enter-from .top-bar {
  opacity: 0;
}

.lyrics-page.slide-up-leave-active .left-side-inner,
.lyrics-page.slide-up-leave-active .lyrics-container {
  transition: transform 0.26s cubic-bezier(0.4, 0, 0.2, 1), opacity 0.22s ease;
}
.lyrics-page.slide-up-leave-to .left-side-inner {
  transform: translateY(16px) scale(0.98);
  opacity: 0;
}
.lyrics-page.slide-up-leave-to .lyrics-container {
  transform: translateY(24px);
  opacity: 0;
}
.lyrics-page.slide-up-leave-active .top-bar {
  transition: opacity 0.2s ease;
}
.lyrics-page.slide-up-leave-to .top-bar {
  opacity: 0;
}
</style>
