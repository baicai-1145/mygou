<script setup>
import { computed } from "vue";

// Feather-style paths, drawn in the text color.
const paths = {
  search: "M11 3a8 8 0 1 0 5.3 14L21 21M11 3a8 8 0 0 1 8 8",
  sun: "M12 4V2m0 20v-2m8-8h2M2 12h2m13.7-5.7 1.4-1.4M4.9 19.1l1.4-1.4m11.4 0 1.4 1.4M4.9 4.9l1.4 1.4M12 8a4 4 0 1 0 0 8 4 4 0 0 0 0-8Z",
  moon: "M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z",
  play: "M6 4.5v15l13-7.5-13-7.5Z",
  pause: "M7 4.5h3.5v15H7zm6.5 0H17v15h-3.5z",
  prev: "M19 5v14L8 12l11-7ZM5 4v16",
  next: "M5 5v14l11-7L5 5Zm14-1v16",
  volume: "M4 9v6h4l5 4V5L8 9H4Zm12.5-2a7 7 0 0 1 0 10M19 4.5a11 11 0 0 1 0 15",
  mute: "M4 9v6h4l5 4V5L8 9H4Zm12 1 5 5m0-5-5 5",
  heart: "M12 20.5 4.6 13a4.6 4.6 0 0 1 6.5-6.5l.9.9.9-.9A4.6 4.6 0 0 1 19.4 13L12 20.5Z",
  plus: "M12 5v14M5 12h14",
  folderPlus: "M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2zM12 11v6M9 14h6",
  repeat: "M17 1l4 4-4 4M3 11V9a4 4 0 0 1 4-4h14M7 23l-4-4 4-4M21 13v2a4 4 0 0 1-4 4H3",
  repeatOne: "M17 1l4 4-4 4M3 11V9a4 4 0 0 1 4-4h14M7 23l-4-4 4-4M21 13v2a4 4 0 0 1-4 4H3M11 10h1v4",
  shuffle: "M16 3h5v5M4 20L21 3M21 16v5h-5M15 15l6 6M4 4l5 5",
  fullscreen: "M8 3H5a2 2 0 0 0-2 2v3m18 0V5a2 2 0 0 0-2-2h-3m0 18h3a2 2 0 0 0 2-2v-3M3 16v3a2 2 0 0 0 2 2h3",
  fullscreenExit: "M8 3v3a2 2 0 0 1-2 2H3m18 0h-3a2 2 0 0 1-2-2V3m0 18v-3a2 2 0 0 1 2-2h3M3 16h3a2 2 0 0 1 2 2v3",
  lyrics: "M4 6h16M4 12h10M4 18h13M19 11v6a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z",
  music: "M9 18V5l11-2v13M9 18a3 3 0 1 1-6 0 3 3 0 0 1 6 0Zm11-2a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z",
};

const props = defineProps({ name: { type: String, required: true } });
const d = computed(() => paths[props.name] || "");
</script>

<template>
  <svg class="icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"
    stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
    <path :d="d" />
  </svg>
</template>

<style scoped>
.icon {
  width: 18px;
  height: 18px;
}
</style>
