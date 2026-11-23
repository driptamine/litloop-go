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

func TestLiveRedisCrossReplica(tester *testing.T) {
	if testing.Short() {
		tester.Skip("Redis integration test")
	}
	raw := os.Getenv("TEST_REDIS_URL")
	if raw == "" {
		tester.Skip("set TEST_REDIS_URL for Redis integration")
	}
	fixture := setup(tester)
	bob, bid := fixture.second()
	if err := fixture.s.Hub.ConnectRedis(raw); err != nil {
		tester.Fatal(err)
	}
	defer fixture.s.Hub.Close()
	cfg := fixture.s.Config
	cfg.RedisURL = raw
	second := New(fixture.s.DB, cfg)
	defer second.Hub.Close()
	http2 := httptest.NewServer(second)
	defer http2.Close()
	chat := fixture.req("GET", fmt.Sprintf("/chats/direct/%d", bid), fixture.Token, nil, 200)
	id := int64(chat["id"].(float64))
	Conn, _, err := Websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(http2.URL, "http")+fmt.Sprintf("/ws/chat/%d?token=%s", id, bob), nil)
	if err != nil {
		tester.Fatal(err)
	}
	defer Conn.Close()
	deadline := time.Now().Add(3 * time.Second)
	for {
		ready := second.Hub.Count(fmt.Sprint(id)) > 0
		if ready {
			break
		}
		if time.Now().After(deadline) {
			tester.Fatal("websocket did not join")
		}
		time.Sleep(time.Millisecond)
	}
	fixture.req("POST", fmt.Sprintf("/chats/%d/send", id), fixture.Token, map[string]any{"text": "Across replicas"}, 200)
	_ = Conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var frame map[string]any
	if err = Conn.ReadJSON(&frame); err != nil {
		tester.Fatal(err)
	}
	if frame["message"].(map[string]any)["text"] != "Across replicas" {
		tester.Fatal(frame)
	}
}
func TestLiveFFmpegEncoding(tester *testing.T) {
	if testing.Short() {
		tester.Skip("FFmpeg integration test")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		tester.Skip("ffmpeg unavailable")
	}
	fixture := setup(tester)
	path := filepath.Join(fixture.s.Config.MediaDir, "source.mp4")
	command := exec.Command("ffmpeg", "-nostdin", "-y", "-f", "lavfi", "-i", "color=c=blue:s=160x120:r=12", "-t", "1", "-c:v", "libx264", "-pix_fmt", "yuv420p", path)
	if out, err := command.CombinedOutput(); err != nil {
		tester.Fatalf("fixture ffmpeg: %v %s", err, out)
	}
	video := models.Video{UserID: &fixture.user, S3Key: "local/source.mp4", Filename: "source.mp4", Visibility: "private", Status: "draft"}
	if err := fixture.s.DB.Create(&video).Error; err != nil {
		tester.Fatal(err)
	}
	queued := fixture.req("POST", fmt.Sprintf("/videos/%d/encode", video.ID), fixture.Token, map[string]any{}, 202)
	var task models.Task
	if err := fixture.s.DB.First(&task, int64(queued["id"].(float64))).Error; err != nil {
		tester.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := worker.New(fixture.s.DB, fixture.s.Config).Process(ctx, task); err != nil {
		tester.Fatal(err)
	}
	detail := fixture.req("GET", fmt.Sprintf("/videos/%d", video.ID), fixture.Token, nil, 200)
	res, err := http.Get(detail["hls_url"].(string))
	if err != nil {
		tester.Fatal(err)
	}
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(body), "#EXTM3U") {
		tester.Fatal(res.Status, string(body))
	}
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "http") {
			segment, err := http.Get(line)
			if err != nil {
				tester.Fatal(err)
			}
			segment.Body.Close()
			if segment.StatusCode != 200 {
				tester.Fatal(segment.Status)
			}
		}
	}
}
