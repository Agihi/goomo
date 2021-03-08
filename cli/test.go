package main

import (
	"iteragit.iteratec.de/go_loomo_go/goomo"
)

var linearVelocity float32
var angularVelocity float32

func main() {
	//trafficsigns.TestData()
	//trafficsigns.TestGrayScale()
	goomo.TestBezierPath()

	//linearVelocity = 0;
	//angularVelocity = 0;
	//window:=gocv.NewWindow("test")
	//for {
	//	key := window.WaitKey(1)
	//	if key != -1 {
	//		switch key {
	//		case 82:
	//			fmt.Println("up")
	//			linearVelocity += float32(0.2)
	//		case 84:
	//			fmt.Println("down")
	//			linearVelocity -= float32(0.2)
	//		case 81:
	//			fmt.Println("left")
	//			angularVelocity += float32(0.2)
	//		case 83:
	//			angularVelocity -= float32(0.2)
	//			fmt.Println("right")
	//		case 32:
	//			angularVelocity = float32(0)
	//			linearVelocity = float32(0)
	//			fmt.Println("stop all")
	//		}
	//	}
	//}
}
