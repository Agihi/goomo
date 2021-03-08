package goomo

import (
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"gocv.io/x/gocv"
	"log"
	"net/http"
	"sync"
)

var DebugScreen = false
var logger = func() *zap.SugaredLogger { l, _ := zap.NewDevelopment(); return l.Sugar() }()
var window *gocv.Window

type Goomo struct {
	wg     *sync.WaitGroup
	lc     *LoomoCommunicator
	dp     *DataProcessor
	matMux *MatMultiplexer
	jpgMux *JPGMultiplexer
	pt     *PostitTracker
	tT     *TrafficSignTracker
	ai     *MovementAI
	slam   *MonoSLAM
	vm     *VideoMaker
}

func NewGoomo() *Goomo {
	g := Goomo{}
	g.wg = &sync.WaitGroup{}
	g.lc = NewLoomoCommunicator()
	g.dp = &DataProcessor{
		OutboundJPG: make(chan JPG),
		OutboundMat: make(chan *ManagedMat),
	}
	g.lc.RegisterHandler("SCAM", g.dp)
	g.jpgMux = &JPGMultiplexer{
		Inbound:       g.dp.OutboundJPG,
		outboundMutex: &sync.Mutex{},
		outbounds:     make(map[string]chan JPG),
	}
	g.matMux = &MatMultiplexer{
		Inbound:       g.dp.OutboundMat,
		outboundMutex: &sync.Mutex{},
		outbounds:     make(map[string]chan *ManagedMat),
	}
	return &g
}

func (g *Goomo) Start() {
	cmdsReady := make(chan bool)
	g.wg.Add(1)
	go func() {
		err := g.lc.Connect()
		if err != nil {
			logger.Fatal("connecting to Loomo: ", err)
		}
		err = g.lc.Start()
		if err != nil {
			logger.Fatal("starting Command loop: ", err)
		}
		cmdsReady <- true
		g.lc.Wait()
		g.wg.Done()
		g.lc.Close()
	}()
	<-cmdsReady
	err := g.lc.ExecuteCommand(&CSSTCommand{"1339", "CAM"})
	if err != nil {
		logger.Errorf("starting camera stream: %v", err)
	}
	go g.jpgMux.Multiplex()
	go g.matMux.Multiplex()

	//g.wg.Wait()
}

func (g *Goomo) Wait() {
	g.wg.Wait()
}

func (g *Goomo) ActivateHTTPEndpoints() {
	lc := g.lc

	jpgChan := make(chan JPG)
	g.jpgMux.Add("http", jpgChan)
	stream := NewStream()
	go stream.StartJpgStream(jpgChan)

	motion := &Motion{Lc: lc}
	streamOpts := &StreamOpts{Lc: lc}
	settings := &Settings{g: g}
	downloadVideo := &DownloadVideo{}

	r := mux.NewRouter()
	r.Handle("/stream", stream)
	r.Handle("/stream/{option}", streamOpts)
	r.Handle("/motion", motion)
	r.Handle("/settings", settings)
	r.Handle("/video", downloadVideo)

	originsOk := handlers.AllowedOrigins([]string{"http://localhost:4200"})
	headersOk := handlers.AllowedHeaders([]string{"content-type"})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "PUT", "OPTIONS"})
	go func() {
		err := http.ListenAndServe(":4000", handlers.CORS(originsOk, headersOk, methodsOk)(r))
		if err != nil {
			log.Fatalf("listening mjpeg on port 4000: %v", err)
		}
	}()
}

func (g *Goomo) IsDebugScreenActive() bool {
	return DebugScreen
}

func (g *Goomo) ActivateDebugScreen() {
	if g.IsDebugScreenActive() {
		return
	}

	if window == nil {
		window = gocv.NewWindow("Debug Screen")
	}

	DebugScreen = true
}

func (g *Goomo) DeactivateDebugScreen() {
	DebugScreen = false

	if window != nil {
		window.Close()
		window = nil
	}
}

const postitTrackerMuxId = "pt"

func (g *Goomo) IsPostitAIActive() bool {
	if g.ai == nil || g.pt == nil {
		return false
	}
	if g.ai.InboundPostits == nil || g.pt.Inbound == nil || g.pt.Outbound == nil {
		return false
	}
	if !g.matMux.Has(postitTrackerMuxId) {
		return false
	}
	return true
}

func (g *Goomo) ActivatePostitAI() {
	if g.IsPostitAIActive() {
		return
	}

	mats := make(chan *ManagedMat)
	features := make(chan [][]Feature)

	// init postit tracker
	if g.pt == nil {
		g.pt = &PostitTracker{}
	}
	g.pt.Inbound = mats
	g.pt.Outbound = features

	// init postit ai
	if g.ai == nil {
		g.ai = NewMovementAI(g.lc.Cmds)
	}

	// add to matmux
	g.matMux.Add(postitTrackerMuxId, mats)

	// start go routines
	go g.pt.StartPostitTracker()
	go g.ai.StartPostitAI(g.pt.Outbound)
}

