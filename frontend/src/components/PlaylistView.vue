<script setup>
import Icon from "./Icon.vue";
import TrackTable from "./TrackTable.vue";

const props = defineProps({
  page: { type: Object, default: null },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["play", "artist", "album", "back"]);

function cover(size = 360) {
  const c = props.page?.playlist?.cover || "";
  return c ? c.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}
</script>

<template>
  <section class="playlist">
    <button class="back" @click="emit('back')">← 我的音乐</button>
    <div v-if="loading && !page" class="hint">正在载入…</div>
    <div v-else-if="error" class="hint error">{{ error }}</div>
    <template v-else-if="page">
      <header class="head">
        <div class="art">
          <img v-if="cover()" :src="cover()" alt="" />
          <Icon v-else name="music" />
        </div>
        <div class="info">
          <div class="kind">歌单</div>
          <h1>{{ page.playlist.name }}</h1>
          <div class="meta">
            <span>{{ page.playlist.creator }}</span>
            <span v-if="page.tracks.length">·</span>
            <span>{{ page.tracks.length }} 首</span>
          </div>
          <button class="playall" @click="emit('play', 0)">
            <Icon name="play" />
            播放全部
          </button>
        </div>
      </header>
      <TrackTable
        :tracks="page.tracks"
        :current-hash="currentHash"
        :playing="playing"
        @play="(i) => emit('play', i)"
        @artist="(t) => emit('artist', t)"
        @album="(t) => emit('album', t)"
      />
    </template>
  </section>
</template>

<style scoped>
.playlist {
  padding-bottom: 40px;
}
.back {
  margin: 22px 0 14px;
  font-size: 13px;
  color: var(--color-secondary);
}
.back:hover {
  color: var(--color-text);
}
.head {
  display: flex;
  gap: 26px;
  align-items: flex-end;
  margin: 8px 0 30px;
}
.art {
  width: 200px;
  height: 200px;
  border-radius: 14px;
  overflow: hidden;
  background: var(--color-secondary-bg);
  display: grid;
  place-items: center;
  color: var(--color-secondary);
  box-shadow: 0 14px 40px rgba(0, 0, 0, 0.28);
  flex: none;
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.kind {
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-secondary);
}
h1 {
  margin: 10px 0 10px;
  font-size: 32px;
  line-height: 1.2;
}
.meta {
  display: flex;
  gap: 8px;
  font-size: 13px;
  color: var(--color-secondary);
}
.playall {
  margin-top: 18px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 22px;
  border-radius: 22px;
  background: var(--color-primary);
  color: #fff;
  font-weight: 600;
  font-size: 14px;
}
.playall .icon {
  width: 16px;
  height: 16px;
}
.hint {
  padding: 40px 0;
  color: var(--color-secondary);
}
.hint.error {
  color: #ff7a7a;
}
</style>
