package goomo

//lc.go is for defining methods of the LoomoCommunicator

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type LoomoData struct {
	timestamp uint64
	data      []byte
}

func (l *LoomoCommunicator) receiveAddr() error {
	c, err := net.ListenPacket("udp", l.BCport)
	if err != nil {
		return fmt.Errorf("listening for broadcast on Port %s: %v", l.BCport, err)
	}
	defer c.Close()

	buffer := make([]byte, 1024)
	n, addr, err := c.ReadFrom(buffer)
	if err != nil {
		return fmt.Errorf("reading broadcast from Port %s: %v", l.BCport, err)
	}
	port := strings.TrimSuffix(string(buffer[:n]), "\n")
	// this is dangerous because it only considers ipv4 addresses
	ip := strings.Split(addr.String(), ":")[0]
	l.loomoAddr = strings.Join([]string{ip, port}, ":")
	return nil
}

func (l *LoomoCommunicator) RegisterHandler(tag string, handler StreamDataHandler) (ok bool) {
	l.handlers[tag] = handler
	return true
}

func (l *LoomoCommunicator) Connect() error {
	logger.Debug("Connecting to Loomo...")
	err := l.receiveAddr()
	if err != nil {
		return fmt.Errorf("receiving Loomo address: %v", err)
	}
	tcpAddr, err := net.ResolveTCPAddr("tcp", l.loomoAddr)
	if err != nil {
		return fmt.Errorf("resolving TCP address '%s': %v", l.loomoAddr, err)
	}
	l.conn, err = net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return fmt.Errorf("dialling TCP '%v': %v", tcpAddr, err)
	}
	logger.Debugf("Connected to Loomo: %v", l.conn)
	return nil
}

func (l *LoomoCommunicator) IsConnected() bool {
	return l.conn != nil
}

func (l *LoomoCommunicator) Close() error {
	return l.conn.Close()
}

func (l *LoomoCommunicator) ExecuteCommand(cmd Command) error {
	l.Cmds <- cmd
	return nil
}

func (l *LoomoCommunicator) packetWorker(id, port int) {
	for p := range l.Streams[port].packets {
		_, seq, tval, start, end := sensorHeaderHandler(p[:headerSize])
		if l.Streams[port].unfinished[tval] == nil {
			l.Streams[port].unfinished[tval] = make(map[uint32][]byte)
		}
		l.Streams[port].unfinished[tval][seq-start] = p[headerSize:]
		if uint32(len(l.Streams[port].unfinished[tval])) == end-start+1 {
			data := flattenData(l.Streams[port].unfinished[tval])
			delete(l.Streams[port].unfinished, tval)
			l.Streams[port].Data <- &LoomoData{
				timestamp: tval,
				data:      data,
			}
		}
	}
}

func (l *LoomoCommunicator) sensorHandler(port int, tag string) {
	l.Streams[port] = NewSensorStream()
	defer l.Streams[port].Close()

	for w := 1; w <= workerThreads; w++ {
		go l.packetWorker(w, port)
	}
	go l.handlers[tag].HandleStream(l.Streams[port], l.Cmds)

	var err error
	l.Streams[port].Conn, err = net.ListenUDP("udp", &net.UDPAddr{IP: []byte{0, 0, 0, 0}, Port: port, Zone: ""})
	if err != nil {
		logger.Error("Listening UDP failed: %v", err.Error())
	}
	defer l.Streams[port].Conn.Close()
	for {
		buf := make([]byte, maxPacketSize)
		n, _, err := l.Streams[port].Conn.ReadFromUDP(buf)
		if err != nil {
			logger.Error("Reading UDP failed:", err.Error())
			break
		}
		l.Streams[port].packets <- buf[0:n]
	}
}

func (s *SensorStream) Close() {
	close(s.Data)
	s.unfinished = nil
	close(s.packets)
}

func sensorHeaderHandler(header []byte) (tag string, seq uint32, tval uint64, start uint32, end uint32) {
	tag = string(header[0:4])
	seq = binary.BigEndian.Uint32(header[4:8])
	tval = binary.BigEndian.Uint64(header[8:16])
	start = binary.BigEndian.Uint32(header[16:20])
	end = binary.BigEndian.Uint32(header[20:24])
	return
}

func (l *LoomoCommunicator) Start() error {
	logger.Debug("Starting to take Loomo Commands")
	go func() {
		for cmd := range l.Cmds {
			//log.Printf("Received Command %v", cmd)
			msg, err := cmd.MsgFormat()
			if err != nil {
				logger.Error("cmd has error: %v", err)
			}
			switch cmd.Tag() {
			case CSST:
				port := int(binary.BigEndian.Uint32(msg[8:12]))
				tag := "S" + string(msg[12:15])
				go l.sensorHandler(port, tag)
			case CEST:
				port := int(binary.BigEndian.Uint32(msg[8:12]))
				l.Streams[port].Conn.Close()
			}
			//log.Printf("writing: %x", msg)
			_, err = l.conn.Write(msg)
			//log.Println("Wrote", n, "bytes")
			if err != nil {
				logger.Error("writing to connection: %v", err)
				continue
			}
		}
		l.done <- true
	}()
	logger.Debug("StartedLoomo Commands")
	return nil
}

func (l *LoomoCommunicator) Wait() {
	<-l.done
}