func (g *Goomo) DeactivatePostitAI() {
	// remove from matmux
	g.matMux.Remove(postitTrackerMuxId)

	// deactivate postit tracker
	if g.pt != nil {
		if g.pt.Inbound != nil {
			close(g.pt.Inbound)
		}
		g.pt.Inbound = nil
		g.pt.Outbound = nil
	}

	// deactivate postit ai
	if g.ai != nil {
		if g.ai.InboundPostits != nil {
			close(g.ai.InboundPostits)
		}
		g.ai.InboundPostits = nil
	}
}

const trafficSignTrackerMuxId = "ts"

func (g *Goomo) IsTrafficSignAIActive() bool {
	if g.tT == nil || g.ai == nil {
		return false
	}
	if g.tT.Inbound == nil || g.tT.Outbound == nil || g.ai.InboundTrafficSigns == nil {
		return false
	}
	if !g.matMux.Has(trafficSignTrackerMuxId) {
		return false
	}
	return true
}

func (g *Goomo) ActivateTrafficSignAI() {
	if g.IsTrafficSignAIActive() {
		return
	}

	mats := make(chan *ManagedMat)
	trafficsigns := make(chan *TrafficSignFeature)

	// init traffic sign tracker
	if g.tT == nil {
		g.tT = &TrafficSignTracker{}
	}
	g.tT.Inbound = mats
	g.tT.Outbound = trafficsigns

	// init traffic sign ai
	if g.ai == nil {
		g.ai = NewMovementAI(g.lc.Cmds)
	}

	// add to matmux
	g.matMux.Add(trafficSignTrackerMuxId, g.tT.Inbound)

	// start go routines
	go g.tT.StartTrafficSignTracker()
	go g.ai.StartTrafficSignAI(g.tT.Outbound)
}

func (g *Goomo) DeactivateTrafficSignAI() {
	// remove from matmux
	g.matMux.Remove(trafficSignTrackerMuxId)

	// deactivate traffic sign tacker
	if g.tT != nil {
		if g.tT.Inbound != nil {
			close(g.tT.Inbound)
		}
		g.tT.Inbound = nil
		g.tT.Outbound = nil
	}

	// deactivate traffic sign ai
	if g.ai != nil {
		if g.ai.InboundTrafficSigns != nil {
			close(g.ai.InboundTrafficSigns)
		}
		g.ai.InboundTrafficSigns = nil
	}
}

const videoWriterMuxId = "vw"

func (g *Goomo) IsVideoCaptureRunning() bool {
	if g.vm == nil {
		return false
	}
	if g.vm.Inbound == nil {
		return false
	}
	if !g.matMux.Has(videoWriterMuxId) {
		return false
	}
	return true
}

func (g *Goomo) StartVideoCapture(filename string) {
	if g.IsVideoCaptureRunning() {
		return
	}

	chanMat := make(chan *ManagedMat)
	if g.vm == nil {
		g.vm = NewVideoMaker(chanMat, filename)
	}
	g.vm.Filename = filename
	g.vm.Inbound = chanMat

	g.matMux.Add(videoWriterMuxId, chanMat)

	go g.vm.SaveVideo()
}

func (g *Goomo) StopVideoCapture() {
	if g.vm != nil {
		if g.vm.Inbound != nil {
			close(g.vm.Inbound)
		}
		g.vm.Inbound = nil
	}

	g.matMux.Remove(videoWriterMuxId)
}

const slamMuxId = "slam"

func (g *Goomo) IsSlamActive() bool {
	if g.slam == nil {
		return false
	}
	if g.slam.Inbound == nil {
		return false
	}
	if !g.matMux.Has(slamMuxId) {
		return false
	}
	return true
}

func (g *Goomo) ActivateSlam() {
	if g.IsSlamActive() {
		return
	}

	chanMat := make(chan *ManagedMat)

	// init slam
	if g.slam == nil {
		g.slam = NewMonoSLAM(
			"slam_lib/ORBvoc.bin",
			"slam_lib/settings.yaml",
			true,
			false)
	}

	// add to matmux
	g.matMux.Add(slamMuxId, chanMat)

	// start go routines
	go g.slam.StartSlam(chanMat)
}

func (g *Goomo) DeactivateSlam() {
	// remove from matmux
	g.matMux.Remove(slamMuxId)

	// deactivate slam
	if g.slam != nil {
		if g.slam.Inbound != nil {
			close(g.slam.Inbound)
		}
		g.slam.Inbound = nil
	}

	g.slam = nil
}

func (g *Goomo) TestSlam() {
	chanMat := make(chan *ManagedMat)
	slam := NewMonoSLAM(
		"slam_lib/ORBvoc.bin",
		"slam_lib/settings.yaml",
		true,
		false)
	go slam.StartSlam(chanMat)

	vc, err := gocv.VideoCaptureFile("/home/markus/Videos/test4.h264")
	if err != nil {
		log.Fatal(err)
	}

	mat := gocv.NewMat()
	defer mat.Close()

	id := int64(0)

	for vc.Read(&mat) {
		managed := (&ManagedMat{
			id:        id,
			timestamp: uint64(id * 1000 / 30.0),
			lock:      &sync.Mutex{},
		}).Init(&mat)

		chanMat <- managed
		id++
		managed.Assign()
	}

	slam.Shutdown()
	slam.Close()
}
