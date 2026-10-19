<script setup>
import { ref, watch } from "vue";

const props = defineProps({
  quality: { type: String, default: "320" },
  mvQuality: { type: String, default: "fhd" },
  cache: { type: Object, default: () => ({ bytes: 0, files: 0 }) },
  statusBarLyric: { type: Boolean, default: true },
});
const emit = defineEmits(["close", "set-quality", "set-mv-quality", "clear-cache", "set-status-bar-lyric"]);

const coverSize = ref(Number(localStorage.getItem("mygou-cover")) || 300);
const lyricSize = ref(Number(localStorage.getItem("mygou-lyric")) || 17);
watch(coverSize, (v) => localStorage.setItem("mygou-cover", String(v)));
watch(lyricSize, (v) => localStorage.setItem("mygou-lyric", String(v)));

function size(bytes) {
  const n = Number(bytes) || 0;
  if (n >= 1024 * 1024 * 1024) return (n / 1024 / 1024 / 1024).toFixed(1) + " GB";
  if (n >= 1024 * 1024) return (n / 1024 / 1024).toFixed(1) + " MB";
  if (n >= 1024) return (n / 1024).toFixed(0) + " KB";
  return n + " B";
}
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <section class="panel">
      <header>
        <h2>设置</h2>
        <button class="close" title="关闭" @click="emit('close')">✕</button>
      </header>

      <div class="section">
        <div class="label">
          <div class="title">播放音质</div>
          <div class="hint">无损和高品质需要 VIP；不可用时自动降级。</div>
        </div>
        <select
          class="select"
          :value="quality"
          @change="emit('set-quality', $event.target.value)"
        >
          <option value="128">标准 128 kbps</option>
          <option value="320">高品质 320 kbps</option>
          <option value="flac">无损 FLAC</option>
        </select>
      </div>

      <div class="section">
        <div class="label">
          <div class="title">MV 清晰度</div>
          <div class="hint">按档位取流，不可用时自动降档。</div>
        </div>
        <select
          class="select"
          :value="mvQuality"
          @change="emit('set-mv-quality', $event.target.value)"
        >
          <option value="fhd">蓝光</option>
          <option value="qhd">超清</option>
          <option value="hd">高清</option>
          <option value="sd">标清</option>
          <option value="ld">流畅</option>
        </select>
      </div>

      <div class="section">
        <div class="label">
          <div class="title">播放页显示</div>
          <div class="hint">封面大小 {{ coverSize }}px · 歌词字号 {{ lyricSize }}px</div>
        </div>
        <div class="sliders">
          <label>封面 <input v-model.number="coverSize" type="range" min="180" max="440" /></label>
          <label>歌词 <input v-model.number="lyricSize" type="range" min="13" max="26" /></label>
        </div>
      </div>

      <div class="section">
        <div class="label">
          <div class="title">状态栏歌词</div>
          <div class="hint">在 macOS 顶部菜单栏实时显示正在播放的歌词</div>
        </div>
        <button
          class="switch-btn"
          :class="{ on: statusBarLyric }"
          :title="statusBarLyric ? '点击关闭状态栏歌词' : '点击开启状态栏歌词'"
          @click="emit('set-status-bar-lyric', !statusBarLyric)"
        >
          <span class="knob"></span>
        </button>
      </div>

      <div class="section">
        <div class="label">
          <div class="title">播放缓存</div>
          <div class="hint">{{ cache.files || 0 }} 个文件 · {{ size(cache.bytes) }}</div>
        </div>
        <button class="ghost" @click="emit('clear-cache')">清理</button>
      </div>

      <div class="section about">
        <div class="label">
          <div class="title">mygou</div>
          <div class="hint">Go 服务 + 系统 WebView，后台播放，低内存。</div>
        </div>
        <span class="version">0.1.0</span>
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
  width: min(460px, 92vw);
  border-radius: 18px;
  padding: 20px 24px 14px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  box-shadow: 0 30px 80px rgba(0, 0, 0, 0.45);
}
header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding-bottom: 10px;
}
h2 {
  margin: 0;
  font-size: 19px;
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
.section {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 0;
  border-top: 1px solid var(--color-border);
}
.title {
  font-size: 13px;
  font-weight: 600;
}
.hint {
  margin-top: 4px;
  font-size: 11px;
  color: var(--color-secondary);
}
.select {
  flex: none;
  height: 32px;
  padding: 0 10px;
  border-radius: 8px;
  font-size: 12px;
  color: var(--color-text);
  background: var(--color-secondary-bg);
  border: 1px solid var(--color-border);
}
.sliders {
  display: grid;
  gap: 8px;
  width: 170px;
}
.sliders label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 11px;
  color: var(--color-secondary);
}
.sliders input[type="range"] {
  flex: 1;
  accent-color: var(--color-primary);
  cursor: pointer;
}
.switch-btn {
  width: 44px;
  height: 24px;
  border-radius: 12px;
  background: var(--color-secondary-bg);
  border: 1px solid var(--color-border);
  position: relative;
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 0;
  flex: none;
}
.switch-btn.on {
  background: var(--color-primary);
  border-color: var(--color-primary);
}
.knob {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background: #fff;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}
.switch-btn.on .knob {
  transform: translateX(20px);
}
.ghost {
  flex: none;
  height: 30px;
  padding: 0 16px;
  border-radius: 15px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.ghost:hover {
  color: var(--color-text);
}
.version {
  font-size: 12px;
  color: var(--color-secondary);
}
</style>
