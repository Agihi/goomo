package goomo

import (
	"gocv.io/x/gocv"
	"net"
	"sync"
)

// JPG is used for byte slices which are only to be interpreted as an JPG formatted image
type JPG []byte

// CommandTag symbolizes the types of Loomo Protocol Messages
type CommandTag uint8

// Possible CommandTags
const (
	CSST CommandTag = iota
	CEST
	CSPS
	CLVL
	CAVL
	CHMD
)

const (
	maxPacketSize = 8192
	headerSize    = 24
	workerThreads = 1
)

type Command interface {
	// Tag returns the CommandTag which is implemented
	Tag() CommandTag
	// MsgFormat returns the bytes of the Command which should be sent over TCP
	MsgFormat() ([]byte, error)
}

// LoomoCommunicator specifies the Main type through which communication with a Loomo should happen
type LoomoCommunicator struct {
	BCport    string
	loomoAddr string
	conn      *net.TCPConn
	done      chan bool
	Cmds      chan Command
	Streams   map[int]*SensorStream
	handlers  map[string]StreamDataHandler
}

type StreamDataHandler interface {
	HandleStream(stream *SensorStream, cmds chan Command)
}

type StreamDataHandlerFunc func(stream *SensorStream, cmds chan Command)

func (s StreamDataHandlerFunc) HandleStream(stream *SensorStream, cmds chan Command) {
	s(stream, cmds)
}

// SensorStream contains all the data and endpoints which are needed when receiving sensoric data from Loomo
type SensorStream struct {
	Data       chan *LoomoData
	unfinished map[uint64]map[uint32][]byte
	packets    chan []byte
	Conn       *net.UDPConn
}

// NewLoomoCommunicator creates a new basic LoomoCommunicator with standard parameters
func NewLoomoCommunicator() *LoomoCommunicator {
	lc := LoomoCommunicator{BCport: ":1336"}
	lc.Cmds = make(chan Command)
	lc.Streams = make(map[int]*SensorStream)
	lc.handlers = make(map[string]StreamDataHandler)
	return &lc
}

func NewSensorStream() *SensorStream {
	s := SensorStream{
		Data:       make(chan *LoomoData),
		unfinished: make(map[uint64]map[uint32][]byte),
		packets:    make(chan []byte),
	}
	return &s
}

type ManagedMat struct {
	id        int64
	mat       *gocv.Mat
	wg        *sync.WaitGroup
	lock      *sync.Mutex
	timestamp uint64
	functions []FinishFunction
}

type FinishFunctionStack struct {
	head *FinishFunctionNode
}

func (m *ManagedMat) put(function FinishFunction) {
	m.lock.Lock()
	m.functions = append(m.functions, function)
	m.lock.Unlock()
}

func (s *ManagedMat) ForEach() {
	for i := len(s.functions) - 1; i >= 0; i-- {
		s.functions[i](s.mat)
	}
}

type FinishFunctionNode struct {
	next   *FinishFunctionNode
	finish FinishFunction
}

type FinishFunction func(mat *gocv.Mat)

func (mm *ManagedMat) Init(mat *gocv.Mat) *ManagedMat {
	mm.mat = mat
	mm.wg = &sync.WaitGroup{}
	mm.functions = make([]FinishFunction, 0, 10)
	return mm
}

func (mm *ManagedMat) Assign() *ManagedMat {
	//logger.Debugf("Assigned usage to mm by caller: %v", MyCaller())
	mm.wg.Add(1)
	return mm
}

func (mm *ManagedMat) Mat() *gocv.Mat {
	return mm.mat
}

func (mm *ManagedMat) Done() *ManagedMat {
	//logger.Debugf("Done with usage of mm by caller: %v", MyCaller())
	mm.wg.Done()
	return mm
}

func (mm *ManagedMat) Finish() {
	mm.wg.Wait()
	if DebugScreen {
		mm.ForEach()
		if DebugScreen {
			window.IMShow(*mm.mat)
			window.WaitKey(1)
		}
	}
	err := mm.mat.Close()
	if err != nil {
		logger.Error(err)
	}
}
