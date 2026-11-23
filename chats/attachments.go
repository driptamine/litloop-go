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

func (handler *Handler) ownsR2Attachment(userID int64, kind, key string) bool {
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
	err := handler.DB.Model(media).Select("id").Where("user_id = ? AND r2_key = ?", userID, key).Limit(1).Scan(&id).Error
	return err == nil && id != 0
}

func (handler *Handler) storeAttachmentKeys(attachments []any, userID int64) {
	for _, item := range attachments {
		attachment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		delete(attachment, "r2_key")
		key := handler.r2AttachmentKey(Str(attachment["url"]))
		if handler.ownsR2Attachment(userID, Str(attachment["type"]), key) {
			delete(attachment, "url")
			attachment["r2_key"] = key
		}
	}
}

func (handler *Handler) r2AttachmentKey(rawURL string) string {
	if handler.Config.R2Bucket == "" || handler.Config.R2Endpoint == "" {
		return ""
	}
	endpoint, err := url.Parse(handler.Config.R2Endpoint)
	if err != nil || endpoint.Host == "" {
		return ""
	}
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Scheme != endpoint.Scheme || !strings.EqualFold(parsedURL.Host, endpoint.Host) {
		return ""
	}
	bucketPath := strings.TrimRight(endpoint.Path, "/") + "/" + handler.Config.R2Bucket + "/"
	key, found := strings.CutPrefix(parsedURL.Path, bucketPath)
	if !found {
		return ""
	}
	return key
}

func (handler *Handler) setAttachmentURLs(attachments []any, message models.Message, owned map[r2Ownership]bool) {
	for _, item := range attachments {
		attachment, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := Str(attachment["r2_key"])
		delete(attachment, "r2_key")
		if key == "" {
			// Messages created before keys were stored can contain a signed R2 URL.
			key = handler.r2AttachmentKey(Str(attachment["url"]))
		}
		if message.UserID != nil && owned[r2Ownership{userID: *message.UserID, kind: Str(attachment["type"]), key: key}] {
			attachment["url"] = storage.PresignedURL(handler.Config, "r2", key)
			continue
		}
		if key := handler.legacyAttachmentKey(Str(attachment["url"]), message.CreatedAt.Unix()); key != "" {
			attachment["url"] = handler.LocalURL(key)
		}
	}
}

func (handler *Handler) legacyAttachmentKey(rawURL string, sentAt int64) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	name, found := strings.CutPrefix(parsedURL.Path, "/media/attachments/")
	if !found || !chatAttachmentFilename(name) {
		return ""
	}
	key := "attachments/" + name
	values := parsedURL.Query()
	expires := values.Get("expires")
	expiry, err := strconv.ParseInt(expires, 10, 64)
	// The stored URL had to be valid when the message was sent.
	if err != nil || expiry < sentAt || !SecureEqual(values.Get("signature"), handler.SignMedia(key, expires)) {
		return ""
	}
	return key
}
