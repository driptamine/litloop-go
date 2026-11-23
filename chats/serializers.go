package chats

import (
	"encoding/json"
	"strings"

	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
)

func messageAttachments(message models.Message) []any {
	var items []any
	if len(message.Attachments) == 0 {
		return nil
	}
	if err := json.Unmarshal(message.Attachments, &items); err != nil {
		return nil
	}
	return items
}

// messageBundle preloads everything message serialization needs so a chat
// with history costs a constant number of queries instead of several per
// message. Without it, opening a conversation runs hundreds of sequential
// queries and the request outlives the client's timeout.
type messageBundle struct {
	users	map[int64]models.User
	voices	map[int64]models.VoiceMessage
	owned	map[r2Ownership]bool
}

type r2Ownership struct {
	userID	int64
	kind	string
	key	string
}

func (handler *Handler) loadMessageBundle(msgs []models.Message) *messageBundle {
	bundle := &messageBundle{
		users:	map[int64]models.User{},
		voices:	map[int64]models.VoiceMessage{},
		owned:	map[r2Ownership]bool{},
	}
	if len(msgs) == 0 {
		return bundle
	}
	userIDs := make([]int64, 0, len(msgs))
	seenUsers := map[int64]bool{}
	msgIDs := make([]int64, 0, len(msgs))
	for _, message := range msgs {
		msgIDs = append(msgIDs, message.ID)
		if message.UserID != nil && !seenUsers[*message.UserID] {
			seenUsers[*message.UserID] = true
			userIDs = append(userIDs, *message.UserID)
		}
	}
	if len(userIDs) > 0 {
		for _, user := range All[models.User](handler.DB.Where("id IN ?", userIDs)) {
			bundle.users[user.ID] = user
		}
	}
	links := All[models.MessageVoice](handler.DB.Where("message_id IN ?", msgIDs))
	voiceIDs := make([]int64, 0, len(links))
	seenVoices := map[int64]bool{}
	for _, link := range links {
		if link.VoiceMessageID != nil && !seenVoices[*link.VoiceMessageID] {
			seenVoices[*link.VoiceMessageID] = true
			voiceIDs = append(voiceIDs, *link.VoiceMessageID)
		}
	}
	if len(voiceIDs) > 0 {
		byID := map[int64]models.VoiceMessage{}
		for _, voice := range All[models.VoiceMessage](handler.DB.Where("id IN ?", voiceIDs)) {
			byID[voice.ID] = voice
		}
		for _, link := range links {
			if link.MessageID != nil && link.VoiceMessageID != nil {
				if voice, ok := byID[*link.VoiceMessageID]; ok {
					bundle.voices[*link.MessageID] = voice
				}
			}
		}
	}
	bundle.owned = handler.loadR2Ownership(msgs)
	return bundle
}

// loadR2Ownership resolves, with at most one query per media kind, which of
// the stored attachment keys belong to each message author. Attachments that
// need no database check (chat-upload filenames, empty keys, legacy signed
// URLs) never reach a query.
func (handler *Handler) loadR2Ownership(msgs []models.Message) map[r2Ownership]bool {
	owned := map[r2Ownership]bool{}
	keysByKind := map[string]map[string]bool{}
	refs := []r2Ownership{}
	seen := map[r2Ownership]bool{}
	for _, message := range msgs {
		if message.UserID == nil {
			continue
		}
		for _, raw := range messageAttachments(message) {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			key := Str(item["r2_key"])
			if key == "" {
				key = handler.r2AttachmentKey(Str(item["url"]))
			}
			if key == "" {
				continue
			}
			ref := r2Ownership{userID: *message.UserID, kind: Str(item["type"]), key: key}
			if name, found := strings.CutPrefix(key, storage.ChatAttachmentPrefix(*message.UserID)); found && chatAttachmentFilename(name) {
				owned[ref] = true
				continue
			}
			if seen[ref] {
				continue
			}
			seen[ref] = true
			refs = append(refs, ref)
			if keysByKind[ref.kind] == nil {
				keysByKind[ref.kind] = map[string]bool{}
			}
			keysByKind[ref.kind][key] = true
		}
	}
	if len(refs) == 0 {
		return owned
	}
	type ownerRow struct {
		UserID	*int64
		R2Key	string
	}
	lookup := func(model any, kind string) {
		keys := make([]string, 0, len(keysByKind[kind]))
		for key := range keysByKind[kind] {
			keys = append(keys, key)
		}
		if len(keys) == 0 {
			return
		}
		var rows []ownerRow
		Check(handler.DB.Model(model).Select("user_id, r2_key").Where("r2_key IN ?", keys).Scan(&rows).Error)
		ownersByKey := map[string]map[int64]bool{}
		for _, row := range rows {
			if row.UserID != nil {
				if ownersByKey[row.R2Key] == nil {
					ownersByKey[row.R2Key] = map[int64]bool{}
				}
				ownersByKey[row.R2Key][*row.UserID] = true
			}
		}
		for _, ref := range refs {
			if ref.kind == kind && ownersByKey[ref.key][ref.userID] {
				owned[ref] = true
			}
		}
	}
	lookup(&models.Photo{}, "photo")
	lookup(&models.Video{}, "video")
	lookup(&models.Track{}, "track")
	return owned
}

func (handler *Handler) VoiceJSON(voice models.VoiceMessage) any {
	out := Object(voice)
	out["url"] = handler.MediaURL(voice.R2Key, voice.GCSKey, voice.S3Key)
	return out
}

