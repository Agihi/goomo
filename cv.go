package goomo

import (
	"gocv.io/x/gocv"
	"gonum.org/v1/plot/vg"
	"image"
	"image/color"
	"log"
)

type Feature struct {
	imagePos    image.Point
	imageBounds image.Rectangle
	realPos     vg.Point
	id          uint64
}

type TrafficSignFeature struct {
	Feature
	Name  string
	Index int64
}

var red = color.RGBA{255, 0, 0, 1}
var green = color.RGBA{0, 255, 0, 1}
var blue = color.RGBA{0, 0, 255, 1}
var white = color.RGBA{255, 255, 255, 1}
var black = gocv.Scalar{0, 0, 0, 1}
var whiteMat = gocv.NewMatWithSizeFromScalar(gocv.NewScalar(255, 255, 255, 1), 480, 640, gocv.MatTypeCV8UC3)
var kernel = gocv.NewMatWithSizeFromScalar(gocv.NewScalar(1, 1, 1, 1), 3, 3, gocv.MatTypeCV8U)

type ColorTracker struct {
	Inbound      chan *ManagedMat
	Outbound     chan [][]Feature
	Descriptions []HSVDescription
}

type PostitTracker struct {
	ColorTracker
}

type TrafficSignTracker struct {
	Inbound  chan *ManagedMat
	Outbound chan *TrafficSignFeature
}

func (tst TrafficSignTracker) StartTrafficSignTracker() {
	logger.Debug("TrafficSignTracker started.")

	ct := ColorTracker{
		Inbound:      make(chan *ManagedMat),
		Outbound:     make(chan [][]Feature),
		Descriptions: NewTrafficSignDescription(),
	}
	go ct.StartColorTracker()

	nn, err := NewTrafficSignNN()
	if err != nil {
		log.Println(err)
		return
	}

	for mat := range tst.Inbound {
		mat.Assign()
		ct.Inbound <- mat
		features := <-ct.Outbound

		for _, featureGroup := range features {
			for _, feature := range featureGroup {
				cropped := mat.mat.Region(feature.imageBounds)
				if cropped.Cols()*cropped.Rows() >= 20*20 {
					dimMat := gocv.NewMat()

					// preprocess 32x32 gray
					gocv.Resize(cropped, &dimMat, size, 0, 0, gocv.InterpolationLinear)
					gocv.CvtColor(dimMat, &dimMat, gocv.ColorBGRToGray)

					// feed into neural net
					trafficsign, certainty, err := nn.PredictWithCertainty(&dimMat)

					if err == nil && feature.imagePos.Y > PixelHorizon {

						if certainty > 0.75 && trafficsign.Name != unknownSign {

							dx, dy := SharedDistanceLookup().Distance(feature.imagePos.X, feature.imagePos.Y)
							feature.realPos = vg.Point{vg.Length(dx), vg.Length(dy)}

							tsf := TrafficSignFeature{
								Feature: feature,
								Name:    trafficsign.Name,
							}

							mat.put(func(mat *gocv.Mat) {
								point := feature.imageBounds.Min
								point.Y -= 10
								gocv.PutText(mat, tsf.Name, point, 0, 0.5, red, 2)
							})
							tst.Outbound <- &tsf
						}
					}

					err = dimMat.Close()
					if err != nil {
						logger.Debug(err)
					}
				}
				cropped.Close()
			}
		}
		mat.Done()
	}
	close(ct.Inbound)
	nn.Close()
	logger.Debug("TrafficSignTracker stopped.")
}

func (pt PostitTracker) StartPostitTracker() {
	logger.Debug("PostitTracker started.")
	pt.StartColorTracker()
	logger.Debug("PostitTracker stopped.")
}

func (ct ColorTracker) StartColorTracker() {
	if ct.Descriptions == nil {
		ct.Descriptions = NewColorTracker()
	}
	inbounds := make([]chan *ManagedMat, len(ct.Descriptions))
	outbounds := make([]chan []Feature, len(ct.Descriptions))
	for i, description := range ct.Descriptions {
		inbounds[i] = make(chan *ManagedMat)
		outbounds[i] = make(chan []Feature)
		go description.findColorFeature(inbounds[i], outbounds[i])
	}
	for mat := range ct.Inbound {
		colorGroups := make([][]Feature, len(ct.Descriptions))
		for i := range ct.Descriptions {
			mat.Assign()
			inbounds[i] <- mat
		}
		for i := range ct.Descriptions {
			colorGroups[i] = <-outbounds[i]
		}
		// TODO: Send on closed channel, when activating / deactivating
		ct.Outbound <- colorGroups
		mat.Done()
	}
	for i := range ct.Descriptions {
		close(inbounds[i])
	}
}

func NewColorTracker() []HSVDescription {
	descriptions := make([]HSVDescription, 2)
	// ORANGE
	descriptions[0] = HSVDescription{
		HSV{
			H: 35,
			S: 40,
			V: 80,
		},
		HSVB{
			HB: 4,
			SB: 20,
			VB: 35, // 20,
		},
	}
	// GREEN
	descriptions[1] = HSVDescription{
		HSV{
			H: 96, //94,
			S: 35,
			V: 80,
		},
		HSVB{
			HB: 10, //4,
			SB: 15, // 10,
			VB: 50, // 30,
		},
	}
	return descriptions
}

func NewTrafficSignDescription() []HSVDescription {
	descriptions := make([]HSVDescription, 1)
	descriptions[0] = HSVDescription{
		HSV{
			H: 343,
			S: 58,
			V: 59,
		},
		HSVB{
			HB: 4,
			SB: 20,
			VB: 50,
		},
	}
	return descriptions
}
