<script setup>
import { computed, ref } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({
  tags: { type: Array, default: () => [] },
  category: { type: String, default: "0" },
  playlists: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  error: { type: String, default: "" },
  hasNext: { type: Boolean, default: false },
});
const emit = defineEmits(["select", "more", "playlist"]);

const group = ref("");

const groups = computed(() => [{ id: "", name: "推荐" }, ...props.tags]);
const current = computed(() => props.tags.find((t) => t.id === group.value));
const children = computed(() => current.value?.children || []);

function pickGroup(id) {
  group.value = id;
  if (id) {
    const g = props.tags.find((t) => t.id === id);
    if (g?.children?.length) emit("select", g.children[0].id);
  } else {
    emit("select", "0");
  }
}
function pickChild(id) {
  emit("select", id);
}
function img(src, size = 240) {
  if (!src) return "";
  return src.replace("{size}", String(size)).replace(/^http:/, "https:");
}
function plays(n) {
  if (!n) return "";
  if (n >= 100000000) return (n / 100000000).toFixed(1) + " 亿播放";
  if (n >= 10000) return (n / 10000).toFixed(1) + " 万播放";
  return n + " 播放";
}
</script>

<template>
  <section class="plaza">
    <header class="head">
      <h1>歌单广场</h1>
      <span class="sub">按场景、心情、风格挑一张歌单</span>
    </header>

    <div class="chips">
      <button
        v-for="g in groups"
        :key="g.id || 'all'"
        class="chip"
        :class="{ active: group === g.id }"
        @click="pickGroup(g.id)"
      >
        {{ g.name }}
      </button>
    </div>
    <div v-if="children.length" class="chips sub-chips">
      <button
        v-for="c in children"
        :key="c.id"
        class="chip small"
        :class="{ active: category === c.id }"
        @click="pickChild(c.id)"
      >
        {{ c.name }}
      </button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>

    <div class="grid">
      <button
        v-for="p in playlists"
        :key="p.id"
        class="card"
        @click="emit('playlist', p)"
      >
        <div class="art">
          <img v-if="p.cover" :src="img(p.cover)" alt="" loading="lazy" />
          <Icon v-else name="music" />
        </div>
        <div class="name">{{ p.name }}</div>
        <div class="meta">{{ p.creator }}<template v-if="plays(p.plays)"> · {{ plays(p.plays) }}</template></div>
      </button>
    </div>

    <div v-if="loading" class="hint">正在载入…</div>
    <div v-else-if="!playlists.length" class="hint">这个分类暂时没有歌单</div>
    <div v-else class="more">
      <button v-if="hasNext" class="more-btn" @click="emit('more')">加载更多</button>
    </div>
  </section>
</template>

<style scoped>
.plaza {
  padding: 24px 0 40px;
}
.head {
  display: flex;
  align-items: baseline;
  gap: 14px;
}
h1 {
  margin: 0;
  font-size: 26px;
  letter-spacing: -0.02em;
}
.sub {
  font-size: 12px;
  color: var(--color-secondary);
}
.chips {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-top: 18px;
}
.chips.sub-chips {
  margin-top: 8px;
}
.chip {
  height: 28px;
  padding: 0 13px;
  border-radius: 14px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.chip:hover {
  color: var(--color-text);
}
.chip.active {
  color: var(--color-primary);
  background: var(--color-primary-bg-for-transparent);
}
.chip.small {
  height: 24px;
  font-size: 11px;
  font-weight: 500;
}
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 22px;
  margin-top: 24px;
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
.name {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.meta {
  margin-top: 3px;
  font-size: 11px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.error {
  padding: 16px 0;
  font-size: 13px;
  color: #e0533f;
}
.hint {
  padding: 40px 0;
  text-align: center;
  font-size: 13px;
  color: var(--color-secondary);
}
.more {
  padding: 28px 0 0;
  text-align: center;
}
.more-btn {
  height: 34px;
  padding: 0 22px;
  border-radius: 17px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-secondary);
  background: var(--color-secondary-bg);
}
.more-btn:hover {
  color: var(--color-text);
}
</style>
