package worker

import (
	"context"
	"litloop/backend/internal/storage"
	"os"
	"path/filepath"
)

func (worker *Worker) uploadHLS(ctx context.Context, dir, keyPrefix string) error {
	store, err := storage.New(worker.Config, "r2")
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || name == "index.m3u8" {
			continue
		}
		contentType := "video/mp2t"
		if name == "thumbnail.jpg" {
			contentType = "image/jpeg"
		}
		if err := uploadHLSFile(ctx, store, dir, name, keyPrefix, contentType); err != nil {
			return err
		}
	}
	return uploadHLSFile(ctx, store, dir, "index.m3u8", keyPrefix, "application/vnd.apple.mpegurl")
}

func uploadHLSFile(ctx context.Context, store *storage.Store, dir, name, keyPrefix, contentType string) error {
	file, err := os.Open(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	defer file.Close()
	return store.Put(ctx, keyPrefix+"/"+name, contentType, file)
}
