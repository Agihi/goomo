package goomo

import (
	"encoding/json"
	"log"
	"net/http"
)

type Motion struct {
	Lc *LoomoCommunicator
}

type MotionRequest struct {
	Type  string
	Value float32
}

func (m *Motion) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//log.Printf("got request: %v", *r)
	var mr MotionRequest
	if r.Body == nil {
		http.Error(w, "Please send a request body", 400)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&mr)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	switch mr.Type {
	case "linear":
		log.Printf("linear: %v", mr.Value)
		err := m.Lc.ExecuteCommand(&CLVLCommand{Lv: mr.Value})
		if err != nil {
			logger.Error(err)
		}
	case "angular":
		log.Printf("angular: %v", mr.Value)
		err := m.Lc.ExecuteCommand(&CAVLCommand{Av: mr.Value})
		if err != nil {
			logger.Error(err)
		}
	}
}
