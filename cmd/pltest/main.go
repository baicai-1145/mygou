// Command pltest lists the signed in account's playlists and the songs of
// the first one, to check the playlist endpoints without the window.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"mygou/internal/kugou"
)

func main() {
	dir := os.Getenv("MYGOU_DATA")
	if dir == "" {
		dir = filepath.Join(".tmp", "appdata")
	}
	c, err := kugou.NewClient(dir)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	c.Debug = os.Getenv("MYGOU_HTTP_DEBUG") == "1"
	if err := c.EnsureDFID(ctx, dir); err != nil {
		panic(err)
	}
	if err := c.LoadAccount(dir); err != nil {
		panic(err)
	}
	if len(os.Args) > 1 && os.Args[1] == "whoami" {
		p, err := c.Profile(ctx)
		fmt.Printf("token=%v user=%d profile=%+v err=%v\n", c.Token != "", c.UserID, p, err)
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "lyric" {
		tracks, err := c.Search(ctx, os.Args[2], 1, 1)
		if err != nil || len(tracks) == 0 {
			panic(err)
		}
		t := tracks[0]
		cands, err := c.SearchLyric(ctx, t.Hash, t.Name, t.Duration)
		if err != nil || len(cands) == 0 {
			panic(err)
		}
		text, err := c.GetLyric(ctx, cands[0].ID, cands[0].AccessKey)
		if err != nil {
			panic(err)
		}
		fmt.Printf("song %s - %s lines=%d\n", t.Name, t.Singer, len(strings.Split(text, "\n")))
		for i, line := range strings.Split(text, "\n") {
			if i > 40 {
				break
			}
			fmt.Printf("%02d %s\n", i, line)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "mv" {
		list, err := c.SearchMVs(ctx, os.Args[2], 3)
		if err != nil {
			panic(err)
		}
		for _, m := range list {
			fmt.Printf("mv %s %s - %s dur=%d\n", m.ID, m.Name, m.Singer, m.Duration)
		}
		if len(list) > 0 {
			q := ""
			if len(os.Args) > 3 {
				q = os.Args[3]
			}
			got, err := c.MVPlay(ctx, list[0].ID, q)
			if err != nil {
				panic(err)
			}
			fmt.Printf("play %s url=%s\n", got.Name, got.URL)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "cmt" {
		list, total, err := c.Comments(ctx, os.Args[2], 1, 5)
		if err != nil {
			panic(err)
		}
		fmt.Printf("comments total=%d n=%d\n", total, len(list))
		for i, cm := range list {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s (%d 赞, %d 回复): %s\n", cm.User, cm.Likes, cm.Replies, cm.Content)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "fm" {
		tracks, err := c.PersonalFM(ctx, 10)
		if err != nil {
			panic(err)
		}
		fmt.Printf("fm: %d tracks\n", len(tracks))
		for i, t := range tracks {
			if i >= 6 {
				break
			}
			fmt.Printf("  %s - %s dur=%d album=%s\n", t.Name, t.Singer, t.Duration, t.Album)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "plaza" {
		cat := "0"
		if len(os.Args) > 2 {
			cat = os.Args[2]
		}
		tags, err := c.PlaylistTags(ctx)
		if err != nil {
			panic(err)
		}
		for _, t := range tags {
			fmt.Printf("tag %s=%s children=%d\n", t.Name, t.ID, len(t.Children))
		}
		page, err := c.PlaylistPlaza(ctx, cat, 1, 6)
		if err != nil {
			panic(err)
		}
		fmt.Printf("plaza cat=%s n=%d hasNext=%v\n", cat, len(page.Playlists), page.HasNext)
		for i, p := range page.Playlists {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s (%d 首, %s) %s\n", p.Name, p.Count, p.Creator, p.ID)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "pl" {
		list, tracks, err := c.PlaylistTracks(ctx, os.Args[2], 1, 5)
		if err != nil {
			panic(err)
		}
		fmt.Printf("playlist %q creator=%q count=%d tracks=%d\n", list.Name, list.Creator, list.Count, len(tracks))
		for i, t := range tracks {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s - %s\n", t.Name, t.Singer)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "findartist" {
		list, err := c.SearchArtists(ctx, os.Args[2], 3)
		if err != nil {
			panic(err)
		}
		for _, a := range list {
			fmt.Printf("%s id=%s fans=%d songs=%d avatar=%s\n", a.Name, a.ID, a.Fans, a.Songs, a.Avatar)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "search" {
		tracks, err := c.Search(ctx, os.Args[2], 1, 3)
		if err != nil {
			panic(err)
		}
		for _, t := range tracks {
			fmt.Printf("%s - %s singerID=%q albumID=%q mix=%q\n", t.Name, t.Singer, t.SingerID, t.AlbumID, t.MixSongID)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "artist" {
		a, err := c.Artist(ctx, os.Args[2])
		if err != nil {
			panic(err)
		}
		fmt.Printf("artist: %s fans=%d songs=%d albums=%d avatar=%s\n", a.Name, a.Fans, a.Songs, a.Albums, a.Avatar)
		songs, err := c.ArtistSongs(ctx, os.Args[2], 1, 5)
		if err != nil {
			panic(err)
		}
		fmt.Printf("  songs: %d\n", len(songs))
		for i, t := range songs {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s - %s dur=%d album=%s\n", t.Name, t.Singer, t.Duration, t.Album)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "album" {
		size := 5
		if len(os.Args) > 3 {
			size, _ = strconv.Atoi(os.Args[3])
		}
		songs, err := c.AlbumSongs(ctx, os.Args[2], 1, size)
		if err != nil {
			panic(err)
		}
		fmt.Printf("album %s: %d songs\n", os.Args[2], len(songs))
		for i, t := range songs {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s - %s album=%s cover=%s\n", t.Name, t.Singer, t.Album, t.Cover)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "ranks" {
		ranks, err := c.Ranks(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Printf("ranks: %d\n", len(ranks))
		for i, r := range ranks {
			if i >= 5 {
				break
			}
			fmt.Printf("  %s %s (%s) %s\n", r.ID, r.Name, r.Frequency, r.Cover)
		}
		return
	}
	if len(os.Args) > 2 && os.Args[1] == "rank" {
		songs, err := c.RankSongs(ctx, os.Args[2], 1, 5)
		if err != nil {
			panic(err)
		}
		fmt.Printf("rank %s: %d songs\n", os.Args[2], len(songs))
		for i, t := range songs {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s - %s dur=%d hash=%s cover=%s\n", t.Name, t.Singer, t.Duration, t.Hash, t.Cover)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "vip" {
		c, err := c.ClaimVIP(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Printf("vip: %s\n", c.String())
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "daily" {
		tracks, err := c.DailyRecommend(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Printf("daily: %d tracks\n", len(tracks))
		for i, t := range tracks {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s - %s dur=%d cover=%s\n", t.Name, t.Singer, t.Duration, t.Cover)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "profile" {
		p, err := c.Profile(ctx)
		if err != nil {
			panic(err)
		}
		fmt.Printf("profile: name=%q avatar=%q\n", p.Name, p.Avatar)
		return
	}
	lists, err := c.UserPlaylists(ctx)
	if err != nil {
		panic(err)
	}
	fmt.Printf("playlists: %d\n", len(lists))
	for _, l := range lists {
		fmt.Printf("  %-12s count=%d id=%s cover=%s\n", l.Name, l.Count, l.ID, l.Cover)
	}
	if len(lists) == 0 {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "fav" {
		tracks, err := c.Search(ctx, "海阔天空", 1, 1)
		if err != nil || len(tracks) == 0 {
			panic(err)
		}
		t := tracks[0]
		if err := c.AddToPlaylist(ctx, 2, t.Name+" - "+t.Singer, t.Hash, t.AlbumID, t.MixSongID); err != nil {
			panic(err)
		}
		fmt.Printf("added %q to 我喜欢\n", t.Name)
	}
	list, tracks, err := c.PlaylistTracks(ctx, lists[1].ID, 1, 60)
	if err != nil {
		panic(err)
	}
	fmt.Printf("list %q: %d tracks, cover=%s\n", list.Name, len(tracks), list.Cover)
	for i, t := range tracks {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. %-30s %-12s album=%q dur=%d cover=%s\n", i+1, t.Name, t.Singer, t.Album, t.Duration, t.Cover)
	}
}
