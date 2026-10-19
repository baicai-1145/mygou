<script setup>
import Icon from "./Icon.vue";
import TrackTable from "./TrackTable.vue";

const props = defineProps({
  ranks: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  detail: { type: Object, default: null },
  songs: { type: Array, default: () => [] },
  songsLoading: { type: Boolean, default: false },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["open", "play", "artist", "album", "back"]);

function cover(src, size = 220) {
  if (!src) return "";
  return src.replace("{size}", String(size)).replace(/^http:/, "https:");
}
</script>

<template>
  <section class="rank">
    <template v-if="!detail">
      <h1>排行榜</h1>
      <p class="sub">酷狗音乐榜单，每天、每周更新。</p>
      <div v-if="loading" class="hint">正在载入…</div>
      <div v-else-if="error" class="hint error">{{ error }}</div>
      <div v-else class="grid">
        <button v-for="r in ranks" :key="r.id" class="card" @click="emit('open', r)">
          <div class="art">
            <img v-if="r.cover" :src="cover(r.cover)" alt="" loading="lazy" />
            <Icon v-else name="music" />
          </div>
          <div class="name">{{ r.name }}</div>
          <div class="freq">{{ r.frequency }}</div>
        </button>
      </div>
    </template>

    <template v-else>
      <button class="back" @click="emit('back')">← 排行榜</button>
      <header class="head">
        <div class="art big">
          <img v-if="detail.cover" :src="cover(detail.cover, 360)" alt="" />
          <Icon v-else name="music" />
        </div>
        <div class="info">
          <div class="kind">排行榜</div>
          <h1>{{ detail.name }}</h1>
          <div class="meta">{{ detail.frequency }}</div>
          <p v-if="detail.intro" class="intro">{{ detail.intro }}</p>
          <button class="playall" @click="emit('play', 0)">
            <Icon name="play" />
            播放全部
          </button>
        </div>
      </header>
      <div v-if="songsLoading" class="hint">正在载入…</div>
      <TrackTable
        v-else
        :tracks="songs"
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
.rank {
  padding: 8px 0 40px;
}
h1 {
  margin: 22px 0 6px;
  font-size: 28px;
  letter-spacing: -0.02em;
}
.sub {
  margin: 0 0 22px;
  font-size: 13px;
  color: var(--color-secondary);
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 22px;
}
.card {
  text-align: left;
  color: inherit;
}
.art {
  aspect-ratio: 1;
  border-radius: 12px;
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
  transition: transform 0.3s ease;
}
.card:hover .art img {
  transform: scale(1.05);
}
.art.big {
  width: 200px;
  height: 200px;
  flex: none;
}
.name {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.freq {
  margin-top: 2px;
  font-size: 11px;
  color: var(--color-secondary);
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
.head h1 {
  margin: 0 0 8px;
  font-size: 34px;
}
.meta {
  font-size: 12px;
  color: var(--color-secondary);
}
.intro {
  margin: 10px 0 0;
  max-width: 620px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  white-space: pre-line;
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
