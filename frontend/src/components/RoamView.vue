<script setup>
import Icon from "./Icon.vue";
import TrackTable from "./TrackTable.vue";

const props = defineProps({
  songs: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  currentHash: { type: String, default: "" },
  playing: { type: Boolean, default: false },
});
const emit = defineEmits(["play", "refresh", "artist", "album"]);
</script>

<template>
  <section class="roam">
    <header class="head">
      <div>
        <div class="kind">私人漫游</div>
        <h1>按你的口味，一直放下去</h1>
        <p>根据听过的歌和收藏随机挑选，每次换一批。</p>
      </div>
      <div class="actions">
        <button class="ghost" @click="emit('refresh')">↻ 换一批</button>
        <button class="playall" @click="emit('play', 0)">
          <Icon name="play" />
          播放全部
        </button>
      </div>
    </header>

    <div v-if="error" class="error">{{ error }}</div>
    <TrackTable
      :tracks="songs"
      :current-hash="currentHash"
      :playing="playing"
      @play="(i) => emit('play', i)"
      @artist="(t) => emit('artist', t)"
      @album="(t) => emit('album', t)"
    />
    <div v-if="loading" class="hint">正在为你挑选…</div>
  </section>
</template>

<style scoped>
.roam {
  padding: 26px 0 40px;
}
.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
  margin-bottom: 26px;
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
  margin: 0 0 6px;
  font-size: 30px;
  letter-spacing: -0.02em;
}
p {
  margin: 0;
  font-size: 12px;
  color: var(--color-secondary);
}
.actions {
  display: flex;
  gap: 8px;
  flex: none;
}
.ghost,
.playall {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 34px;
  padding: 0 16px;
  border-radius: 17px;
  font-size: 12px;
  font-weight: 600;
}
.ghost {
  color: var(--color-text);
  background: var(--color-secondary-bg);
}
.ghost:hover {
  background: var(--color-secondary-bg-for-transparent);
}
.playall {
  color: #fff;
  background: var(--color-primary);
}
.ghost .icon,
.playall .icon {
  width: 13px;
  height: 13px;
}
.error {
  padding: 12px 0;
  font-size: 13px;
  color: #e0533f;
}
.hint {
  padding: 26px 0;
  font-size: 13px;
  color: var(--color-secondary);
}
</style>
