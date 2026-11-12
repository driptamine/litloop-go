package videos

import (
	"encoding/json"
	. "litloop/backend/internal/httpapi"
)

func (s *Handler) LaunchSpot(q *Request) any {
	q.Admin()
	if s.Config.SpotURL == "" {
		Fail(503, "SPOT_LAUNCH_URL is not configured")
	}
	payload, _ := json.Marshal(q.Body())
	return s.FetchJSON(q, "POST", s.Config.SpotURL, payload, map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + s.Config.TranscriptionSecret})
}
