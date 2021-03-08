package goomo

import (
	"gocv.io/x/gocv"
	"sync"
)

type DataProcessor struct {
	InboundData chan *LoomoData
	OutboundJPG chan JPG
	OutboundMat chan *ManagedMat
}

func (d *DataProcessor) HandleStream(stream *SensorStream, _ chan Command) {
	logger.Debug("DataProcessor started.")
	d.InboundData = stream.Data
	id := int64(0)
	for loomoData := range d.InboundData {
		//logger.Debug("dp")
		mat, err := gocv.IMDecode(loomoData.data, gocv.IMReadColor)
		if err != nil {
			logger.Error("failed to decode image", "error", err)
		}

		managed := (&ManagedMat{
			id:        id,
			timestamp: loomoData.timestamp,
			lock:      &sync.Mutex{},
		}).Init(&mat)

		id++
		managed.Assign()
		select {
		case d.OutboundMat <- managed:

		default:
			//nothing to do
			managed.Done()
		}
		go managed.Finish()

		select {
		case d.OutboundJPG <- JPG(loomoData.data):
		default:
		}
	}
	logger.Debug("DataProcessor stopped.")
}
