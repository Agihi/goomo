package goomo

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
)

type PositionedObject struct {
	X          float32 `json:"x"`
	Y          float32 `json:"z"`
	ObjectType string  `json:"type"`
}

type PositionedObjects []PositionedObject

type FeatureWebsocket struct {
	Features     chan [][]Feature
	TrafficSigns chan TrafficSign
	upgrader     websocket.Upgrader
}

func NewPositionWebsocket() FeatureWebsocket {
	pws := FeatureWebsocket{
		Features: make(chan [][]Feature),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
	return pws
}

func (pws FeatureWebsocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pws.upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	ws, err := pws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(err)
		return
	}
	log.Println("Websocket connection established")
	pws.communicate(ws)
}

func (pws FeatureWebsocket) communicate(conn *websocket.Conn) {
	trafficSignSet := false
	var currentTS TrafficSign
	for {
		select {
		case ts, ok := <-pws.TrafficSigns:
			log.Printf("got trafficsign: %v", ts)
			if !ok {
				return
			}
			trafficSignSet = true
			currentTS = ts
		case pgs, ok := <-pws.Features:
			if !ok {
				return
			}
			pos := PostitsToPositionedObjects(pgs)
			if trafficSignSet {
				log.Printf("a traffic sign was set")
				pos = append(pos, TrafficSignToPositionedObject(currentTS))
			}
			toSend, err := json.Marshal(pos)
			if err != nil {
				log.Printf("%v could not be marshalled: %v", pos, err)
				continue
			}
			//log.Printf("websocket will send: %v", toSend)
			err = conn.WriteMessage(websocket.TextMessage, toSend)
			if err != nil {
				log.Printf("%v could not be sent: %v", pos, err)
				return
			}
		}
	}
}

func PostitsToPositionedObjects(postits [][]Feature) PositionedObjects {
	pos := make(PositionedObjects, 0, 15)
	for i, p := range postits {
		var objectType string
		if i == 0 {
			objectType = "orange"
		} else if i == 1 {
			objectType = "green"
		}
		for _, postit := range p {
			dx, dy := SharedDistanceLookup().Distance(postit.imagePos.X, postit.imagePos.Y)
			po := PositionedObject{
				X:          float32(dx),
				Y:          float32(dy),
				ObjectType: objectType,
			}
			pos = append(pos, po)
		}
	}
	return pos
}

func TrafficSignToPositionedObject(ts TrafficSign) PositionedObject {
	po := PositionedObject{
		X:          float32(ts.Position.X),
		Y:          float32(ts.Position.Y),
		ObjectType: ts.Name,
	}
	return po
}
