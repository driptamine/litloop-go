package chats

import (
	"encoding/json"
	"strings"

	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
	"litloop/backend/internal/storage"
)

func messageAttachments(m models.Message) []any {
	var items []any
	if len(m.Attachments) == 0 {
		return nil
	}
	if err := json.Unmarshal(m.Attachments, &items); err != nil {
		return nil
	}
	return items
}

// messageBundle preloads everything message serialization needs so a chat
// with history costs a constant number of queries instead of several per
// message. Without it, opening a conversation runs hundreds of sequential
// queries and the request outlives the client's timeout.
type messageBundle struct {
	users  map[int64]models.User
	voices map[int64]models.VoiceMessage
	owned  map[r2Ownership]bool
}

type r2Ownership struct {
	userID int64
	kind   string
	key    string
}

func (s *Handler) loadMessageBundle(msgs []models.Message) *messageBundle {
	b := &messageBundle{
		users:  map[int64]models.User{},
		voices: map[int64]models.VoiceMessage{},
		owned:  map[r2Ownership]bool{},
	}
	if len(msgs) == 0 {
		return b
	}
	userIDs := make([]int64, 0, len(msgs))
	seenUsers := map[int64]bool{}
	msgIDs := make([]int64, 0, len(msgs))
	for _, m := range msgs {
		msgIDs = append(msgIDs, m.ID)
		if m.UserID != nil && !seenUsers[*m.UserID] {
			seenUsers[*m.UserID] = true
			userIDs = append(userIDs, *m.UserID)
		}
	}
	if len(userIDs) > 0 {
		for _, u := range All[models.User](s.DB.Where("id IN ?", userIDs)) {
			b.users[u.ID] = u
		}
	}
	links := All[models.MessageVoice](s.DB.Where("message_id IN ?", msgIDs))
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
		for _, v := range All[models.VoiceMessage](s.DB.Where("id IN ?", voiceIDs)) {
			byID[v.ID] = v
		}
		for _, link := range links {
			if link.MessageID != nil && link.VoiceMessageID != nil {
				if v, ok := byID[*link.VoiceMessageID]; ok {
					b.voices[*link.MessageID] = v
				}
			}
		}
	}
	b.owned = s.loadR2Ownership(msgs)
	return b
}

