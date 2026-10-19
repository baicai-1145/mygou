<script setup>
import Icon from "./Icon.vue";
import TrackTable from "./TrackTable.vue";

const props = defineProps({
  keyword: { type: String, default: "" },
  type: { type: String, default: "song" },
  results: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["play", "artist", "album", "playlist", "mv", "change-type"]);

const tabs = [
  { key: "song", label: "单曲" },
  { key: "playlist", label: "歌单" },
  { key: "artist", label: "歌手" },
  { key: "album", label: "专辑" },
  { key: "mv", label: "MV" },
];
const units = { song: "首单曲", playlist: "个歌单", artist: "位歌手", album: "张专辑", mv: "个 MV" };

function img(src, size = 240) {
  if (!src) return "";
  return src.replace("{size}", String(size)).replace(/^http:/, "https:");
}
function count() {
  if (props.loading) return "搜索中…";
  return `${props.results.length} ${units[props.type] || ""}`;
}
</script>

<template>
  <div class="search-view">
    <div class="head">
      <h1>搜索 “{{ keyword }}”</h1>
      <span class="count">{{ count() }}</span>
    </div>
    <div class="tabs">
      <button
        v-for="t in tabs"
        :key="t.key"
        :class="{ active: type === t.key }"
        @click="emit('change-type', t.key)"
      >
        {{ t.label }}
      </button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <TrackTable
      v-if="type === 'song'"
      :tracks="results"
      :current-hash="currentHash"
      :playing="playing"
      @play="emit('play', $event)"
      @artist="(t) => emit('artist', t)"
      @album="(t) => emit('album', t)"
    />

    <div v-else-if="type === 'playlist'" class="grid">
      <button v-for="p in results" :key="p.id" class="card" @click="emit('playlist', p)">
        <div class="art">
          <img v-if="p.cover" :src="img(p.cover)" alt="" loading="lazy" />
          <Icon v-else name="music" />
        </div>
        <div class="name">{{ p.name }}</div>
        <div class="sub">{{ p.creator }} · {{ p.count }} 首</div>
      </button>
    </div>

    <div v-else-if="type === 'artist'" class="grid artists">
      <button
        v-for="a in results"
        :key="a.id"
        class="card artist"
        @click="emit('artist', { singerID: a.id, singer: a.name })"
      >
        <div class="avatar">
          <img v-if="a.avatar" :src="img(a.avatar, 240)" alt="" loading="lazy" />
          <Icon v-else name="music" />
        </div>
        <div class="name">{{ a.name }}</div>
        <div class="sub">{{ a.songs }} 首歌曲 · {{ a.albums }} 张专辑</div>
      </button>
    </div>

    <div v-else-if="type === 'mv'" class="grid mvs">
      <button
        v-for="m in results"
        :key="m.id"
        class="card"
        @click="emit('mv', m)"
      >
        <div class="art mv-art">
          <img v-if="m.cover" :src="img(m.cover, 480)" alt="" loading="lazy" />
          <Icon v-else name="music" />
          <span class="badge">MV</span>
        </div>
        <div class="name">{{ m.name }}</div>
        <div class="sub">{{ m.singer }} · {{ Math.floor((m.duration || 0) / 60) }}:{{ String((m.duration || 0) % 60).padStart(2, "0") }}</div>
      </button>
    </div>

    <div v-else class="grid">
      <button
        v-for="al in results"
        :key="al.id"
        class="card"
        @click="
          emit('album', {
            albumID: al.id,
            album: al.name,
            singer: al.artist,
            cover: al.cover,
          })
        "
      >
        <div class="art">
          <img v-if="al.cover" :src="img(al.cover)" alt="" loading="lazy" />
          <Icon v-else name="music" />
        </div>
        <div class="name">{{ al.name }}</div>
        <div class="sub">{{ al.artist }} · {{ al.publish?.slice(0, 4) || "" }}</div>
      </button>
    </div>

    <div v-if="!loading && !results.length && !error" class="empty">
      没有找到{{ units[type] || "结果" }}
    </div>
  </div>
</template>

<style scoped>
.search-view {
  padding: 24px 0 40px;
}
.head {
  display: flex;
  align-items: baseline;
  gap: 16px;
}
h1 {
  margin: 0;
  font-size: 26px;
  letter-spacing: -0.02em;
}
.count {
  font-size: 12px;
  color: var(--color-secondary);
}
.tabs {
  display: flex;
  gap: 6px;
  margin: 18px 0 22px;
}
.tabs button {
  height: 30px;
  padding: 0 14px;
  border-radius: 15px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary);
}
.tabs button:hover {
  color: var(--color-text);
}
.tabs button.active {
  color: var(--color-primary);
  background: var(--color-primary-bg-for-transparent);
}
.error {
  color: #e0533f;
  font-size: 13px;
  padding: 20px 0;
}
.empty {
  padding: 60px 0;
  text-align: center;
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
.art img,
.avatar img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease;
}
.card:hover .art img,
.card:hover .avatar img {
  transform: scale(1.05);
}
.mv-art {
  position: relative;
  aspect-ratio: 16 / 9;
}
.mv-art .badge {
  position: absolute;
  left: 8px;
  bottom: 8px;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.04em;
  color: #fff;
  background: rgba(0, 0, 0, 0.55);
}
.avatar {
  width: 100%;
  aspect-ratio: 1;
  border-radius: 50%;
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
.name {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.sub {
  margin-top: 2px;
  font-size: 11px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
