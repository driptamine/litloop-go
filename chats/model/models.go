// Package chats contains the chats domain models and SQL relationships.
package model

import (
	"litloop/backend/internal/types"
	photos "litloop/backend/photos/model"
	tracks "litloop/backend/tracks/model"
	users "litloop/backend/users/model"
	videos "litloop/backend/videos/model"
	"time"
)

type JSON = types.JSON

type Chat struct {
	ID              int64  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ChatType        string `json:"chat_type" gorm:"column:chat_type;size:20;default:direct"`
	Name            string `json:"name" gorm:"column:name;size:20"`
	ImageURL        string `json:"image_url" gorm:"column:image_url;size:400"`
	Description     string `json:"description" gorm:"column:description;size:400"`
	IsSavedMessages bool   `json:"is_saved_messages" gorm:"column:is_saved_messages;default:false"`
}

func (Chat) TableName() string { return "chats_chat" }

type Message struct {
	ID          int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ChatID      *int64      `json:"chat_id" gorm:"column:chat_id;index"`
	UserID      *int64      `json:"user_id" gorm:"column:user_id;index"`
	ReplyTo     *int64      `json:"reply_to" gorm:"column:reply_to;index"`
	Text        string      `json:"text" gorm:"column:text"`
	Attachments JSON        `json:"attachments" gorm:"column:attachments;type:json"`
	IsRead      bool        `json:"is_read" gorm:"column:is_read;default:false"`
	CreatedAt   time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	ChatIDRef   *Chat       `json:"-" gorm:"belongsTo:true;foreignKey:ChatID;references:ID;constraint:OnDelete:SET NULL"`
	UserIDRef   *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:SET NULL"`
}

func (Message) TableName() string { return "chats_message" }

type MessageVideo struct {
	ID           int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MessageID    *int64        `json:"message_id" gorm:"column:message_id;index"`
	VideoID      *int64        `json:"video_id" gorm:"column:video_id;index"`
	MessageIDRef *Message      `json:"-" gorm:"belongsTo:true;foreignKey:MessageID;references:ID;constraint:OnDelete:SET NULL"`
	VideoIDRef   *videos.Video `json:"-" gorm:"belongsTo:true;foreignKey:VideoID;references:ID;constraint:OnDelete:SET NULL"`
}

func (MessageVideo) TableName() string { return "chats_messagevideo" }

type MessagePhoto struct {
	ID           int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MessageID    *int64        `json:"message_id" gorm:"column:message_id;index"`
	PhotoID      *int64        `json:"photo_id" gorm:"column:photo_id;index"`
	MessageIDRef *Message      `json:"-" gorm:"belongsTo:true;foreignKey:MessageID;references:ID;constraint:OnDelete:SET NULL"`
	PhotoIDRef   *photos.Photo `json:"-" gorm:"belongsTo:true;foreignKey:PhotoID;references:ID;constraint:OnDelete:SET NULL"`
}

func (MessagePhoto) TableName() string { return "chats_messagephoto" }

type MessageTrack struct {
	ID           int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MessageID    *int64        `json:"message_id" gorm:"column:message_id;index"`
	TrackID      *int64        `json:"track_id" gorm:"column:track_id;index"`
	MessageIDRef *Message      `json:"-" gorm:"belongsTo:true;foreignKey:MessageID;references:ID;constraint:OnDelete:SET NULL"`
	TrackIDRef   *tracks.Track `json:"-" gorm:"belongsTo:true;foreignKey:TrackID;references:ID;constraint:OnDelete:SET NULL"`
}

func (MessageTrack) TableName() string { return "chats_messagetrack" }