func (handler *Handler) messageJSONWith(message models.Message, bundle *messageBundle) map[string]any {
	out := Object(message)
	if attachments, ok := out["attachments"].([]any); ok {
		handler.setAttachmentURLs(attachments, message, bundle.owned)
	}
	out["username"] = "AI"
	out["avatar"] = nil
	out["voice_message"] = nil
	if message.UserID != nil {
		if user, ok := bundle.users[*message.UserID]; ok {
			out["username"] = user.Username
			out["avatar"] = user.Avatar
		}
	}
	if voice, ok := bundle.voices[message.ID]; ok {
		out["voice_message"] = handler.VoiceJSON(voice)
	}
	return out
}

func (handler *Handler) MessageJSON(message models.Message) map[string]any {
	return handler.messageJSONWith(message, handler.loadMessageBundle([]models.Message{message}))
}

func (handler *Handler) messagesJSON(msgs []models.Message) []any {
	bundle := handler.loadMessageBundle(msgs)
	out := make([]any, 0, len(msgs))
	for _, message := range msgs {
		out = append(out, handler.messageJSONWith(message, bundle))
	}
	return out
}

// chatPreload carries everything ChatJSON needs for one conversation so the
// chat list can fetch shared data for all conversations up front.
type chatPreload struct {
	participants	[]models.User
	unread		int64
	last		*models.Message
	detail		[]models.Message
	bundle		*messageBundle
}

func (handler *Handler) loadChatPreload(chats []models.Chat, viewer int64, detail bool) map[int64]*chatPreload {
	out := make(map[int64]*chatPreload, len(chats))
	for _, comment := range chats {
		out[comment.ID] = &chatPreload{}
	}
	if len(chats) == 0 {
		return out
	}
	ids := make([]int64, 0, len(chats))
	for _, comment := range chats {
		ids = append(ids, comment.ID)
	}
	parts := All[models.ChatParticipant](handler.DB.Where("chat_id IN ?", ids))
	participantsByChat := map[int64][]int64{}
	userIDs := []int64{}
	seenUsers := map[int64]bool{}
	for _, post := range parts {
		participantsByChat[post.ChatID] = append(participantsByChat[post.ChatID], post.UserID)
		if !seenUsers[post.UserID] {
			seenUsers[post.UserID] = true
			userIDs = append(userIDs, post.UserID)
		}
	}
	usersByID := map[int64]models.User{}
	if len(userIDs) > 0 {
		for _, user := range All[models.User](handler.DB.Where("id IN ?", userIDs)) {
			usersByID[user.ID] = user
		}
	}
	type unreadRow struct {
		ChatID	int64
		N	int64
	}
	var unreadRows []unreadRow
	Check(handler.DB.Model(&models.Message{}).Select("chat_id, COUNT(*) AS n").
		Where("chat_id IN (?) AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", ids, viewer, false).
		Group("chat_id").Scan(&unreadRows).Error)
	var lasts []models.Message
	Check(handler.DB.Where("id IN (?)",
		handler.DB.Model(&models.Message{}).Select("MAX(id)").Where("chat_id IN (?)", ids).Group("chat_id"),
	).Find(&lasts).Error)
	lastByChat := map[int64]*models.Message{}
	bundleMsgs := make([]models.Message, 0, len(lasts))
	for index := range lasts {
		message := lasts[index]
		if message.ChatID != nil {
			lastByChat[*message.ChatID] = &lasts[index]
			bundleMsgs = append(bundleMsgs, message)
		}
	}
	detailByChat := map[int64][]models.Message{}
	if detail {
		for _, comment := range chats {
			msgs := All[models.Message](handler.DB.Where("chat_id = ?", comment.ID).Order("id DESC").Limit(200))
			detailByChat[comment.ID] = msgs
			bundleMsgs = append(bundleMsgs, msgs...)
		}
	}
	bundle := handler.loadMessageBundle(bundleMsgs)
	for _, comment := range chats {
		pre := out[comment.ID]
		for _, id := range participantsByChat[comment.ID] {
			if user, ok := usersByID[id]; ok {
				pre.participants = append(pre.participants, user)
			}
		}
		pre.last = lastByChat[comment.ID]
		pre.detail = detailByChat[comment.ID]
		pre.bundle = bundle
	}
	for _, row := range unreadRows {
		if pre, ok := out[row.ChatID]; ok {
			pre.unread = row.N
		}
	}
	return out
}

func (handler *Handler) ChatJSON(chat models.Chat, user int64, detail bool) map[string]any {
	return handler.chatJSONWith(chat, user, detail, handler.loadChatPreload([]models.Chat{chat}, user, detail)[chat.ID])
}

func (handler *Handler) chatJSONWith(chat models.Chat, user int64, detail bool, pre *chatPreload) map[string]any {
	out := Object(chat)
	participants := []any{}
	out["other_participant"] = nil
	for _, member := range pre.participants {
		participant := map[string]any{"id": member.ID, "username": member.Username, "avatar": member.Avatar}
		participants = append(participants, participant)
		if member.ID != user {
			out["other_participant"] = participant
			if chat.ChatType == "direct" {
				if at := lastSeen(member); !at.IsZero() {
					participant["last_seen_at"] = at
				}
				out["username"] = member.Username
				out["avatar"] = member.Avatar
				if detail {
					out["target_user"] = member.Username
				}
			}
		}
	}
	out["participants"] = participants
	out["participant_count"] = len(participants)
	out["unread_count"] = pre.unread
	if pre.last != nil {
		message := handler.messageJSONWith(*pre.last, pre.bundle)
		message["user"] = message["username"]
		out["last_message"] = message
	}
	if detail {
		messages := []any{}
		for _, message := range pre.detail {
			messages = append(messages, handler.messageJSONWith(message, pre.bundle))
		}
		for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
			messages[left], messages[right] = messages[right], messages[left]
		}
		out["messages"] = messages
	}
	return out
}
