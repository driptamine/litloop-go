package app

import (
	"context"
	"fmt"
	"io"
	"litloop/backend/internal/models"
	"litloop/backend/internal/worker"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	Websocket "github.com/gorilla/websocket"
)

func TestLiveRedisCrossReplica(t *testing.T) {
	if testing.Short() {
		t.Skip("Redis integration test")
	}
	raw := os.Getenv("TEST_REDIS_URL")
	if raw == "" {
		t.Skip("set TEST_REDIS_URL for Redis integration")
	}
	f := setup(t)
	bob, bid := f.second()
	if e := f.s.Hub.ConnectRedis(raw); e != nil {
		t.Fatal(e)
	}
	defer f.s.Hub.Close()
	cfg := f.s.Config
	cfg.RedisURL = raw
	second := New(f.s.DB, cfg)
	defer second.Hub.Close()
	http2 := httptest.NewServer(second)
	defer http2.Close()
	chat := f.req("GET", fmt.Sprintf("/chats/direct/%d", bid), f.Token, nil, 200)
	id := int64(chat["id"].(float64))
	Conn, _, e := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(http2.URL, "http")+fmt.Sprintf("/ws/chat/%d?token=%s", id, bob), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer Conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		ready := second.Hub.Count(fmt.Sprint(id)) > 0
		if ready {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("websocket did not join")
		}
		time.Sleep(time.Millisecond)
	}
	f.req("POST", fmt.Sprintf("/chats/%d/send", id), f.Token, map[string]any{"text": "Across replicas"}, 200)
	_ = Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var frame map[string]any
	if e = Conn.ReadJSON(&frame); e != nil {
		t.Fatal(e)
	}
	if frame["message"].(map[string]any)["text"] != "Across replicas" {
		t.Fatal(frame)
	}
}
func TestLiveFFmpegEncoding(t *testing.T) {
	if testing.Short() {
		t.Skip("FFmpeg integration test")
	}
	if _, e := exec.LookPath("ffmpeg"); e != nil {
		t.Skip("ffmpeg unavailable")
	}
	f := setup(t)
	path := filepath.Join(f.s.Config.MediaDir, "source.mp4")
	command := exec.Command("ffmpeg", "-nostdin", "-y", "-f", "lavfi", "-i", "color=c=blue:s=160x120:r=12", "-t", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path)
	if out, e := command.CombinedOutput(); e != nil {
		t.Fatalf("fixture ffmpeg: %v %s", e, out)
	}
	video := models.Video{UserID: &f.user, S3Key: "local/source.mp4", Filename: "source.mp4", Visibility: "private", Status: "draft"}
	if e := f.s.DB.Create(&video).Error; e != nil {
		t.Fatal(e)
	}
	queued := f.req("POST", fmt.Sprintf("/videos/%d/encode", video.ID), f.Token, map[string]any{}, 202)
	var task models.Task
	if e := f.s.DB.First(&task, int64(queued["id"].(float64))).Error; e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if e := worker.New(f.s.DB, f.s.Config).Process(ctx, task); e != nil {
		t.Fatal(e)
	}
	detail := f.req("GET", fmt.Sprintf("/videos/%d", video.ID), f.Token, nil, 200)
	res, e := http.Get(detail["hls_url"].(string))
	if e != nil {
		t.Fatal(e)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "#EXTM3U") {
		t.Fatal(res.Status, string(body))
	}
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "http") {
			segment, e := http.Get(line)
			if e != nil {
				t.Fatal(e)
			}
			segment.Body.Close()
			if segment.StatusCode != 200 {
				t.Fatal(segment.Status)
			}
		}
	}
}
