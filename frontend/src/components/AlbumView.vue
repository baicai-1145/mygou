<script setup>
import Icon from "./Icon.vue";
import TrackTable from "./TrackTable.vue";

const props = defineProps({
  album: { type: Object, default: null },
  songs: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["play", "artist", "back"]);

function cover(size = 360) {
  const c = props.album?.cover || "";
  return c ? c.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}
</script>

<template>
  <section class="album">
    <button class="back" @click="emit('back')">← 返回</button>
    <div v-if="loading && !songs.length" class="hint">正在载入…</div>
    <div v-else-if="error" class="hint error">{{ error }}</div>
    <template v-else-if="album">
      <header class="head">
        <div class="art">
          <img v-if="cover()" :src="cover()" alt="" />
          <Icon v-else name="music" />
        </div>
        <div class="info">
          <div class="kind">专辑</div>
          <h1>{{ album.name }}</h1>
          <div class="meta">
            <button v-if="album.artist" class="artist-link" @click="emit('artist', album)">
              {{ album.artist }}
            </button>
            <span v-if="songs.length">· {{ songs.length }} 首</span>
          </div>
          <button class="playall" @click="emit('play', 0)">
            <Icon name="play" />
            播放全部
          </button>
        </div>
      </header>
      <TrackTable
        :tracks="songs"
        :current-hash="currentHash"
        :playing="playing"
        @play="(i) => emit('play', i)"
        @artist="(t) => emit('artist', t)"
        @album="() => {}"
      />
    </template>
  </section>
</template>

<style scoped>
.album {
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
  flex: none;
  border-radius: 14px;
  overflow: hidden;
  display: grid;
  place-items: center;
  color: var(--color-secondary);
  background: linear-gradient(
    135deg,
    var(--color-primary-bg-for-transparent),
    var(--color-secondary-bg)
  );
}
.art img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.info {
  min-width: 0;
}
.kind {
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--color-primary);
  text-transform: uppercase;
  margin-bottom: 8px;
}
h1 {
  margin: 0 0 8px;
  font-size: 34px;
  letter-spacing: -0.02em;
}
.meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: var(--color-secondary);
}
.artist-link {
  color: var(--color-secondary);
  font-size: 12px;
}
.artist-link:hover {
  color: var(--color-primary);
  text-decoration: underline;
}
.playall {
  margin-top: 18px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 20px;
  border-radius: 19px;
  background: var(--color-primary);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
}
.playall .icon {
  width: 14px;
  height: 14px;
}
.hint {
  padding: 30px 0;
  font-size: 13px;
  color: var(--color-secondary);
}
.hint.error {
  color: #e0533f;
}
@media (max-width: 900px) {
  .head {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
