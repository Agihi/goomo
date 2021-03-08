package main

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"iteragit.iteratec.de/go_loomo_go/goomo"
	"log"
	"net/http"
)

func main() {
	muxer := goomo.NewDataStreamMux()
	stream := goomo.NewStream()
	matMuxer := goomo.NewMatMuxer()

	muxer.RegisterHandler(matMuxer)
	muxer.RegisterHandler(stream)

	pws := goomo.NewPositionWebsocket()

	descriptions := goomo.NewColorTracker(pws.Features)
	matMuxer.RegisterHandler(descriptions)
	signDescription := goomo.NewTrafficSignDescription(pws.TrafficSigns)
	matMuxer.RegisterHandler(signDescription)

	lc := goomo.NewLoomoCommunicator()
	lc.RegisterHandler("SCAM", muxer)
	motion := &goomo.Motion{Lc: lc}
	streamOpts := &goomo.StreamOpts{Lc: lc}

	// init AI
	ai := goomo.InitMovementAI(lc.Cmds)
	go ai.BigBrainAI()

	r := mux.NewRouter()
	r.Handle("/stream", stream)
	r.Handle("/stream/{option}", streamOpts)
	r.Handle("/motion", motion)
	r.Handle("/ws", pws)

	cmdsReady := make(chan bool)
	go setupComm(lc, cmdsReady)
	<-cmdsReady
	log.Println("commands ready")

	originsOk := handlers.AllowedOrigins([]string{"http://localhost:4200"})
	headersOk := handlers.AllowedHeaders([]string{"content-type"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "PUT", "OPTIONS"})
	err := http.ListenAndServe(":4000", handlers.CORS(originsOk, headersOk, methodsOk)(r))
	if err != nil {
		log.Fatalf("listening mjpeg on port 4000: %v", err)
	}
}

func setupComm(lc *goomo.LoomoCommunicator, cmdsReady chan<- bool) {

	err := lc.Connect()
	if err != nil {
		log.Fatal("connecting to Loomo: ", err)
	}
	err = lc.Start()
	if err != nil {
		log.Fatal("starting Command loop: ", err)
	}
	cmdsReady <- true
	lc.Wait()
	lc.Close()
}
