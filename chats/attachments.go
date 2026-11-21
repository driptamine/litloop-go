package chats

import (
	"encoding/hex"
	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
	"net/url"
	"strconv"
	"strings"
)

func chatAttachmentFilename(name string) bool {
	id, _, _ := strings.Cut(name, ".")
	if len(id) != 48 || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	_, err := hex.DecodeString(id)
	return err == nil
}

func (s *Handler) ownsR2Attachment(userID int64, kind, key string) bool {
	name, found := strings.CutPrefix(key, storage.ChatAttachmentPrefix(userID))
	if found {
		return chatAttachmentFilename(name)
	}
	if key == "" {
		return false
	}

	var media any
	switch kind {
	case "photo":
		media = &models.Photo{}
	case "video":
		media = &models.Video{}
	case "track":
		media = &models.Track{}
	default:
		return false
	}
	var id int64
	err := s.DB.Model(media).Select("id").Where("user_id = ? AND r2_key = ?", userID, key).Limit(1).Scan(&id).Error
	return err == nil && id != 0
}

func (s *Handler) storeAttachmentKeys(attachments []any, userID int64) {
	for _, item := range attachments {
		attachment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		delete(attachment, "r2_key")
		key := s.r2AttachmentKey(Str(attachment["url"]))
		if s.ownsR2Attachment(userID, Str(attachment["type"]), key) {
			delete(attachment, "url")
			attachment["r2_key"] = key
		}
	}
}

func (s *Handler) r2AttachmentKey(rawURL string) string {
	if s.Config.R2Bucket == "" || s.Config.R2Endpoint == "" {
		return ""
	}
	endpoint, err := url.Parse(s.Config.R2Endpoint)
	if err != nil || endpoint.Host == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != endpoint.Scheme || !strings.EqualFold(u.Host, endpoint.Host) {
		return ""
	}
	bucketPath := strings.TrimRight(endpoint.Path, "/") + "/" + s.Config.R2Bucket + "/"
	key, found := strings.CutPrefix(u.Path, bucketPath)
	if !found {
		return ""
	}
	return key
}

func (s *Handler) setAttachmentURLs(attachments []any, message models.Message) {
	for _, item := range attachments {
		attachment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := Str(attachment["r2_key"])
		delete(attachment, "r2_key")
		if key == "" {
			// Messages created before keys were stored can contain a signed R2 URL.
			key = s.r2AttachmentKey(Str(attachment["url"]))
		}
		if message.UserID != nil && s.ownsR2Attachment(*message.UserID, Str(attachment["type"]), key) {
			attachment["url"] = storage.PresignedURL(s.Config, "r2", key)
			continue
		}
		if key := s.legacyAttachmentKey(Str(attachment["url"]), message.CreatedAt.Unix()); key != "" {
			attachment["url"] = s.LocalURL(key)
		}
	}
}

func (s *Handler) legacyAttachmentKey(rawURL string, sentAt int64) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	name, found := strings.CutPrefix(u.Path, "/media/attachments/")
	if !found || !chatAttachmentFilename(name) {
		return ""
	}
	key := "attachments/" + name
	values := u.Query()
	expires := values.Get("expires")
	expiry, err := strconv.ParseInt(expires, 10, 64)
	// The stored URL had to be valid when the message was sent.
	if err != nil || expiry < sentAt || !SecureEqual(values.Get("signature"), s.SignMedia(key, expires)) {
		return ""
	}
	return key
}
