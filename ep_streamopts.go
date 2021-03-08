package goomo

import (
	"github.com/gorilla/mux"
	"net/http"
)

type StreamOpts struct {
	Lc *LoomoCommunicator
}

func (so *StreamOpts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	//`log.Printf("got request: %v", *r)
	var err error
	switch mux.Vars(r)["option"] {
	case "start":
		err = so.Lc.ExecuteCommand(&CSSTCommand{"1339", "CAM"})
	case "stop":
		err = so.Lc.ExecuteCommand(&CESTCommand{"1339", "CAM"})
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}
