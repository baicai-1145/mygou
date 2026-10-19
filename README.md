# mygou

A small macOS music player for 酷狗概念版 (KuGou concept edition): a Go
service that talks to the KuGou API and decodes audio, a Vue front end in a
WKWebView, and a native menu bar mini player.

## What works

- **Search** with tabs for songs, playlists, singers and albums
  (`/v2/search/song`, `/v1/search/special`, `/v1/search/author`,
  `/v1/search/album`). Public playlists open like the account's own.
- **Playback** decoded in Go (MP3, Ogg Vorbis and FLAC), so closing the
  window does not stop the music: the app keeps running in the menu bar.
- **Quality tiers**: 标准 (128), 高品质 (320) and 无损 (FLAC). Each tier
  has its own hash in the privilege answer (`hash_flac`, ...), and the URL
  endpoint only serves the file for that hash: asking with the plain hash
  returns the wrong file. VIP songs play a 60 s trial when the account
  cannot unlock them.
- **Daily recommendations** from the account, on the home view.
- **Charts**: the store's 50+ rankings, with covers, update cadence and
  the full song table (`/ocean/v6/rank/list`, `/openapi/kmr/v2/rank/audio`).
- **Playlist plaza**: browse public playlists by scene, mood, style,
  language and era (`/pubsongs/v1/get_tags_by_type`,
  `/v2/special_recommend`), with 加载更多.
- **Private radio**: a batch of songs picked from the account's taste
  (`/v2/personal_recommend`), refreshable from the home view.
- **MVs**: a search tab for music videos and an in-app player with a
  quality picker (blue-ray / ultra / high / standard / fluent); the play
  endpoint needs the web signature and a short id resolved from the MV
  page (`wwwapi.kugou.com/play/mv`).
- **Comments**: a song's comments, with likes, replies and paging
  (`/mcomment/v1/cmtlist`).
- **Play modes**: list repeat, repeat one and shuffle, cycled from the
  player bar; repeat-one replays on track end, manual next still skips.
- **Queue and downloads**: the player bar opens the live queue (click a
  row to jump) and copies the playing file to `~/Music/mygou`.
- **Settings**: default quality (synced with the player bar), playback
  cache size with a clear button, and the version. The cache is capped
  at 1 GiB: older files are dropped first, never the playing one.
- **Translated lyrics**: KRC ships translations in a base64 `[language:]`
  line; the lyric view shows them under each line.
- **Artist and album pages**: singer and album names in every song table
  are links; the artist page shows avatar, counts and intro, the album page
  its cover and track list. Singers without an id are resolved by name.
- **Automatic daily VIP**: the concept edition earns VIP hours by
  reporting a played ad; the app claims every three hours (up to eight
  times a day) while it runs and on launch, keeping the account VIP, so
  320 keeps working.
- **QR login** with the KuGou concept edition app; the token is kept in
  `~/Library/Application Support/mygou/account.json`.
- **My music**: the account's playlists, with songs and album covers.
- **Favorites**: the heart reflects whether the playing song is in
  我喜欢 and adds it when it is not. Removing is not wired: the delete
  endpoint wants a list-entry id that the public track endpoints do not
  return.
- **Lyrics**: KRC/LRC lyrics, timed and scrolling, in the full screen
  now-playing view.
- **Recent plays** on the home view, persisted locally.
- **Menu bar mini player**: cover, title, progress, play/pause, previous,
  next, open the main window.
- **Media keys** and Control Centre now playing (macOS).
- **Themes**: dark and light, following the YesPlayMusic design language.

## Build

```sh
# once
(cd frontend && npm install)

# a runnable .app in build/
./scripts/build-app.sh
open build/mygou.app
```

For a quick run from source (the front end must be built first, it is
embedded in the binary):

```sh
(cd frontend && npm run build)
go run .
```

Requirements: macOS 12+, Go 1.24+ (uses `mygo`, see `go.mod` replace),
Node 20+.

## Layout

| Path | What it is |
| --- | --- |
| `internal/kugou` | API client: device registration, signing, search, privileges, song URLs, lyrics, login, playlists |
| `internal/player` | resolves a track to a URL, caches the file, decodes and plays it |
| `internal/service` | the bridge the page calls (search, queue, login, playlists, favorites, lyrics, account) |
| `tray.go` | menu bar icon and native mini player |
| `frontend/` | Vue 3 + Vite UI |
| `packaging/`, `scripts/` | app bundle and icon |

## Data

- `~/Library/Application Support/mygou`: `device.json`, `account.json`,
  `recent.json`
- `~/Library/Caches/mygou/tracks`: decoded-source files, named by hash,
  quality and whether they are a trial excerpt

## Development switches

| Variable | Effect |
| --- | --- |
| `MYGOU_URL` | page path to open, e.g. `/?q=海阔天空&play=1` |
| `MYGOU_DATA` | data directory (defaults to Application Support) |
| `MYGOU_QUALITY` | preferred quality: `128`, `320`, `flac` |
| `MYGOU_PANEL=1` | open the menu bar mini player at launch |
| `MYGOU_AUTOCLOSE=6` | close the main window after 6 s (background playback test) |
| `MYGOU_EXIT=20` | quit the app after 20 s |
| `MYGOU_LOG=1` | log the player state every five seconds |

## 免责声明 (Disclaimer)

1. 本项目仅供个人学习、技术研究与日常开发交流使用，请勿用于任何商业用途。
2. 本项目不提供任何音频下载付费绕过、版权保护破解等功能；所有音乐音频、封面、歌词及相关元数据版权均归属其原始权利人（酷狗音乐及其版权合作方）所有。
3. 用户使用本项目所产生的一切后果及法律责任由使用者自行承担，与本项目开发者无关。
4. 如相关权利方认为本项目侵犯了您的合法权益，请联系开发者，我们将第一时间配合处理。

## 致谢 (Credits)

- [KuGouMusicApi](https://github.com/MakcRe/KuGouMusicApi) - 酷狗 API 协议与加密逻辑参考
- [mygo](https://github.com/egoist/mygo) - 轻量级 Go 跨平台应用与原生窗口绑定
- [YesPlayMusic](https://github.com/qier222/YesPlayMusic) - 界面设计语言与交互参考
- [Vue.js](https://github.com/vuejs/core) - 前端响应式界面框架

## 开源协议 (License)

本项目采用 [MIT 许可证](LICENSE) 开源。


