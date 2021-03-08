package goomo

import (
	"log"

	"gocv.io/x/gocv"
)

type VideoMaker struct {
	Filename    string
	Inbound     chan *ManagedMat
	videoWriter *gocv.VideoWriter
}

func NewVideoMaker(inbound chan *ManagedMat, filename string) *VideoMaker {
	return &VideoMaker{
		Filename: filename,
		Inbound:  inbound,
	}
}

func (v VideoMaker) SaveVideo() {
	logger.Debug("VideoMaker started.")
	var err error
	v.videoWriter, err = gocv.VideoWriterFile(v.Filename, "MPEG", 30, 640, 480, true)
	if err != nil {
		log.Fatal(err)
		return
	}
	for mat := range v.Inbound {
		img := *mat.mat
		if err != nil {
			return
		}
		err = v.videoWriter.Write(img)
		mat.Done()
	}
	err = v.videoWriter.Close()
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Debug("VideoMaker stopped.")
}
