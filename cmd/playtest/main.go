// Command playtest plays the first result of a search for a few seconds and
// prints the engine state, to check playback without the window.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"mygou/internal/kugou"
	"mygou/internal/player"
)

func main() {
	keyword := "海阔天空"
	if len(os.Args) > 1 {
		keyword = os.Args[1]
	}
	dir := os.Getenv("MYGOU_DATA")
	if dir == "" {
		dir = filepath.Join(".tmp", "appdata")
	}
	c, err := kugou.NewClient(dir)
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	if err := c.EnsureDFID(ctx, dir); err != nil {
		panic(err)
	}
	if err := c.LoadAccount(dir); err != nil {
		panic(err)
	}
	quality := "320"
	if len(os.Args) > 2 {
		quality = os.Args[2]
	}
	fmt.Printf("account: loggedIn=%v userid=%d quality=%s\n", c.Token != "", c.UserID, quality)
	tracks, err := c.Search(ctx, keyword, 1, 3)
	if err != nil {
		panic(err)
	}
	if len(tracks) == 0 {
		panic("no tracks")
	}
	fmt.Printf("playing %s - %s\n", tracks[0].Name, tracks[0].Singer)
	e := player.New(c, filepath.Join(".tmp", "cache"), quality)
	e.Play(tracks[0])
	for i := 0; i < 24; i++ {
		time.Sleep(500 * time.Millisecond)
		s := e.State()
		fmt.Printf("%4.1fs pos=%.2f dur=%.1f playing=%v loading=%v q=%s err=%q\n",
			float64(i+1)/2, s.Position, s.Duration, s.Playing, s.Loading, s.Quality, s.Error)
		if s.Error != "" {
			os.Exit(1)
		}
	}
}