type MessageVoice struct {
	ID                int64         `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	MessageID         *int64        `json:"message_id" gorm:"column:message_id;index"`
	VoiceMessageID    *int64        `json:"voice_message_id" gorm:"column:voice_message_id;index"`
	MessageIDRef      *Message      `json:"-" gorm:"belongsTo:true;foreignKey:MessageID;references:ID;constraint:OnDelete:SET NULL"`
	VoiceMessageIDRef *VoiceMessage `json:"-" gorm:"belongsTo:true;foreignKey:VoiceMessageID;references:ID;constraint:OnDelete:SET NULL"`
}

func (MessageVoice) TableName() string { return "chats_messagevoice" }

type VoiceMessage struct {
	ID                    int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	S3Key                 string      `json:"s3_key" gorm:"column:s3_key;size:400"`
	GCSKey                string      `json:"gcs_key" gorm:"column:gcs_key;size:400"`
	R2Key                 string      `json:"r2_key" gorm:"column:r2_key;size:400"`
	Filename              string      `json:"filename" gorm:"column:filename;size:400"`
	Duration              float64     `json:"duration" gorm:"column:duration;default:0.0"`
	UserID                *int64      `json:"user_id" gorm:"column:user_id;index"`
	CreatedAt             time.Time   `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	Transcription         string      `json:"transcription" gorm:"column:transcription"`
	TranscriptionLanguage string      `json:"transcription_language" gorm:"column:transcription_language;size:10"`
	TranscribedAt         *time.Time  `json:"transcribed_at" gorm:"column:transcribed_at"`
	TranscriptionStatus   string      `json:"transcription_status" gorm:"column:transcription_status;size:20;default:not_requested"`
	UserIDRef             *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (VoiceMessage) TableName() string { return "chats_voicemessage" }

type Call struct {
	ID            int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	ChatID        int64       `json:"chat_id" gorm:"column:chat_id;index"`
	CallerID      int64       `json:"caller_id" gorm:"column:caller_id;index"`
	ReceiverID    *int64      `json:"receiver_id" gorm:"column:receiver_id;index"`
	CallType      string      `json:"call_type" gorm:"column:call_type;size:10;default:voice"`
	Status        string      `json:"status" gorm:"column:status;size:20;default:initiated"`
	StartedAt     time.Time   `json:"started_at" gorm:"column:started_at;autoCreateTime"`
	EndedAt       *time.Time  `json:"ended_at" gorm:"column:ended_at"`
	ChatIDRef     *Chat       `json:"-" gorm:"belongsTo:true;foreignKey:ChatID;references:ID;constraint:OnDelete:CASCADE"`
	CallerIDRef   *users.User `json:"-" gorm:"belongsTo:true;foreignKey:CallerID;references:ID;constraint:OnDelete:CASCADE"`
	ReceiverIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:ReceiverID;references:ID;constraint:OnDelete:CASCADE"`
}

func (Call) TableName() string { return "chats_call" }

type CallParticipant struct {
	CallID    int64       `gorm:"column:call_id;primaryKey;autoIncrement:false"`
	UserID    int64       `gorm:"column:user_id;primaryKey;autoIncrement:false"`
	SessionID string      `gorm:"column:session_id;size:80;not null"`
	SeenAt    time.Time   `gorm:"column:seen_at;not null"`
	Call      *Call       `gorm:"foreignKey:CallID;constraint:OnDelete:CASCADE"`
	User      *users.User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

func (CallParticipant) TableName() string { return "chats_call_participant" }

type ChatParticipant struct {
	ID        int64       `json:"id" gorm:"primaryKey"`
	ChatID    int64       `json:"chat_id" gorm:"column:chat_id;uniqueIndex:uq_chats_chat_participants;index"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;uniqueIndex:uq_chats_chat_participants;index"`
	ChatIDRef *Chat       `json:"-" gorm:"belongsTo:true;foreignKey:ChatID;references:ID;constraint:OnDelete:CASCADE"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ChatParticipant) TableName() string { return "chats_chat_participants" }

type ChatTab struct {
	ID        int64       `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	UserID    int64       `json:"user_id" gorm:"column:user_id;index"`
	Emoji     string      `json:"emoji" gorm:"column:emoji;size:16"`
	Position  int         `json:"position" gorm:"column:position;default:0"`
	ChatIDs   JSON        `json:"chat_ids" gorm:"column:chat_ids;type:json"`
	UserIDRef *users.User `json:"-" gorm:"belongsTo:true;foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"`
}

func (ChatTab) TableName() string { return "chats_chattab" }
