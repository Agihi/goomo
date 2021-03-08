package main

import (
	goomo2 "iteragit.iteratec.de/go_loomo_go/goomo"
	"log"
)

func main() {
	lc := goomo2.NewLoomoCommunicator()
	defer lc.Close()
	lc.RegisterHandler("SCAM", &goomo2.VideoMaker{
		Filename: "../../2019-08-02.h264",
	})

	err := lc.Connect()
	if err != nil {
		log.Fatal("connecting to Loomo: ", err)
	}
	err = lc.Start()
	if err != nil {
		log.Fatal("starting Command loop: ", err)
	}
	err = lc.ExecuteCommand(&goomo2.CSSTCommand{"1339", "CAM"})
	if err != nil {
		log.Fatal("executing MJPGStream start: ", err)
	}
	lc.Wait()
}
