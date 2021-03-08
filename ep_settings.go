package goomo

import (
	"encoding/json"
	"net/http"
)

type Settings struct {
	g *Goomo
}

const (
	debugScreenStr   = "debug-screen"
	postitAIStr      = "postit-ai"
	trafficsignAIStr = "trafficsign-ai"
	slamStr          = "slam"
	videoCaptureStr  = "video-capture"
)

func (s *Settings) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	response := map[string]bool{}

	switch r.Method {
	case http.MethodGet:
		response[debugScreenStr] = s.g.IsDebugScreenActive()
		response[postitAIStr] = s.g.IsPostitAIActive()
		response[trafficsignAIStr] = s.g.IsTrafficSignAIActive()
		response[slamStr] = s.g.IsSlamActive()
		response[videoCaptureStr] = s.g.IsVideoCaptureRunning()
	case http.MethodPut:
		if r.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}
		var body map[string]bool
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		deactivateAndToggle(&body, &response, debugScreenStr, s.g.ActivateDebugScreen, s.g.DeactivateDebugScreen, slamStr, s.g.DeactivateSlam)
		toggle(&body, &response, postitAIStr, s.g.ActivatePostitAI, s.g.DeactivatePostitAI)
		toggle(&body, &response, trafficsignAIStr, s.g.ActivateTrafficSignAI, s.g.DeactivateTrafficSignAI)
		deactivateAndToggle(&body, &response, slamStr, s.g.ActivateSlam, s.g.DeactivateSlam, debugScreenStr, s.g.DeactivateDebugScreen)
		toggle(&body, &response, videoCaptureStr, s.StartVideoCapture, s.g.StopVideoCapture)
	}

	responseJSON, err := json.Marshal(response)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJSON)
}

func toggle(body *map[string]bool, response *map[string]bool, key string, activate, deactivate func()) {
	b, ok := (*body)[key]
	if ok {
		if b {
			activate()
		} else {
			deactivate()
		}
		(*response)[key] = b
	}
}

func deactivateAndToggle(body *map[string]bool, response *map[string]bool, key string, activate, deactivate func(), otherKey string, deactivateOther func()) {
	b, ok := (*body)[key]
	if ok {
		if b {
			deactivateOther()
			activate()
		} else {
			deactivate()
		}
		(*response)[key] = b
		(*response)[otherKey] = false
	}
}

const videofilePath = "video/tmp.h264"

func (s *Settings) StartVideoCapture() {
	s.g.StartVideoCapture(videofilePath)
}
