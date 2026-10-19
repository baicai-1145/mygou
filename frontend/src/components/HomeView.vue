<script setup>
import { computed } from "vue";
import Icon from "./Icon.vue";

const props = defineProps({
  recent: { type: Array, default: () => [] },
  daily: { type: Array, default: () => [] },
});
const emit = defineEmits(["search", "play", "playDaily", "roam"]);

const hot = [
  "海阔天空",
  "周杰伦",
  "夜航",
  "起风了",
  "陈奕迅",
  "Lo-Fi",
  "City Pop",
  "林俊杰",
];

const greeting = computed(() => {
  const h = new Date().getHours();
  if (h < 6) return "夜深了";
  if (h < 12) return "早上好";
  if (h < 14) return "中午好";
  if (h < 18) return "下午好";
  return "晚上好";
});

function cover(t) {
  const c = t.cover || "";
  return c ? c.replace("{size}", "100").replace(/^http:/, "https:") : "";
}

function time(sec) {
  sec = Math.max(0, Math.floor(sec || 0));
  return `${Math.floor(sec / 60)}:${String(sec % 60).padStart(2, "0")}`;
}
</script>

<template>
  <div class="home">
    <div class="top">
    <section class="intro">
      <div class="eyebrow">酷狗概念版 · 第三方客户端</div>
      <h1>{{ greeting }}</h1>
      <p class="sub">
        搜索你想听的歌，按回车开始。播放器在后台运行，关掉窗口也不会停。
      </p>
      <div class="chips">
        <button v-for="k in hot" :key="k" class="chip" @click="emit('search', k)">
          {{ k }}
        </button>
      </div>
      <div class="facts">
        <span>Go 解码播放</span>
        <i></i>
        <span>关窗不停止</span>
        <i></i>
        <span>扫码登录解锁音质</span>
      </div>
    </section>

    <aside class="recent">
      <div class="recent-head">
        <span>最近播放</span>
        <span v-if="recent.length" class="count">{{ recent.length }}</span>
      </div>
      <div v-if="!recent.length" class="empty">
        <Icon name="music" />
        <p>播放过的歌会出现在这里</p>
      </div>
      <button
        v-for="(t, i) in recent.slice(0, 9)"
        :key="t.hash + i"
        class="row"
        @click="emit('play', i)"
      >
        <img v-if="cover(t)" :src="cover(t)" alt="" loading="lazy" />
        <div v-else class="ph"><Icon name="music" /></div>
        <div class="meta">
          <div class="name">{{ t.name }}</div>
          <div class="singer">{{ t.singer }}</div>
        </div>
        <span class="time">{{ time(t.duration) }}</span>
      </button>
    </aside>
    </div>

    <section v-if="daily.length" class="daily">
      <div class="daily-head">
        <div>
          <h2>每日推荐</h2>
          <p>按你的口味，每天换一批</p>
        </div>
        <div class="daily-actions">
          <button class="roam" @click="emit('roam')">私人漫游</button>
          <button class="playall" @click="emit('playDaily', 0)">播放全部</button>
        </div>
      </div>
      <div class="daily-grid">
        <button
          v-for="(t, i) in daily.slice(0, 10)"
          :key="t.hash + i"
          class="dcard"
          @click="emit('playDaily', i)"
        >
          <div class="dart">
            <img v-if="cover(t)" :src="cover(t).replace('/100/', '/300/')" alt="" loading="lazy" />
            <Icon v-else name="music" />
          </div>
          <div class="dname">{{ t.name }}</div>
          <div class="dsinger">{{ t.singer }}</div>
        </button>
      </div>
    </section>
  </div>
</template>

<style scoped>
.home {
  padding: 8px 0 40px;
}
.top {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 400px;
  gap: 56px;
  align-items: start;
}
.intro {
  padding: 44px 0 24px;
  max-width: 560px;
}
.eyebrow {
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-primary);
  margin-bottom: 12px;
}
h1 {
  font-size: 54px;
  line-height: 1.05;
  letter-spacing: -0.03em;
  margin: 0 0 12px;
  font-weight: 800;
}
.sub {
  margin: 0 0 28px;
  color: var(--color-secondary);
  font-size: 15px;
  line-height: 1.6;
}
.chips {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}
.chip {
  height: 34px;
  padding: 0 16px;
  border-radius: 17px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  background: var(--color-secondary-bg-for-transparent);
}
.chip:hover {
  background: var(--color-primary-bg-for-transparent);
  color: var(--color-primary);
}
.facts {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 30px;
  font-size: 12px;
  color: var(--color-secondary);
}
.facts i {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: currentColor;
  opacity: 0.6;
}
.recent {
  margin-top: 44px;
  padding: 20px;
  border-radius: 16px;
  background: var(--color-secondary-bg-for-transparent);
}
.recent-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
  font-weight: 700;
  margin-bottom: 12px;
}
.count {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-secondary);
}
.empty {
  padding: 34px 0;
  text-align: center;
  color: var(--color-secondary);
}
.empty .icon {
  width: 28px;
  height: 28px;
  opacity: 0.5;
}
.empty p {
  margin: 10px 0 0;
  font-size: 12px;
}
.row {
  width: 100%;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 10px;
  text-align: left;
  color: inherit;
}
.row:hover {
  background: var(--color-secondary-bg);
}
.row img,
.ph {
  width: 40px;
  height: 40px;
  border-radius: 7px;
  object-fit: cover;
  flex: none;
  background: var(--color-body-bg);
  display: grid;
  place-items: center;
  color: var(--color-secondary);
}
.meta {
  min-width: 0;
  flex: 1;
}
.name {
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.singer {
  margin-top: 2px;
  font-size: 11px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.time {
  font-size: 11px;
  color: var(--color-secondary);
}
@media (max-width: 1100px) {
  .top {
    grid-template-columns: 1fr;
    gap: 0;
  }
  .recent {
    margin-top: 0;
  }
}
.daily {
  margin-top: 48px;
}
.daily-head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  margin-bottom: 16px;
}
.daily-head h2 {
  margin: 0;
  font-size: 22px;
}
.daily-head p {
  margin: 4px 0 0;
  font-size: 12px;
  color: var(--color-secondary);
}
.daily-actions {
  display: flex;
  gap: 8px;
  flex: none;
}
.roam {
  padding: 8px 16px;
  border-radius: 18px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-primary);
  background: var(--color-primary-bg-for-transparent);
}
.roam:hover {
  color: #fff;
  background: var(--color-primary);
}
.playall {
  padding: 8px 18px;
  border-radius: 18px;
  background: var(--color-secondary-bg-for-transparent);
  font-size: 13px;
  font-weight: 600;
}
.playall:hover {
  background: var(--color-primary);
  color: #fff;
}
.daily-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
  gap: 18px;
}
.dcard {
  text-align: left;
  color: inherit;
}
.dart {
  aspect-ratio: 1;
  border-radius: 10px;
  overflow: hidden;
  background: linear-gradient(
    135deg,
    var(--color-primary-bg-for-transparent),
    var(--color-secondary-bg)
  );
  display: grid;
  place-items: center;
  color: var(--color-secondary);
}
.dart img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform 0.3s ease;
}
.dcard:hover .dart img {
  transform: scale(1.05);
}
.dname {
  margin-top: 8px;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dsinger {
  margin-top: 2px;
  font-size: 11px;
  color: var(--color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
