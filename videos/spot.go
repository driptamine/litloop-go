package videos

import (
	"encoding/json"
	. "litloop/backend/internal/httpapi"
)

func (handler *Handler) LaunchSpot(req *Request) any {
	req.Admin()
	if handler.Config.SpotURL == "" {
		Fail(503, "SPOT_LAUNCH_URL is not configured")
	}
	payload, _ := json.Marshal(req.Body())
	return handler.FetchJSON(req, "POST", handler.Config.SpotURL, payload, map[string]string{"Content-Type": "application/json", "Authorization": "Bearer " + handler.Config.TranscriptionSecret})
}
