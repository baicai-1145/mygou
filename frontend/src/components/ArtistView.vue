<script setup>
import Icon from "./Icon.vue";
import TrackTable from "./TrackTable.vue";

const props = defineProps({
  artist: { type: Object, default: null },
  songs: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  loadingMore: { type: Boolean, default: false },
  hasMore: { type: Boolean, default: false },
  error: { type: String, default: "" },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["play", "album", "back", "more"]);

function avatar(size = 240) {
  const a = props.artist?.avatar || "";
  return a ? a.replace("{size}", String(size)).replace(/^http:/, "https:") : "";
}
function n(v) {
  if (!v) return "0";
  return v >= 10000 ? (v / 10000).toFixed(1) + " 万" : String(v);
}
</script>

<template>
  <section class="artist">
    <button class="back" @click="emit('back')">← 返回</button>
    <div v-if="loading" class="hint">正在载入…</div>
    <div v-else-if="error" class="hint error">{{ error }}</div>
    <template v-else-if="artist">
      <header class="head">
        <div class="avatar">
          <img v-if="avatar()" :src="avatar()" alt="" />
          <Icon v-else name="music" />
        </div>
        <div class="info">
          <div class="kind">歌手</div>
          <h1>{{ artist.name }}</h1>
          <div class="meta">
            <span>{{ n(artist.songs) }} 首歌曲</span>
            <span>·</span>
            <span>{{ n(artist.albums) }} 张专辑</span>
            <span>·</span>
            <span>{{ n(artist.fans) }} 粉丝</span>
          </div>
          <p v-if="artist.intro" class="intro">{{ artist.intro }}</p>
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
        @album="(t) => emit('album', t)"
        @artist="() => {}"
      />
      <div v-if="hasMore" class="more-box">
        <button class="more-btn" :disabled="loadingMore" @click="emit('more')">
          {{ loadingMore ? "正在载入更多歌曲…" : `加载更多歌曲（已显示 ${songs.length} / 共 ${artist.songs || songs.length} 首）` }}
        </button>
      </div>
    </template>
  </section>
</template>

<style scoped>
.artist {
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
  gap: 28px;
  align-items: flex-end;
  margin: 8px 0 30px;
}
.avatar {
  width: 168px;
  height: 168px;
  flex: none;
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
.avatar img {
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
  font-size: 36px;
  letter-spacing: -0.02em;
}
.meta {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: var(--color-secondary);
}
.intro {
  margin: 12px 0 0;
  max-width: 620px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
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
.more-box {
  margin-top: 28px;
  text-align: center;
}
.more-btn {
  padding: 10px 26px;
  border-radius: 20px;
  background: var(--color-secondary-bg);
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition: all 0.15s ease;
}
.more-btn:hover {
  background: var(--color-secondary-bg-for-transparent);
  color: var(--color-primary);
}
.more-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
@media (max-width: 900px) {
  .head {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
