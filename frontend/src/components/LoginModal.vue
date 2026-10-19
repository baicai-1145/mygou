<script setup>
import { onMounted, onUnmounted, ref } from "vue";
import QRCode from "qrcode";
import Icon from "./Icon.vue";

const emit = defineEmits(["close", "logged-in"]);

const call = (method, ...args) => window.mygo.call(method, ...args);

const canvas = ref(null);
const status = ref(1);
const message = ref("正在获取二维码…");
const failed = ref(false);
let timer = null;

const text = {
  0: "二维码已过期，请重新打开",
  1: "使用酷狗概念版 App 扫码",
  2: "已扫码，请在手机上确认",
  4: "登录成功",
};

async function start() {
  try {
    const qr = await call("App.LoginQRCode");
    if (!qr?.url) throw new Error("没有拿到二维码");
    await QRCode.toCanvas(canvas.value, qr.url, {
      width: 220,
      margin: 1,
      color: { dark: "#111111", light: "#ffffff" },
      errorCorrectionLevel: "M",
    });
    status.value = 1;
    message.value = text[1];
    poll(qr.key);
  } catch (e) {
    failed.value = true;
    message.value = e?.message || String(e);
  }
}

function poll(key) {
  clearInterval(timer);
  timer = setInterval(async () => {
    try {
      const r = await call("App.LoginQRCheck", key);
      status.value = r.status;
      message.value = text[r.status] || `状态 ${r.status}`;
      if (r.status === 4) {
        clearInterval(timer);
        setTimeout(() => emit("logged-in"), 600);
      }
      if (r.status === 0) clearInterval(timer);
    } catch (e) {
      message.value = e?.message || String(e);
    }
  }, 2000);
}

onMounted(start);
onUnmounted(() => clearInterval(timer));
</script>

<template>
  <div class="overlay" @click.self="emit('close')">
    <div class="modal">
      <button class="close" title="关闭" @click="emit('close')">
        <Icon name="next" />
      </button>
      <div class="title">扫码登录</div>
      <div class="qr">
        <canvas ref="canvas" width="220" height="220"></canvas>
      </div>
      <div class="status" :class="{ ok: status === 4, failed }">{{ message }}</div>
      <div class="hint">在手机酷狗概念版里扫一扫，确认后即可解锁音质和个人歌单。</div>
    </div>
  </div>
</template>

<style scoped>
.overlay {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: grid;
  place-items: center;
  background: rgba(0, 0, 0, 0.42);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}
.modal {
  position: relative;
  width: 320px;
  padding: 26px 24px 22px;
  border-radius: 18px;
  background: var(--color-body-bg);
  border: 1px solid var(--color-secondary-bg-for-transparent);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.45);
  text-align: center;
}
.close {
  position: absolute;
  top: 12px;
  right: 12px;
  width: 28px;
  height: 28px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-secondary);
  transform: rotate(45deg);
}
.close:hover {
  background: var(--color-secondary-bg-for-transparent);
  color: var(--color-text);
}
.title {
  font-size: 17px;
  font-weight: 700;
  margin-bottom: 18px;
}
.qr {
  width: 240px;
  height: 240px;
  margin: 0 auto 14px;
  padding: 10px;
  border-radius: 12px;
  background: #fff;
  display: grid;
  place-items: center;
}
.qr canvas {
  width: 220px;
  height: 220px;
  display: block;
}
.status {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  min-height: 20px;
}
.status.ok {
  color: #2eb872;
}
.status.failed {
  color: #ff6b6b;
}
.hint {
  margin-top: 10px;
  font-size: 12px;
  line-height: 1.55;
  color: var(--color-secondary);
}
</style>
