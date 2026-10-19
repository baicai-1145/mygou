<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import NavBar from "./components/NavBar.vue";
import HomeView from "./components/HomeView.vue";
import SearchView from "./components/SearchView.vue";
import MyView from "./components/MyView.vue";
import PlaylistView from "./components/PlaylistView.vue";
import RankView from "./components/RankView.vue";
import PlazaView from "./components/PlazaView.vue";
import RoamView from "./components/RoamView.vue";
import CommentsModal from "./components/CommentsModal.vue";
import VideoModal from "./components/VideoModal.vue";
import QueueModal from "./components/QueueModal.vue";
import SettingsModal from "./components/SettingsModal.vue";
import ArtistView from "./components/ArtistView.vue";
import AlbumView from "./components/AlbumView.vue";
import NowPlaying from "./components/NowPlaying.vue";
import AddToPlaylistModal from "./components/AddToPlaylistModal.vue";
import PlayerBar from "./components/PlayerBar.vue";
import LoginModal from "./components/LoginModal.vue";

const call = (method, ...args) => {
  if (!window.mygo?.call) return Promise.reject(new Error("mygo bridge unavailable"));
  return window.mygo.call(method, ...args);
};

const theme = ref(localStorage.getItem("theme") || "dark");
watch(
  theme,
  (t) => {
    document.documentElement.dataset.theme = t;
    localStorage.setItem("theme", t);
  },
  { immediate: true },
);

watch(
  () => state.value?.track?.hash,
  () => {
    liked.value = false;
    refreshLiked();
  },
);

const account = ref({ loggedIn: false, name: "" });
const showLogin = ref(false);

async function refreshAccount() {
  try {
    account.value = await call("App.Account");
  } catch {
    account.value = { loggedIn: false, name: "" };
  }
}

async function onLoggedIn() {
  showLogin.value = false;
  await refreshAccount();
  await loadPlaylists();
  loadDaily();
  claimVip();
  if (keyword.value) search(keyword.value);
}

const view = ref(localStorage.getItem("mygou-view") || "home");
const showNow = ref(localStorage.getItem("mygou-shownow") === "1");

watch(showNow, (val) => {
  localStorage.setItem("mygou-shownow", val ? "1" : "0");
});

watch(view, (v) => {
  if (v && v !== "search" && v !== "playlist" && v !== "artist" && v !== "album") {
    localStorage.setItem("mygou-view", v);
  }
});
const playlists = ref([]);
const recent = ref([]);
const daily = ref([]);
const searchType = ref("song");
const playlistsLoading = ref(false);
const playlistsError = ref("");
const page = ref(null);
const pageLoading = ref(false);
const pageError = ref("");
const ranks = ref([]);
const ranksLoading = ref(false);
const ranksError = ref("");
const rankDetail = ref(null);
const rankSongs = ref([]);
const rankSongsLoading = ref(false);
const mode = ref("list");
const liked = ref(false);
const showVideo = ref(false);
const showQueue = ref(false);
const showSettings = ref(false);
const cacheInfo = ref({ bytes: 0, files: 0 });
const qualityPref = ref(localStorage.getItem("quality") || "320");
const mvQuality = ref(localStorage.getItem("mvQuality") || "fhd");
const queueState = ref(null);
const video = ref(null);
const showComments = ref(false);
const comments = ref([]);
const commentsTotal = ref(0);
const commentsPage = ref(1);
const commentsMore = ref(false);
const commentsLoading = ref(false);
const commentsError = ref("");
const roamSongs = ref([]);
const roamLoading = ref(false);
const roamError = ref("");
const plazaTags = ref([]);
const plazaCategory = ref("0");
const plazaPlaylists = ref([]);
const plazaPage = ref(1);
const plazaHasNext = ref(false);
const plazaLoading = ref(false);
const plazaError = ref("");
const artistPage = ref(null);
const artistLoading = ref(false);
const artistError = ref("");
let artistPageNum = 1;
const artistLoadingMore = ref(false);
const artistHasMore = computed(() => {
  const total = Number(artistPage.value?.artist?.songs) || 0;
  const current = artistPage.value?.songs?.length || 0;
  return (total > 0 && total > current) || (current > 0 && current % 300 === 0);
});
const albumPage = ref(null);
const albumLoading = ref(false);
const albumError = ref("");
const backView = ref("home");