// loadR2Ownership resolves, with at most one query per media kind, which of
// the stored attachment keys belong to each message author. Attachments that
// need no database check (chat-upload filenames, empty keys, legacy signed
// URLs) never reach a query.
func (s *Handler) loadR2Ownership(msgs []models.Message) map[r2Ownership]bool {
	owned := map[r2Ownership]bool{}
	keysByKind := map[string]map[string]bool{}
	refs := []r2Ownership{}
	seen := map[r2Ownership]bool{}
	for _, m := range msgs {
		if m.UserID == nil {
			continue
		}
		for _, raw := range messageAttachments(m) {
			item, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			key := Str(item["r2_key"])
			if key == "" {
				key = s.r2AttachmentKey(Str(item["url"]))
			}
			if key == "" {
				continue
			}
			ref := r2Ownership{userID: *m.UserID, kind: Str(item["type"]), key: key}
			if name, found := strings.CutPrefix(key, storage.ChatAttachmentPrefix(*m.UserID)); found && chatAttachmentFilename(name) {
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
		UserID *int64
		R2Key  string
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
		Check(s.DB.Model(model).Select("user_id, r2_key").Where("r2_key IN ?", keys).Scan(&rows).Error)
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

func (s *Handler) VoiceJSON(v models.VoiceMessage) any {
	o := Object(v)
	o["url"] = s.MediaURL(v.R2Key, v.GCSKey, v.S3Key)
	return o
}

func (s *Handler) messageJSONWith(m models.Message, b *messageBundle) map[string]any {
	o := Object(m)
	if attachments, ok := o["attachments"].([]any); ok {
		s.setAttachmentURLs(attachments, m, b.owned)
	}
	o["username"] = "AI"
	o["avatar"] = nil
	o["voice_message"] = nil
	if m.UserID != nil {
		if u, ok := b.users[*m.UserID]; ok {
			o["username"] = u.Username
			o["avatar"] = u.Avatar
		}
	}
	if v, ok := b.voices[m.ID]; ok {
		o["voice_message"] = s.VoiceJSON(v)
	}
	return o
}

func (s *Handler) MessageJSON(m models.Message) map[string]any {
	return s.messageJSONWith(m, s.loadMessageBundle([]models.Message{m}))
}

func (s *Handler) messagesJSON(msgs []models.Message) []any {
	b := s.loadMessageBundle(msgs)
	out := make([]any, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, s.messageJSONWith(m, b))
	}
	return out
}

// chatPreload carries everything ChatJSON needs for one conversation so the
// chat list can fetch shared data for all conversations up front.
type chatPreload struct {
	participants []models.User
	unread       int64
	last         *models.Message
	detail       []models.Message
	bundle       *messageBundle
}

func (s *Handler) loadChatPreload(chats []models.Chat, viewer int64, detail bool) map[int64]*chatPreload {
	out := make(map[int64]*chatPreload, len(chats))
	for _, c := range chats {
		out[c.ID] = &chatPreload{}
	}
	if len(chats) == 0 {
		return out
	}
	ids := make([]int64, 0, len(chats))
	for _, c := range chats {
		ids = append(ids, c.ID)
	}
	parts := All[models.ChatParticipant](s.DB.Where("chat_id IN ?", ids))
	participantsByChat := map[int64][]int64{}
	userIDs := []int64{}
	seenUsers := map[int64]bool{}
	for _, p := range parts {
		participantsByChat[p.ChatID] = append(participantsByChat[p.ChatID], p.UserID)
		if !seenUsers[p.UserID] {
			seenUsers[p.UserID] = true
			userIDs = append(userIDs, p.UserID)
		}
	}
	usersByID := map[int64]models.User{}
	if len(userIDs) > 0 {
		for _, u := range All[models.User](s.DB.Where("id IN ?", userIDs)) {
			usersByID[u.ID] = u
		}
	}
	type unreadRow struct {
		ChatID int64
		N      int64
	}
	var unreadRows []unreadRow
	Check(s.DB.Model(&models.Message{}).Select("chat_id, COUNT(*) AS n").
		Where("chat_id IN (?) AND (user_id <> ? OR user_id IS NULL) AND is_read = ?", ids, viewer, false).
		Group("chat_id").Scan(&unreadRows).Error)
	var lasts []models.Message
	Check(s.DB.Where("id IN (?)",
		s.DB.Model(&models.Message{}).Select("MAX(id)").Where("chat_id IN (?)", ids).Group("chat_id"),
	).Find(&lasts).Error)
	lastByChat := map[int64]*models.Message{}
	bundleMsgs := make([]models.Message, 0, len(lasts))
	for i := range lasts {
		m := lasts[i]
		if m.ChatID != nil {
			lastByChat[*m.ChatID] = &lasts[i]
			bundleMsgs = append(bundleMsgs, m)
		}
	}
	detailByChat := map[int64][]models.Message{}
	if detail {
		for _, c := range chats {
			msgs := All[models.Message](s.DB.Where("chat_id = ?", c.ID).Order("id DESC").Limit(200))
			detailByChat[c.ID] = msgs
			bundleMsgs = append(bundleMsgs, msgs...)
		}
	}
	bundle := s.loadMessageBundle(bundleMsgs)
	for _, c := range chats {
		pre := out[c.ID]
		for _, id := range participantsByChat[c.ID] {
			if u, ok := usersByID[id]; ok {
				pre.participants = append(pre.participants, u)
			}
		}
		pre.last = lastByChat[c.ID]
		pre.detail = detailByChat[c.ID]
		pre.bundle = bundle
	}
	for _, row := range unreadRows {
		if pre, ok := out[row.ChatID]; ok {
			pre.unread = row.N
		}
	}
	return out
}

func (s *Handler) ChatJSON(c models.Chat, user int64, detail bool) map[string]any {
	return s.chatJSONWith(c, user, detail, s.loadChatPreload([]models.Chat{c}, user, detail)[c.ID])
}

func (s *Handler) chatJSONWith(c models.Chat, user int64, detail bool, pre *chatPreload) map[string]any {
	o := Object(c)
	participants := []any{}
	o["other_participant"] = nil
	for _, u := range pre.participants {
		p := map[string]any{"id": u.ID, "username": u.Username, "avatar": u.Avatar}
		participants = append(participants, p)
		if u.ID != user {
			o["other_participant"] = p
			if c.ChatType == "direct" {
				if at := lastSeen(u); !at.IsZero() {
					p["last_seen_at"] = at
				}
				o["username"] = u.Username
				o["avatar"] = u.Avatar
				if detail {
					o["target_user"] = u.Username
				}
			}
		}
	}
	o["participants"] = participants
	o["participant_count"] = len(participants)
	o["unread_count"] = pre.unread
	if pre.last != nil {
		message := s.messageJSONWith(*pre.last, pre.bundle)
		message["user"] = message["username"]
		o["last_message"] = message
	}
	if detail {
		messages := []any{}
		for _, m := range pre.detail {
			messages = append(messages, s.messageJSONWith(m, pre.bundle))
		}
		for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
			messages[i], messages[j] = messages[j], messages[i]
		}
		o["messages"] = messages
	}
	return o
}
