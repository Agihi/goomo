package main

import (
	"iteragit.iteratec.de/go_loomo_go/goomo"
)

func main() {
	g := goomo.NewGoomo()
	//g.Start()
	//g.ActivateHTTPEndpoints()
	g.TestSlam()
	g.Wait()
}
