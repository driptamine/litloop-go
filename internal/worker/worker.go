package worker

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"io"
	"litloop/backend/internal/config"
	"litloop/backend/internal/engagement"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Worker struct {
	DB     *gorm.DB
	Config config.Config
	Client *http.Client
	astra  *engagement.Store
}

func New(db *gorm.DB, c config.Config) *Worker {
	return &Worker{DB: db, Config: c, Client: &http.Client{Timeout: 10 * time.Minute}}
}
func (w *Worker) claim() (*models.Task, error) {
	var task models.Task
	e := w.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if e := tx.Model(&models.Task{}).Where("status = ? AND locked_at < ?", "processing", now.Add(-35*time.Minute)).Updates(map[string]any{"status": "pending", "locked_at": nil}).Error; e != nil {
			return e
		}
		pending := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND run_at <= ?", "pending", now)
		if !engagement.Enabled(w.Config) {
			pending = pending.Where("kind <> ?", engagement.TaskKind)
		}
		if e := pending.Order("id").First(&task).Error; e != nil {
			return e
		}
		task.Status = "processing"
		task.LockedAt = &now
		task.Attempts++
		return tx.Save(&task).Error
	})
	if e != nil {
		return nil, e
	}
	return &task, nil
}
func (w *Worker) Run(ctx context.Context) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	defer func() {
		if w.astra != nil {
			w.astra.Close()
			w.astra = nil
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		task, e := w.claim()
		if errors.Is(e, gorm.ErrRecordNotFound) {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
			}
			continue
		}
		if e != nil {
			return e
		}
		jobctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		e = w.Process(jobctx, *task)
		cancel()
		if e == nil && task.Kind == engagement.TaskKind {
			if err := w.DB.Delete(task).Error; err != nil {
				return err
			}
			continue
		}
		updates := map[string]any{"locked_at": nil, "status": "completed", "error": ""}
		if e != nil {
			updates["error"] = e.Error()
			updates["status"] = "failed"
			if task.Attempts < 3 || task.Kind == engagement.TaskKind {
				attempt := min(task.Attempts, 6)
				updates["status"] = "pending"
				updates["run_at"] = time.Now().Add(time.Duration(attempt*attempt) * time.Minute)
			} else if task.Kind == "transcribe" {
				var payload struct {
					VoiceID int64 `json:"voice_id"`
				}
				if json.Unmarshal(task.Payload, &payload) == nil {
					if updateErr := w.DB.Model(&models.VoiceMessage{}).Where("id = ?", payload.VoiceID).Update("transcription_status", "failed").Error; updateErr != nil {
						return updateErr
					}
				}
			}
			slog.Error("task failed", "id", task.ID, "kind", task.Kind, "error", e)
		}
		if e = w.DB.Model(task).Updates(updates).Error; e != nil {
			return e
		}
	}
}
func (w *Worker) Process(ctx context.Context, t models.Task) error {
	if t.Kind == engagement.TaskKind {
		var event engagement.Event
		if err := json.Unmarshal(t.Payload, &event); err != nil {
			return err
		}
		if w.astra == nil {
			store, err := engagement.Open(ctx, w.Config)
			if err != nil {
				return err
			}
			w.astra = store
		}
		return w.astra.Record(ctx, t.ID, event)
	}
	var p struct {
		VideoID int64  `json:"video_id"`
		VoiceID int64  `json:"voice_id"`
		URL     string `json:"url"`
	}
	if e := json.Unmarshal(t.Payload, &p); e != nil {
		return e
	}
	switch t.Kind {
	case "encode_video":
		return w.encode(ctx, p.VideoID)
	case "transcribe":
		return w.transcribe(ctx, p.VoiceID, p.URL)
	default:
		return fmt.Errorf("unknown task kind %q", t.Kind)
	}
}
func (w *Worker) transcribe(ctx context.Context, id int64, uri string) error {
	var voice models.VoiceMessage
	if e := w.DB.First(&voice, id).Error; e != nil {
		return e
	}
	uri = storage.MediaURL(w.Config, voice.R2Key, voice.GCSKey, voice.S3Key)
	if uri == "" {
		return fmt.Errorf("voice source is unavailable")
	}
	if w.Config.TranscriptionURL == "" {
		return fmt.Errorf("TRANSCRIPTION_URL is not configured")
	}
	if e := w.DB.Model(&models.VoiceMessage{}).Where("id = ?", id).Update("transcription_status", "processing").Error; e != nil {
		return e
	}
	body, _ := json.Marshal(map[string]any{"voice_id": id, "url": uri, "callback_url": strings.TrimRight(w.Config.PublicURL, "/") + "/chats/voice/transcribe-callback/"})
	req, e := http.NewRequestWithContext(ctx, "POST", w.Config.TranscriptionURL, bytes.NewReader(body))
	if e != nil {
		return e
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.Config.TranscriptionSecret)
	res, e := w.Client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("transcriber HTTP %d", res.StatusCode)
	}
	if res.StatusCode == 202 {
		return nil
	}
	var result struct {
		Text          string `json:"text"`
		Transcription string `json:"transcription"`
		Language      string `json:"language"`
	}
	if e = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&result); e != nil {
		return e
	}
	if result.Transcription == "" {
		result.Transcription = result.Text
	}
	return w.DB.Model(&models.VoiceMessage{}).Where("id = ?", id).Updates(map[string]any{"transcription": result.Transcription, "transcription_language": result.Language, "transcription_status": "completed", "transcribed_at": time.Now()}).Error
}
func (w *Worker) encode(ctx context.Context, id int64) error {
	var video models.Video
	if e := w.DB.First(&video, id).Error; e != nil {
		return e
	}
	key, provider := video.R2Key, "r2"
	if key == "" {
		key, provider = video.GCSKey, "gcs"
	}
	if key == "" {
		key, provider = video.S3Key, "s3"
	}
	if key == "" {
		return fmt.Errorf("video has no source")
	}
	tmp, e := os.MkdirTemp("", "litloop-encode-*")
	if e != nil {
		return e
	}
	defer os.RemoveAll(tmp)
	input := filepath.Join(tmp, "input")
	if strings.HasPrefix(key, "local/") {
		input = filepath.Join(w.Config.MediaDir, strings.TrimPrefix(key, "local/"))
	} else {
		uri := key
		if !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "http://") {
			uri = storage.MediaURL(w.Config, video.R2Key, video.GCSKey, video.S3Key)
			if uri == "" {
				return fmt.Errorf("%s video source is unavailable", provider)
			}
		}
		req, e := http.NewRequestWithContext(ctx, "GET", uri, nil)
		if e != nil {
			return e
		}
		res, e := w.Client.Do(req)
		if e != nil {
			return e
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return fmt.Errorf("download HTTP %d", res.StatusCode)
		}
		f, e := os.Create(input)
		if e != nil {
			return e
		}
		n, copyErr := io.Copy(f, io.LimitReader(res.Body, (2<<30)+1))
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if n > 2<<30 {
			return fmt.Errorf("video exceeds 2 GiB")
		}
	}
	rel := fmt.Sprintf("hls/%d", id)
	output := filepath.Join(w.Config.MediaDir, rel)
	if w.Config.UsesR2ForMedia() {
		output = filepath.Join(tmp, "hls")
	}
	if e = os.MkdirAll(output, 0755); e != nil {
		return e
	}
	manifest := filepath.Join(output, "index.m3u8")
	segment := filepath.Join(output, "segment-%05d.ts")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-y", "-protocol_whitelist", "file,pipe,crypto,data", "-i", input, "-c:v", "libx264", "-preset", "fast", "-c:a", "aac", "-hls_time", "5", "-hls_list_size", "0", "-hls_segment_filename", segment, manifest)
	if out, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("ffmpeg: %w: %s", e, tail(out))
	}
	thumb := filepath.Join(output, "thumbnail.jpg")
	cmd = exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-y", "-i", input, "-frames:v", "1", "-q:v", "2", thumb)
	if out, e := cmd.CombinedOutput(); e != nil {
		return fmt.Errorf("thumbnail: %w: %s", e, tail(out))
	}
	location := "local/" + rel
	if w.Config.UsesR2ForMedia() {
		rel += "/" + rand.Text()
		if e := w.uploadHLS(ctx, output, rel); e != nil {
			return e
		}
		location = "r2/" + rel
	}
	return w.DB.Model(&video).Updates(map[string]any{"hls_s3": location + "/index.m3u8", "thumbnail": location + "/thumbnail.jpg"}).Error
}
func tail(b []byte) string {
	if len(b) > 2000 {
		b = b[len(b)-2000:]
	}
	return string(b)
}
