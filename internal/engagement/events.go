// Package engagement exports viewing and reaction events to Astra DB.
package engagement

import (
	"litloop/backend/internal/config"
	"litloop/backend/internal/models"
	"time"

	"gorm.io/gorm"
)

const TaskKind = "astra_engagement"

type Event struct {
	ResourceType string    `json:"resource_type"`
	ResourceID   int64     `json:"resource_id"`
	ActorID      int64     `json:"actor_id"`
	Action       string    `json:"action"`
	OccurredAt   time.Time `json:"occurred_at"`
}

func Enabled(c config.Config) bool {
	return c.AstraDatabaseID != "" && c.AstraToken != ""
}

// Enqueue records an event in the same transaction as the view or reaction.
func Enqueue(tx *gorm.DB, c config.Config, resource string, id, actor int64, action string) error {
	if !Enabled(c) {
		return nil
	}
	now := time.Now().UTC()
	event := Event{
		ResourceType: resource,
		ResourceID:   id,
		ActorID:      actor,
		Action:       action,
		OccurredAt:   now,
	}
	return tx.Create(&models.Task{Kind: TaskKind, Payload: models.AsJSON(event), Status: "pending", RunAt: now}).Error
}

func EnqueueReaction(tx *gorm.DB, c config.Config, resource string, id, actor int64, liked bool) error {
	action := "unlike"
	if liked {
		action = "like"
	}
	return Enqueue(tx, c, resource, id, actor, action)
}