async function loadDaily() {
  if (!account.value.loggedIn) return;
  try {
    const list = await call("App.Daily");
    daily.value = Array.isArray(list) ? list : [];
  } catch {
    daily.value = [];
  }
}

async function claimVip() {
  try {
    const c = await call("App.ClaimVIP");
    if (c && c.awardHours > 0) {
      toast("已自动领取 " + c.awardHours + " 小时 VIP（今日 " + c.done + "/" + c.total + "）");
    }
  } catch {
    // Not logged in, or the service is busy: the player works anyway.
  }
}

async function playDaily(index) {
  const t = daily.value[index];
  if (!t) return;
  try {
    await call("App.PlayQueue", daily.value, index);
    loadRecent();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function loadRecent() {
  try {
    const list = await call("App.Recent");
    recent.value = Array.isArray(list) ? list : [];
  } catch {
    recent.value = [];
  }
}

async function playRecent(index) {
  const t = recent.value[index];
  if (!t) return;
  try {
    await call("App.PlayQueue", recent.value, index);
    loadRecent();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function loadPlaylists() {
  if (!account.value.loggedIn) return;
  playlistsLoading.value = true;
  playlistsError.value = "";
  try {
    const lists = await call("App.Playlists");
    playlists.value = Array.isArray(lists) ? lists : [];
  } catch (e) {
    playlists.value = [];
    playlistsError.value = e?.message || String(e);
  } finally {
    playlistsLoading.value = false;
  }
}

async function loadComments(mixID, page = 1) {
  if (page === 1) {
    comments.value = [];
    commentsTotal.value = 0;
    commentsPage.value = 1;
    commentsMore.value = false;
  }
  commentsError.value = "";
  commentsLoading.value = true;
  try {
    const res = await call("App.Comments", mixID, page);
    const list = res?.comments || [];
    comments.value = page === 1 ? list : comments.value.concat(list);
    commentsTotal.value = res?.total || 0;
    commentsPage.value = page;
    commentsMore.value = comments.value.length < commentsTotal.value && list.length > 0;
  } catch (e) {
    commentsError.value = e?.message || String(e);
  } finally {
    commentsLoading.value = false;
  }
}

function moreComments() {
  const track = state.value?.track;
  if (!track || commentsLoading.value || !commentsMore.value) return;
  loadComments(track.mixSongID || "", commentsPage.value + 1);
}

async function openComments() {
  const track = state.value?.track;
  if (!track) return;
  showComments.value = true;
  await loadComments(track.mixSongID || "");
}

async function loadRoam(play = false) {
  roamLoading.value = true;
  roamError.value = "";
  try {
    const songs = (await call("App.PersonalFM", 20)) || [];
    roamSongs.value = songs;
    if (play && songs.length) {
      await call("App.PlayQueue", songs, 0);
      loadRecent();
    }
  } catch (e) {
    roamError.value = e?.message || String(e);
  } finally {
    roamLoading.value = false;
  }
}

function startRoam() {
  view.value = "roam";
  loadRoam(true);
}

async function playRoam(i) {
  if (!roamSongs.value[i]) return;
  try {
    await call("App.PlayQueue", roamSongs.value, i);
    loadRecent();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function loadPlaza(page = 1) {
  plazaLoading.value = true;
  plazaError.value = "";
  try {
    if (!plazaTags.value.length) {
      plazaTags.value = (await call("App.PlaylistTags")) || [];
    }
    const res = await call("App.PlaylistPlaza", plazaCategory.value, page);
    const list = res?.playlists || [];
    plazaPlaylists.value = page === 1 ? list : plazaPlaylists.value.concat(list);
    plazaHasNext.value = !!res?.hasNext;
    plazaPage.value = page;
  } catch (e) {
    plazaError.value = e?.message || String(e);
  } finally {
    plazaLoading.value = false;
  }
}

function selectPlazaCategory(id) {
  if (id === plazaCategory.value && plazaPlaylists.value.length) return;
  plazaCategory.value = id;
  loadPlaza(1);
}

function loadMorePlaza() {
  if (!plazaLoading.value && plazaHasNext.value) loadPlaza(plazaPage.value + 1);
}

function navigate(to) {
  if (to === "plaza") {
    view.value = "plaza";
    if (!plazaPlaylists.value.length) loadPlaza(1);
  } else if (to === "mine") {
    view.value = "mine";
    loadPlaylists();
  } else if (to === "rank") {
    view.value = "rank";
    rankDetail.value = null;
    loadRanks();
  } else {
    view.value = "home";
  }
}

function goBack() {
  view.value = backView.value;
}

async function openArtist(t) {
  let id = t?.singerID;
  if (!id && t?.singer) {
    try {
      id = await call("App.FindArtist", t.singer);
    } catch {
      id = "";
    }
  }
  if (!id) {
    toast("没有这位歌手的信息");
    return;
  }
  backView.value = view.value === "artist" ? backView.value : view.value;
  view.value = "artist";
  artistPage.value = { artist: null, songs: [] };
  artistLoading.value = true;
  artistError.value = "";
  artistPageNum = 1;
  try {
    const [artist, songs] = await Promise.all([
      call("App.Artist", id),
      call("App.ArtistSongs", id, 1),
    ]);
    artistPage.value = { artist, songs: songs || [] };
  } catch (e) {
    artistError.value = e?.message || String(e);
  } finally {
    artistLoading.value = false;
  }
}

async function playArtist(i) {
  const songs = artistPage.value?.songs || [];
  if (!songs[i]) return;
  try {
    await call("App.PlayQueue", songs, i);
    loadRecent();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function moreArtistSongs() {
  const id = artistPage.value?.artist?.id;
  if (!id || artistLoadingMore.value) return;
  artistLoadingMore.value = true;
  try {
    artistPageNum++;
    const more = await call("App.ArtistSongs", id, artistPageNum);
    if (more && more.length) {
      artistPage.value.songs.push(...more);
    }
  } catch (e) {
    toast(e?.message || "加载更多失败");
  } finally {
    artistLoadingMore.value = false;
  }
}

async function openAlbum(t) {
  const id = t?.albumID;
  if (!id) {
    toast("没有这张专辑的信息");
    return;
  }
  backView.value = view.value === "album" ? backView.value : view.value;
  view.value = "album";
  albumPage.value = {
    album: { name: t.album || "", artist: t.singer || "", cover: t.cover || "" },
    songs: [],
  };
  albumLoading.value = true;
  albumError.value = "";
  try {
    const songs = (await call("App.AlbumSongs", id, 1)) || [];
    const first = songs[0] || {};
    albumPage.value = {
      album: {
        name: albumPage.value.album.name || first.album || "未知专辑",
        artist: albumPage.value.album.artist || first.singer || "",
        cover: albumPage.value.album.cover || first.cover || "",
      },
      songs,
    };
  } catch (e) {
    albumError.value = e?.message || String(e);
  } finally {
    albumLoading.value = false;
  }
}

async function playAlbum(i) {
  const songs = albumPage.value?.songs || [];
  if (!songs[i]) return;
  try {
    await call("App.PlayQueue", songs, i);
    loadRecent();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function loadRanks() {
  if (ranks.value.length) return;
  ranksLoading.value = true;
  ranksError.value = "";
  try {
    ranks.value = (await call("App.Ranks")) || [];
  } catch (e) {
    ranksError.value = e?.message || String(e);
  } finally {
    ranksLoading.value = false;
  }
}

async function openRank(r) {
  view.value = "rank";
  rankDetail.value = r;
  rankSongs.value = [];
  rankSongsLoading.value = true;
  try {
    rankSongs.value = (await call("App.RankSongs", r.id, 1)) || [];
  } catch (e) {
    toast(e?.message || String(e));
  } finally {
    rankSongsLoading.value = false;
  }
}

async function playRank(i) {
  if (!rankSongs.value[i]) return;
  try {
    await call("App.PlayQueue", rankSongs.value, i);
    loadRecent();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function openPlaylist(p) {
  view.value = "playlist";
  page.value = { playlist: p, tracks: [] };
  pageLoading.value = true;
  pageError.value = "";
  try {
    const res = await call("App.PlaylistTracks", p.id);
    if (res?.playlist) {
      if (p.name && (p.name === "我喜欢" || p.name === "默认收藏" || res.playlist.name?.endsWith("喜欢的音乐"))) {
        res.playlist.name = p.name || "我喜欢";
      }
      if (!res.playlist.cover && p.cover) {
        res.playlist.cover = p.cover;
      }
    }
    page.value = res;
  } catch (e) {
    pageError.value = e?.message || String(e);
  } finally {
    pageLoading.value = false;
  }
}

async function playPage(index) {
  const tracks = page.value?.tracks || [];
  if (!tracks[index]) return;
  try {
    await call("App.PlayQueue", tracks, index);
  } catch (e) {
    pageError.value = e?.message || String(e);
  }
}

const keyword = ref("");
const results = ref([]);
const loading = ref(false);
const searched = ref(false);
const error = ref("");

async function search(kw, type = "song") {
  view.value = "search";
  keyword.value = kw;
  searchType.value = type;
  searched.value = true;
  loading.value = true;
  error.value = "";
  results.value = [];
  try {
    let list = [];
    if (type === "playlist") list = await call("App.SearchPlaylists", kw);
    else if (type === "artist") list = await call("App.SearchArtists", kw);
    else if (type === "album") list = await call("App.SearchAlbums", kw);
    else if (type === "mv") list = await call("App.SearchMVs", kw);
    else list = await call("App.Search", kw);
    results.value = Array.isArray(list) ? list : [];
  } catch (e) {
    error.value = e?.message || String(e);
  } finally {
    loading.value = false;
  }
}

function changeSearchType(type) {
  if (type === searchType.value) return;
  search(keyword.value, type);
}

function openPlaylistHit(p) {
  openPlaylist({ id: p.id, name: p.name, cover: p.cover, creator: p.creator, count: p.count });
}

// The player state from Go, polled twice a second; the position between
// polls is interpolated in the page, so the bar moves smoothly.
const state = ref(null);
const anchor = ref({ position: 0, at: performance.now() });
const now = ref(performance.now());
let pollTimer = null;
let frameTimer = null;
let updateTimers = null;
const unplayable = new Set();

async function poll() {
  try {
    const s = await call("App.State");
    state.value = s;
    if (s.err && /no playable url/i.test(s.err) && s.track?.hash && !unplayable.has(s.track.hash)) {
      unplayable.add(s.track.hash);
      toast(`《${s.track.name}》暂不可播放，已跳过`);
      call("App.Next").catch(() => {});
    }
    anchor.value = { position: s.position || 0, at: performance.now() };
  } catch {
    // The window may be closing; keep the last state.
  }
}

const position = computed(() => {
  const s = state.value;
  if (!s) return 0;
  if (!s.playing || s.loading) return anchor.value.position;
  return Math.min(s.duration || 0, anchor.value.position + (now.value - anchor.value.at) / 1000);
});

async function playAt(index) {
  const track = results.value[index];
  if (!track) return;
  try {
    await call("App.PlayQueue", results.value, index);
    loadRecent();
  } catch (e) {
    error.value = e?.message || String(e);
  }
}

async function toggle() {
  await call("App.Toggle");
}
async function prev() {
  await call("App.Prev");
}
async function next() {
  await call("App.Next");
}
async function seek(seconds) {
  anchor.value = { position: seconds, at: performance.now() };
  await call("App.Seek", seconds);
}
let volRaf = null;
let pendingVol = null;

async function volume(v) {
  if (state.value) {
    state.value.volume = v;
  }
  pendingVol = v;
  if (!volRaf) {
    volRaf = requestAnimationFrame(async () => {
      volRaf = null;
      if (pendingVol !== null) {
        const send = pendingVol;
        pendingVol = null;
        try {
          await call("App.SetVolume", send);
        } catch {}
      }
    });
  }
}

const toastText = ref("");
let toastTimer = null;
function toast(msg) {
  toastText.value = msg;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (toastText.value = ""), 2600);
}

const showCollectModal = ref(false);
const collectTrack = ref(null);
function openCollect(track) {
  collectTrack.value = track || state.value?.track || null;
  if (!collectTrack.value?.hash) {
    toast("当前没有可收藏的歌曲");
    return;
  }
  showCollectModal.value = true;
}
function onCollected(name) {
  toast(`已收藏到歌单「${name}」`);
}

async function refreshLiked() {
  const hash = state.value?.track?.hash;
  if (!hash) {
    liked.value = false;
    return;
  }
  try {
    liked.value = await call("App.FavoriteState", hash);
  } catch {
    liked.value = false;
  }
}

async function createPlaylist() {
  const name = window.prompt("新歌单名称", "我的歌单");
  if (!name || !name.trim()) return;
  try {
    playlists.value = await call("App.CreatePlaylist", name.trim());
    toast("歌单已创建");
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function toggleFavTrack(t) {
  try {
    const liked = await call("App.ToggleFavoriteTrack", t);
    toast(liked ? `已添加《${t.name}》到「我喜欢」` : `已取消喜欢《${t.name}》`);
    if (view.value === "mine") loadPlaylists();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function favorite() {
  try {
    const was = liked.value;
    liked.value = await call("App.ToggleFavorite");
    toast(was ? (liked.value ? "已在我喜欢中" : "已取消喜欢") : "已添加到「我喜欢」");
    if (view.value === "mine") loadPlaylists();
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function openSettings() {
  showSettings.value = true;
  try {
    cacheInfo.value = await call("App.CacheInfo");
  } catch {
    cacheInfo.value = { bytes: 0, files: 0 };
  }
}

async function setMvQuality(q) {
  mvQuality.value = q;
  localStorage.setItem("mvQuality", q);
  if (showVideo.value && video.value?.id) {
    await openMV({ id: video.value.id, name: video.value.name, singer: video.value.singer });
  }
}

async function setQuality(q) {
  qualityPref.value = q;
  localStorage.setItem("quality", q);
  window.dispatchEvent(new CustomEvent("mygou-quality", { detail: q }));
  try {
    const got = await call("App.SetQuality", q);
    if (got && got !== q) {
      toast("当前账号只能播放 " + (got === "flac" ? "无损" : got === "320" ? "高品质" : "标准") + " 音质");
    }
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function clearCache() {
  try {
    await call("App.ClearCache");
    cacheInfo.value = await call("App.CacheInfo");
    toast("缓存已清理");
  } catch (e) {
    toast(e?.message || String(e));
  }
}

const statusBarLyric = ref(true);
async function setStatusBarLyric(on) {
  statusBarLyric.value = on;
  try {
    await call("App.SetStatusBarLyric", on);
  } catch (e) {
    toast(e?.message || "切换失败");
  }
}

async function openQueue() {
  showQueue.value = true;
  try {
    queueState.value = await call("App.Queue");
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function playQueueIndex(i) {
  try {
    await call("App.PlayIndex", i);
    queueState.value = await call("App.Queue");
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function download() {
  try {
    const path = await call("App.DownloadCurrent");
    toast("已下载到 " + path);
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function openMV(m) {
  showVideo.value = true;
  video.value = { id: m.id, name: m.name, singer: m.singer, url: "", error: "" };
  try {
    const got = await call("App.PlayMV", m.id, mvQuality.value);
    video.value = {
      id: m.id,
      name: got?.name || m.name,
      singer: got?.singer || m.singer,
      url: got?.url || "",
      error: got?.url ? "" : "没有可播放的视频文件",
    };
  } catch (e) {
    video.value = { ...video.value, error: e?.message || String(e) };
  }
}

async function cycleMode() {
  const next = mode.value === "list" ? "one" : mode.value === "one" ? "shuffle" : "list";
  try {
    mode.value = await call("App.SetMode", next);
    toast(
      mode.value === "one" ? "单曲循环" : mode.value === "shuffle" ? "随机播放" : "列表循环",
    );
  } catch (e) {
    toast(e?.message || String(e));
  }
}

async function quality(q) {
  try {
    const got = await call("App.SetQuality", q);
    if (got !== q) {
      const label =
        got === "flac" || got === "high" ? "无损" : got === "320" ? "高品质" : got === "128" ? "标准" : got;
      toast("当前账号只能播放 " + label + " 音质");
    }
  } catch (e) {
    toast(e?.message || String(e));
  }
}

onMounted(async () => {
  window.addEventListener("mygou-fav", (e) => toggleFavTrack(e.detail));
  window.addEventListener("mygou-collect", (e) => openCollect(e.detail));
  try {
    statusBarLyric.value = await call("App.StatusBarLyric");
  } catch {}
  // Deep links and development: /?q=keyword&play=1 searches at once.
  const params = new URLSearchParams(location.search);
  if (params.get("login") === "1") showLogin.value = true;
  if (params.get("theme")) theme.value = params.get("theme");
  if (params.get("mine") === "1") {
    await refreshAccount();
    navigate("mine");
  }
  if (params.get("ranks") === "1") {
    view.value = "rank";
    await loadRanks();
  }
  if (params.get("settings") === "1") await openSettings();
  const mvID = params.get("mv");
  if (mvID) {
    await openMV({ id: mvID, name: params.get("mvname") || "", singer: "" });
  }
  const cmt = params.get("cmt");
  if (cmt) {
    showComments.value = true;
    await loadComments(cmt);
  }
  if (params.get("roam") === "1") {
    view.value = "roam";
    await loadRoam(params.get("play") === "1");
  }
  if (params.get("plaza") === "1") {
    plazaCategory.value = params.get("cat") || "0";
    view.value = "plaza";
    await loadPlaza(1);
  }
  const artistName = params.get("artistname");
  if (artistName) {
    try {
      const id = await call("App.FindArtist", artistName);
      await openArtist({ singerID: id, singer: artistName });
    } catch (e) {
      toast(e?.message || String(e));
    }
  }
  const artistID = params.get("artist");
  if (artistID) await openArtist({ singerID: artistID });
  const albumID = params.get("album");
  if (albumID) await openAlbum({ albumID: albumID });
  const rankID = params.get("rank");
  if (rankID) {
    await loadRanks();
    const r = ranks.value.find((x) => x.id === rankID);
    if (r) await openRank(r);
  }
  const plid = params.get("plid");
  if (plid) {
    await refreshAccount();
    await loadPlaylists();
    const p = playlists.value.find((x) => x.id === plid);
    await openPlaylist(p || { id: plid, name: "", cover: "", creator: "" });
  }
  if (params.get("now") === "1") showNow.value = true;
  const q = params.get("q");
  if (q) {
    const type = params.get("type") || "song";
    await search(q, type);
    try {
    mode.value = await call("App.Mode");
  } catch {}
  const vol = params.get("vol");
    if (vol) await call("App.SetVolume", Number(vol));
    if (params.get("play") === "1" && type === "song") playAt(0);
    if (params.get("download") === "1") setTimeout(download, 6000);
    if (params.get("queue") === "1") setTimeout(openQueue, 4000);
    if (params.get("fav") === "1") setTimeout(favorite, 6000);
  }
  await refreshAccount();
  await loadRecent();
  loadDaily();
  claimVip();
  if (!q && params.get("play") === "1") {
    if (recent.value?.length) playRecent(0);
  }
  const qParam = new URLSearchParams(location.search).get("quality");
  if (qParam) localStorage.setItem("quality", qParam);
  const saved = qParam || localStorage.getItem("quality");
  if (saved) call("App.SetQuality", saved).catch(() => {});
  poll();
  pollTimer = setInterval(poll, 500);
  frameTimer = setInterval(() => (now.value = performance.now()), 100);

  updateTimers = function () {
    clearInterval(pollTimer);
    clearInterval(frameTimer);
    document.documentElement.dataset.hidden = document.hidden ? "true" : "false";
    if (document.hidden) {
      pollTimer = setInterval(poll, 3000);
    } else {
      poll();
      pollTimer = setInterval(poll, 1000);
      frameTimer = setInterval(() => (now.value = performance.now()), 100);
    }
  };
  document.documentElement.dataset.hidden = document.hidden ? "true" : "false";
  document.addEventListener("visibilitychange", updateTimers);
  window.addEventListener("keydown", onGlobalKeydown);
});
onUnmounted(() => {
  clearInterval(pollTimer);
  clearInterval(frameTimer);
  if (updateTimers) document.removeEventListener("visibilitychange", updateTimers);
  window.removeEventListener("keydown", onGlobalKeydown);
});

async function onGlobalKeydown(e) {
  if (e.key === "Escape") {
    if (showNow.value) return;
    if (showSettings.value || showQueue.value || showVideo.value || showComments.value || showLogin.value) return;
    try {
      const isFS = await call("App.IsFullScreen");
      if (isFS) {
        e.preventDefault();
        await call("App.ExitFullScreen");
      }
    } catch {}
  }
}
</script>

<template>
  <NavBar
    :theme="theme"
    :logged-in="account.loggedIn"
    :avatar="account.avatar || ''"
    :view="view"
    @toggle-theme="theme = theme === 'dark' ? 'light' : 'dark'"
    @search="search"
    @login="showLogin = true"
    @navigate="navigate"
    @settings="openSettings"
  />
  <main class="content">
    <SearchView
      v-if="view === 'search'"
      :keyword="keyword"
      :type="searchType"
      :results="results"
      :loading="loading"
      :error="error"
      :current-hash="state?.track?.hash || ''"
      :playing="state?.playing || false"
      @play="playAt"
      @artist="openArtist"
      @album="openAlbum"
      @playlist="openPlaylistHit"
      @mv="openMV"
      @change-type="changeSearchType"
    />
    <PlaylistView
      v-else-if="view === 'playlist'"
      :page="page"
      :loading="pageLoading"
      :error="pageError"
      :current-hash="state?.track?.hash || ''"
      :playing="state?.playing || false"
      @play="playPage"
      @artist="openArtist"
      @album="openAlbum"
      @back="navigate('mine')"
    />
    <ArtistView
      v-else-if="view === 'artist'"
      :artist="artistPage?.artist || null"
      :songs="artistPage?.songs || []"
      :loading="artistLoading"
      :loading-more="artistLoadingMore"
      :has-more="artistHasMore"
      :error="artistError"
      :current-hash="state?.track?.hash || ''"
      :playing="state?.playing || false"
      @play="playArtist"
      @album="openAlbum"
      @more="moreArtistSongs"
      @back="goBack"
    />
    <AlbumView
      v-else-if="view === 'album'"
      :album="albumPage?.album || null"
      :songs="albumPage?.songs || []"
      :loading="albumLoading"
      :error="albumError"
      :current-hash="state?.track?.hash || ''"
      :playing="state?.playing || false"
      @play="playAlbum"
      @artist="openArtist"
      @back="goBack"
    />
    <MyView
      v-else-if="view === 'mine'"
      :playlists="playlists"
      :loading="playlistsLoading"
      :error="playlistsError"
      :logged-in="account.loggedIn"
      :account="account"
      @open="openPlaylist"
      @login="showLogin = true"
    />
    <RoamView
      v-else-if="view === 'roam'"
      :songs="roamSongs"
      :loading="roamLoading"
      :error="roamError"
      :current-hash="state?.track?.hash || ''"
      :playing="state?.playing || false"
      @play="playRoam"
      @refresh="loadRoam(false)"
      @artist="openArtist"
      @album="openAlbum"
    />
    <PlazaView
      v-else-if="view === 'plaza'"
      :tags="plazaTags"
      :category="plazaCategory"
      :playlists="plazaPlaylists"
      :loading="plazaLoading"
      :error="plazaError"
      :has-next="plazaHasNext"
      @select="selectPlazaCategory"
      @more="loadMorePlaza"
      @playlist="openPlaylistHit"
    />
    <RankView
      v-else-if="view === 'rank'"
      :ranks="ranks"
      :loading="ranksLoading"
      :error="ranksError"
      :detail="rankDetail"
      :songs="rankSongs"
      :songs-loading="rankSongsLoading"
      :current-hash="state?.track?.hash || ''"
      :playing="state?.playing || false"
      @open="openRank"
      @play="playRank"
      @artist="openArtist"
      @album="openAlbum"
      @back="rankDetail = null"
    />
    <HomeView
      v-else
      :recent="recent"
      :daily="daily"
      @search="search"
      @play="playRecent"
      @play-daily="playDaily"
      @roam="startRoam"
    />
  </main>
  <transition name="slide-up">
    <NowPlaying
      v-if="showNow"
      :state="state"
      :position="position"
      :mode="mode"
      :liked="liked"
      @close="showNow = false"
      @toggle="toggle"
      @prev="prev"
      @next="next"
      @seek="seek"
      @volume="volume"
      @mode="cycleMode"
      @favorite="favorite"
      @collect="openCollect"
    />
  </transition>
  <SettingsModal
    v-if="showSettings"
    :quality="qualityPref"
    :mv-quality="mvQuality"
    :cache="cacheInfo"
    :status-bar-lyric="statusBarLyric"
    @set-quality="setQuality"
    @set-mv-quality="setMvQuality"
    @set-status-bar-lyric="setStatusBarLyric"
    @clear-cache="clearCache"
    @close="showSettings = false"
  />
  <QueueModal
    v-if="showQueue"
    :queue="queueState"
    @play="playQueueIndex"
    @close="showQueue = false"
  />
  <VideoModal
    v-if="showVideo"
    :video="video"
    :quality="mvQuality"
    @quality="setMvQuality"
    @close="showVideo = false"
  />
  <CommentsModal
    v-if="showComments"
    :track="state?.track || null"
    :comments="comments"
    :total="commentsTotal"
    :loading="commentsLoading"
    :error="commentsError"
    :has-more="commentsMore"
    @more="moreComments"
    @close="showComments = false"
  />
  <LoginModal
    v-if="showLogin"
    @close="showLogin = false"
    @logged-in="onLoggedIn"
  />
  <PlayerBar
    :state="state"
    :position="position"
    :mode="mode"
    :liked="liked"
    @lyrics="showNow = true"
    @mode="cycleMode"
    @toggle="toggle"
    @prev="prev"
    @next="next"
    @seek="seek"
    @volume="volume"
    @favorite="favorite"
    @collect="openCollect"
    @quality="quality"
    @comments="openComments"
    @download="download"
    @queue="openQueue"
  />
  <AddToPlaylistModal
    v-if="showCollectModal"
    :track="collectTrack"
    @close="showCollectModal = false"
    @added="onCollected"
  />
  <transition name="toast">
    <div v-if="toastText" class="toast">{{ toastText }}</div>
  </transition>
</template>

<style scoped>
.toast {
  position: fixed;
  left: 50%;
  bottom: calc(var(--player-height) + 18px);
  transform: translateX(-50%);
  z-index: 60;
  padding: 9px 18px;
  border-radius: 20px;
  background: rgba(20, 20, 22, 0.9);
  color: #fff;
  font-size: 13px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.35);
}
.toast-enter-active,
.toast-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translate(-50%, 6px);
}
.content {
  position: fixed;
  inset: 64px 0 var(--player-height) 0;
  overflow-y: auto;
  padding: 0 40px;
}
.content > * {
  max-width: 1180px;
  margin: 0 auto;
}
</style>

<style>
button,
a,
select,
input[type="range"],
[role="button"],
.avatar,
.round,
.card,
.line,
.track-row,
.playlist-card,
.links span,
.chip,
.roam,
.playall,
.dcard,
.progress,
.volume,
.now img,
.play-btn,
.fav,
.link,
.side,
.main,
.back,
.ghost,
.new,
.primary,
.more-btn,
.tabs button {
  cursor: pointer !important;
}

button:disabled,
.main:disabled {
  cursor: not-allowed !important;
}

button,
.avatar,
.round,
.main,
.side {
  transition: transform 0.12s ease, filter 0.12s ease;
}
button:active,
.avatar:active,
.round:active,
.main:active,
.side:active {
  transform: scale(0.94);
}
</style>
