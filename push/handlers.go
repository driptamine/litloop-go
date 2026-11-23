package push

import (
	"encoding/json"
	"fmt"
	"log/slog"

	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"

	webpush "github.com/SherClockHolmes/webpush-go"
	"gorm.io/gorm/clause"
)

func (handler *Handler) vapidConfigured() bool {
	return handler.Config.VAPIDPublicKey != "" && handler.Config.VAPIDPrivateKey != ""
}

// VAPIDKey exposes the public VAPID key so browsers can subscribe for push.
func (handler *Handler) VAPIDKey(req *Request) any {
	if !handler.vapidConfigured() {
		Fail(503, "Web push is not configured")
	}
	return map[string]any{"publicKey": handler.Config.VAPIDPublicKey}
}

// Subscribe stores or refreshes a browser push subscription for the viewer.
func (handler *Handler) Subscribe(req *Request) any {
	if !handler.vapidConfigured() {
		Fail(503, "Web push is not configured")
	}
	body := req.Body()
	endpoint := Str(body["endpoint"])
	keys, _ := body["keys"].(map[string]any)
	if endpoint == "" || Str(keys["p256dh"]) == "" || Str(keys["auth"]) == "" {
		Fail(400, "endpoint and keys are required")
	}
	subscription := models.PushSubscription{UserID: req.User.ID, Endpoint: endpoint, P256DH: Str(keys["p256dh"]), Auth: Str(keys["auth"])}
	Check(handler.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "endpoint"}},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "p256dh", "auth"}),
	}).Create(&subscription).Error)
	return map[string]any{"status": "subscribed"}
}

// Unsubscribe removes a browser push subscription of the viewer.
func (handler *Handler) Unsubscribe(req *Request) any {
	endpoint := Str(req.Body()["endpoint"])
	if endpoint == "" {
		Fail(400, "endpoint is required")
	}
	Check(handler.DB.Where("user_id = ? AND endpoint = ?", req.User.ID, endpoint).Delete(&models.PushSubscription{}).Error)
	return map[string]any{"status": "unsubscribed"}
}

type directPush struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// SendDirectMessage pushes a new direct message to every browser subscription
// of the recipients. It runs in its own goroutine; delivery failures only drop
// dead subscriptions. Group chats and saved messages never push.
func (handler *Handler) SendDirectMessage(chat models.Chat, sender models.User, recipients []models.ChatParticipant, text string, voiceID int64, attachments []any) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("push send failed", "error", recovered)
		}
	}()
	if chat.ChatType != "direct" || chat.IsSavedMessages || !handler.vapidConfigured() || len(recipients) == 0 {
		return
	}
	ids := make([]int64, 0, len(recipients))
	for _, recipient := range recipients {
		ids = append(ids, recipient.UserID)
	}
	subscriptions := All[models.PushSubscription](handler.DB.Where("user_id IN ?", ids))
	if len(subscriptions) == 0 {
		return
	}
	body := Truncate(text, 100)
	if body == "" {
		body = "sent a voice message"
		if len(attachments) > 0 {
			body = "sent an attachment"
		}
	}
	if voiceID > 0 && text == "" && len(attachments) == 0 {
		body = "sent a voice message"
	}
	payload, err := json.Marshal(directPush{Title: sender.Username, Body: body, URL: fmt.Sprintf("/chat/%d", sender.ID)})
	if err != nil {
		slog.Error("push payload failed", "error", err)
		return
	}
	options := &webpush.Options{
		Subscriber:      handler.Config.VAPIDSubject,
		VAPIDPublicKey:  handler.Config.VAPIDPublicKey,
		VAPIDPrivateKey: handler.Config.VAPIDPrivateKey,
		TTL:             86400,
	}
	for _, subscription := range subscriptions {
		pushSubscription := &webpush.Subscription{Endpoint: subscription.Endpoint, Keys: webpush.Keys{P256dh: subscription.P256DH, Auth: subscription.Auth}}
		response, err := webpush.SendNotification(payload, pushSubscription, options)
		if response != nil && (response.StatusCode == 404 || response.StatusCode == 410) {
			Check(handler.DB.Where("id = ?", subscription.ID).Delete(&models.PushSubscription{}).Error)
			continue
		}
		if err != nil {
			slog.Error("push delivery failed", "endpoint", subscription.Endpoint, "error", err)
		}
	}
}
