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
	DB	*gorm.DB
	Config	config.Config
	Client	*http.Client
	astra	*engagement.Store
}

func New(db *gorm.DB, appConfig config.Config) *Worker {
	return &Worker{DB: db, Config: appConfig, Client: &http.Client{Timeout: 10 * time.Minute}}
}
func (worker *Worker) claim() (*models.Task, error) {
	var task models.Task
	err := worker.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if err := tx.Model(&models.Task{}).Where("status = ? AND locked_at < ?", "processing", now.Add(-35*time.Minute)).Updates(map[string]any{"status": "pending", "locked_at": nil}).Error; err != nil {
			return err
		}
		pending := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("status = ? AND run_at <= ?", "pending", now)
		if !engagement.Enabled(worker.Config) {
			pending = pending.Where("kind <> ?", engagement.TaskKind)
		}
		if err := pending.Order("id").First(&task).Error; err != nil {
			return err
		}
		task.Status = "processing"
		task.LockedAt = &now
		task.Attempts++
		return tx.Save(&task).Error
	})
	if err != nil {
		return nil, err
	}
	return &task, nil
}
func (worker *Worker) Run(ctx context.Context) error {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	defer func() {
		if worker.astra != nil {
			worker.astra.Close()
			worker.astra = nil
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		task, err := worker.claim()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
			}
			continue
		}
		if err != nil {
			return err
		}
		jobctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
		err = worker.Process(jobctx, *task)
		cancel()
		if err == nil && task.Kind == engagement.TaskKind {
			if err := worker.DB.Delete(task).Error; err != nil {
				return err
			}
			continue
		}
		updates := map[string]any{"locked_at": nil, "status": "completed", "error": ""}
		if err != nil {
			updates["error"] = err.Error()
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
					if updateErr := worker.DB.Model(&models.VoiceMessage{}).Where("id = ?", payload.VoiceID).Update("transcription_status", "failed").Error; updateErr != nil {
						return updateErr
					}
				}
			}
			slog.Error("task failed", "id", task.ID, "kind", task.Kind, "error", err)
		}
		if err = worker.DB.Model(task).Updates(updates).Error; err != nil {
			return err
		}
	}
}
func (worker *Worker) Process(ctx context.Context, task models.Task) error {
	if task.Kind == engagement.TaskKind {
		var event engagement.Event
		if err := json.Unmarshal(task.Payload, &event); err != nil {
			return err
		}
		if worker.astra == nil {
			store, err := engagement.Open(ctx, worker.Config)
			if err != nil {
				return err
			}
			worker.astra = store
		}
		return worker.astra.Record(ctx, task.ID, event)
	}
	var payload struct {
		VideoID	int64	`json:"video_id"`
		VoiceID	int64	`json:"voice_id"`
		URL	string	`json:"url"`
	}
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return err
	}
	switch task.Kind {
	case "encode_video":
		return worker.encode(ctx, payload.VideoID)
	case "transcribe":
		return worker.transcribe(ctx, payload.VoiceID, payload.URL)
	default:
		return fmt.Errorf("unknown task kind %q", task.Kind)
	}
}
func (worker *Worker) transcribe(ctx context.Context, id int64, uri string) error {
	var voice models.VoiceMessage
	if err := worker.DB.First(&voice, id).Error; err != nil {
		return err
	}
	uri = storage.MediaURL(worker.Config, voice.R2Key, voice.GCSKey, voice.S3Key)
	if uri == "" {
		return fmt.Errorf("voice source is unavailable")
	}
	if worker.Config.TranscriptionURL == "" {
		return fmt.Errorf("TRANSCRIPTION_URL is not configured")
	}
	if err := worker.DB.Model(&models.VoiceMessage{}).Where("id = ?", id).Update("transcription_status", "processing").Error; err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]any{"voice_id": id, "url": uri, "callback_url": strings.TrimRight(worker.Config.PublicURL, "/") + "/chats/voice/transcribe-callback/"})
	req, err := http.NewRequestWithContext(ctx, "POST", worker.Config.TranscriptionURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+worker.Config.TranscriptionSecret)
	res, err := worker.Client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("transcriber HTTP %d", res.StatusCode)
	}
	if res.StatusCode == 202 {
		return nil
	}
	var result struct {
		Text		string	`json:"text"`
		Transcription	string	`json:"transcription"`
		Language	string	`json:"language"`
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&result); err != nil {
		return err
	}
	if result.Transcription == "" {
		result.Transcription = result.Text
	}
	return worker.DB.Model(&models.VoiceMessage{}).Where("id = ?", id).Updates(map[string]any{"transcription": result.Transcription, "transcription_language": result.Language, "transcription_status": "completed", "transcribed_at": time.Now()}).Error
}
func (worker *Worker) encode(ctx context.Context, id int64) error {
	var video models.Video
	if err := worker.DB.First(&video, id).Error; err != nil {
		return err
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
	tmp, err := os.MkdirTemp("", "litloop-encode-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	input := filepath.Join(tmp, "input")
	if strings.HasPrefix(key, "local/") {
		input = filepath.Join(worker.Config.MediaDir, strings.TrimPrefix(key, "local/"))
	} else {
		uri := key
		if !strings.HasPrefix(uri, "https://") && !strings.HasPrefix(uri, "http://") {
			uri = storage.MediaURL(worker.Config, video.R2Key, video.GCSKey, video.S3Key)
			if uri == "" {
				return fmt.Errorf("%s video source is unavailable", provider)
			}
		}
		req, err := http.NewRequestWithContext(ctx, "GET", uri, nil)
		if err != nil {
			return err
		}
		res, err := worker.Client.Do(req)
		if err != nil {
			return err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return fmt.Errorf("download HTTP %d", res.StatusCode)
		}
		file, err := os.Create(input)
		if err != nil {
			return err
		}
		copied, copyErr := io.Copy(file, io.LimitReader(res.Body, (2<<30)+1))
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
		if copied > 2<<30 {
			return fmt.Errorf("video exceeds 2 GiB")
		}
	}
	rel := fmt.Sprintf("hls/%d", id)
	output := filepath.Join(worker.Config.MediaDir, rel)
	if worker.Config.UsesR2ForMedia() {
		output = filepath.Join(tmp, "hls")
	}
	if err = os.MkdirAll(output, 0755); err != nil {
		return err
	}
	manifest := filepath.Join(output, "index.m3u8")
	segment := filepath.Join(output, "segment-%05d.ts")
	cmd := exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-y", "-protocol_whitelist", "file,pipe,crypto,data", "-i", input, "-c:v", "libx264", "-preset", "fast", "-c:a", "aac", "-hls_time", "5", "-hls_list_size", "0", "-hls_segment_filename", segment, manifest)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %w: %s", err, tail(out))
	}
	thumb := filepath.Join(output, "thumbnail.jpg")
	cmd = exec.CommandContext(ctx, "ffmpeg", "-nostdin", "-y", "-i", input, "-frames:v", "1", "-q:v", "2", thumb)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("thumbnail: %w: %s", err, tail(out))
	}
	location := "local/" + rel
	if worker.Config.UsesR2ForMedia() {
		rel += "/" + rand.Text()
		if err := worker.uploadHLS(ctx, output, rel); err != nil {
			return err
		}
		location = "r2/" + rel
	}
	return worker.DB.Model(&video).Updates(map[string]any{"hls_s3": location + "/index.m3u8", "thumbnail": location + "/thumbnail.jpg"}).Error
}
func tail(data []byte) string {
	if len(data) > 2000 {
		data = data[len(data)-2000:]
	}
	return string(data)
}
