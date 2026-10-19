<script setup>
import Icon from "./Icon.vue";

const props = defineProps({
  video: { type: Object, default: null },
  quality: { type: String, default: "fhd" },
});
const emit = defineEmits(["close", "quality"]);
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <section class="panel">
      <header>
        <div class="titles">
          <h2>{{ video?.name || "MV" }}</h2>
          <p v-if="video?.singer">{{ video.singer }}</p>
        </div>
        <div class="tools">
          <select
            class="q"
            :value="quality"
            @change="emit('quality', $event.target.value)"
          >
            <option value="fhd">蓝光</option>
            <option value="qhd">超清</option>
            <option value="hd">高清</option>
            <option value="sd">标清</option>
            <option value="ld">流畅</option>
          </select>
          <button class="close" title="关闭" @click="emit('close')">✕</button>
        </div>
      </header>
      <div class="stage">
        <video
          v-if="video?.url"
          :src="video.url"
          controls
          autoplay
          playsinline
          class="video"
        ></video>
        <div v-else-if="video?.error" class="hint error">{{ video.error }}</div>
        <div v-else class="hint">
          <Icon name="music" />
          正在解析视频…
        </div>
      </div>
    </section>
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
  background: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(8px);
}
.panel {
  width: min(900px, 92vw);
  border-radius: 18px;
  overflow: hidden;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  box-shadow: 0 30px 90px rgba(0, 0, 0, 0.5);
}
header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 16px 20px 12px;
}
h2 {
  margin: 0;
  font-size: 17px;
  letter-spacing: -0.01em;
}
p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--color-secondary);
}
.tools {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: none;
}
.q {
  height: 26px;
  padding: 0 8px;
  border-radius: 7px;
  font-size: 11px;
  color: var(--color-text);
  background: var(--color-secondary-bg);
  border: 1px solid var(--color-border);
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
.stage {
  aspect-ratio: 16 / 9;
  background: #000;
  display: grid;
  place-items: center;
}
.video {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.hint {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: #aaa;
}
.hint.error {
  color: #e0533f;
}
.hint .icon {
  width: 16px;
  height: 16px;
}
</style>
