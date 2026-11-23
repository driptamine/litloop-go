package users

import (
	"fmt"
	"strings"

	. "litloop/backend/internal/httpapi"
	"litloop/backend/internal/models"
)

// The username admin is the superuser flag with a fallback to the drip
// account, so the panel works even if flags were never granted.
func (handler *Handler) requireUsernameAdmin(req *Request) {
	if req.User == nil {
		Fail(401, "Authentication required")
	}
	if req.User.IsSuperuser {
		return
	}
	if strings.EqualFold(strings.TrimSpace(req.User.Username), "drip") {
		return
	}
	Fail(403, "Forbidden")
}

func validReservedUsername(raw string) (string, bool) {
	username := strings.TrimSpace(raw)
	if username == "" || len(username) > 64 {
		return "", false
	}
	return username, true
}

func (handler *Handler) usernameReserved(username string) bool {
	var count int64
	Check(handler.DB.Model(&models.ReservedUsername{}).Where("username = ?", username).Count(&count).Error)
	return count > 0
}

func (handler *Handler) reservedUsernameJSON(entry models.ReservedUsername) map[string]any {
	out := map[string]any{"id": entry.ID, "username": entry.Username, "assigned_to": nil, "reserved_by": nil}
	if entry.AssignedToID != nil {
		var user models.User
		if handler.DB.First(&user, *entry.AssignedToID).Error == nil {
			out["assigned_to"] = map[string]any{"id": user.ID, "username": user.Username}
		}
	}
	if entry.ReservedByID != nil {
		var user models.User
		if handler.DB.First(&user, *entry.ReservedByID).Error == nil {
			out["reserved_by"] = map[string]any{"id": user.ID, "username": user.Username}
		}
	}
	return out
}

func (handler *Handler) reservedUsernameTaken(username string, exceptUserID int64) bool {
	var count int64
	Check(handler.DB.Model(&models.User{}).Where("username = ? AND id <> ?", username, exceptUserID).Count(&count).Error)
	if count > 0 {
		return true
	}
	var reserved models.ReservedUsername
	if handler.DB.Where("username = ?", username).First(&reserved).Error == nil {
		if reserved.AssignedToID == nil || *reserved.AssignedToID != exceptUserID {
			return true
		}
	}
	return false
}

// ReservedUsernames lists every reserved username with its assignee.
func (handler *Handler) ReservedUsernames(req *Request) any {
	handler.requireUsernameAdmin(req)
	entries := All[models.ReservedUsername](handler.DB.Order("username ASC"))
	out := make([]any, 0, len(entries))
	for _, entry := range entries {
		out = append(out, handler.reservedUsernameJSON(entry))
	}
	return map[string]any{"usernames": out}
}

// ReserveUsernames reserves handles and attaches them to one existing user.
// Body: {usernames: "neo, trinity" | [...], assign_to: "bob"}.
func (handler *Handler) ReserveUsernames(req *Request) any {
	handler.requireUsernameAdmin(req)
	body := req.Body()
	var raw []string
	switch value := body["usernames"].(type) {
	case string:
		raw = strings.FieldsFunc(value, func(ch rune) bool { return ch == ',' || ch == '\n' || ch == ';' })
	case []any:
		for _, item := range value {
			raw = append(raw, Str(item))
		}
	default:
		Fail(400, "usernames is required")
	}
	assignee := strings.TrimSpace(Str(body["assign_to"]))
	if assignee == "" {
		Fail(400, "assign_to is required")
	}
	var target models.User
	if handler.DB.Where("username = ?", assignee).First(&target).Error != nil {
		Fail(404, fmt.Sprintf("User %s not found", assignee))
	}
	reserved := []any{}
	skipped := []any{}
	seen := map[string]bool{}
	for _, name := range raw {
		username, ok := validReservedUsername(name)
		if !ok {
			continue
		}
		if seen[username] {
			skipped = append(skipped, map[string]any{"username": username, "reason": "duplicate"})
			continue
		}
		seen[username] = true
		var taken int64
		Check(handler.DB.Model(&models.User{}).Where("username = ?", username).Count(&taken).Error)
		if taken > 0 && target.Username != username {
			skipped = append(skipped, map[string]any{"username": username, "reason": "taken"})
			continue
		}
		if taken > 0 {
			skipped = append(skipped, map[string]any{"username": username, "reason": "already owned by assignee"})
			continue
		}
		var existing models.ReservedUsername
		if handler.DB.Where("username = ?", username).First(&existing).Error == nil {
			skipped = append(skipped, map[string]any{"username": username, "reason": "already reserved"})
			continue
		}
		entry := models.ReservedUsername{Username: username, AssignedToID: Ptr(target.ID), ReservedByID: Ptr(req.User.ID)}
		Check(handler.DB.Create(&entry).Error)
		reserved = append(reserved, handler.reservedUsernameJSON(entry))
	}
	return map[string]any{"reserved": reserved, "skipped": skipped}
}

// ReleaseUsername frees one reserved handle.
func (handler *Handler) ReleaseUsername(req *Request) any {
	handler.requireUsernameAdmin(req)
	username := strings.TrimSpace(req.Param("username"))
	if username == "" {
		Fail(400, "username is required")
	}
	deleted := handler.DB.Where("username = ?", username).Delete(&models.ReservedUsername{})
	Check(deleted.Error)
	if deleted.RowsAffected == 0 {
		Fail(404, "Not reserved")
	}
	return map[string]any{"status": "released"}
}
