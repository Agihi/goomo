package goomo

import (
	"errors"
	"fmt"
	"gocv.io/x/gocv"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"sync"
	"time"
)

type MJPGStream struct {
	m        sync.Mutex
	s        map[chan []byte]struct{}
	Interval time.Duration
}

func NewStream() *MJPGStream {
	return &MJPGStream{
		s: make(map[chan []byte]struct{}),
	}
}

func NewStreamWithInterval(interval time.Duration) *MJPGStream {
	return &MJPGStream{
		s:        make(map[chan []byte]struct{}),
		Interval: interval,
	}
}

func (s *MJPGStream) Close() error {
	s.m.Lock()
	defer s.m.Unlock()
	for c := range s.s {
		close(c)
		delete(s.s, c)
	}
	s.s = nil
	return nil
}

func (s *MJPGStream) Update(b []byte) error {
	s.m.Lock()
	defer s.m.Unlock()
	if s.s == nil {
		return errors.New("stream was closed")
	}
	for c := range s.s {
		select {
		case c <- b:
		default:
		}
	}
	return nil
}

func (s *MJPGStream) StartJpgStream(jpgChan chan JPG) {
	for jpg := range jpgChan {
		err := s.Update(jpg)
		if err != nil {
			logger.Error(err)
		}
	}
}

func (s *MJPGStream) StartMatStream(matChan chan *gocv.Mat) {
	for mat := range matChan {
		buf, err := gocv.IMEncode(".jpg", *mat)
		if err != nil {
			logger.Error(err)
			return
		}
		err = s.Update(buf)
		if err != nil {
			logger.Error(err)
			return
		}
	}
}

func (s *MJPGStream) add(c chan []byte) {
	s.m.Lock()
	s.s[c] = struct{}{}
	s.m.Unlock()
}

func (s *MJPGStream) destroy(c chan []byte) {
	s.m.Lock()
	if s.s != nil {
		close(c)
		delete(s.s, c)
	}
	s.m.Unlock()
}

func (s *MJPGStream) NWatch() int {
	return len(s.s)
}

func (s *MJPGStream) Current() []byte {
	c := make(chan []byte)
	s.add(c)
	defer s.destroy(c)

	return <-c
}

func (s *MJPGStream) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c := make(chan []byte)
	s.add(c)
	defer s.destroy(c)

	m := multipart.NewWriter(w)
	defer m.Close()

	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary="+m.Boundary())
	w.Header().Set("Connection", "close")
	h := textproto.MIMEHeader{}
	st := fmt.Sprint(time.Now().Unix())
	for {
		time.Sleep(s.Interval)

		b, ok := <-c
		if !ok {
			break
		}
		h.Set("Content-Type", "image/jpeg")
		h.Set("Content-Length", fmt.Sprint(len(b)))
		h.Set("X-StartTime", st)
		h.Set("X-TimeStamp", fmt.Sprint(time.Now().Unix()))
		mw, err := m.CreatePart(h)
		if err != nil {
			break
		}
		_, err = mw.Write(b)
		if err != nil {
			break
		}
		if flusher, ok := mw.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}
