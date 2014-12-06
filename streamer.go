package main

import (
	"errors"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

// Playground wrapper
type Playground interface {
	Pack() string
	Updated() bool
}

type stream struct {
	playground  Playground
	subscribers []*websocket.Conn
}

func newStream(pg Playground, first *websocket.Conn) *stream {
	return &stream{pg, []*websocket.Conn{first}}
}

func (s *stream) addSubscriber(conn *websocket.Conn) {
	if !s.connExists(conn) {
		s.subscribers = append(s.subscribers, conn)
	}
}

func (s *stream) delSubscriber(i int) {
	if i > -1 && len(s.subscribers) > i {
		s.subscribers = append(
			s.subscribers[:i],
			s.subscribers[i+1:]...,
		)
	}
}

func (s *stream) connExists(conn *websocket.Conn) bool {
	return s.connIndex(conn) > -1
}

func (s *stream) connIndex(conn *websocket.Conn) int {
	for i := range s.subscribers {
		if s.subscribers[i] == conn {
			return i
		}
	}
	return -1
}

func (s *stream) push() {
	if s.playground.Updated() {
		data := []byte(s.playground.Pack())
		for i := 0; i < len(s.subscribers); {
			err := s.subscribers[i].WriteMessage(
				websocket.TextMessage,
				data,
			)

			if err != nil {
				s.delSubscriber(i)
				continue
			}

			i++
		}
	}
}

type Streamer struct {
	delay    time.Duration
	streams  []*stream
	pingPong chan chan struct{}
	parCxt   context.Context    // Parent context
	cancel   context.CancelFunc // Cancel func of child context
}

func NewStreamer(cxt context.Context, delay time.Duration,
) (*Streamer, error) {
	if err := cxt.Err(); err != nil {
		return nil, err
	}
	if delay <= 0 {
		return nil, errors.New("Invalid delay")
	}

	return &Streamer{
		delay:    delay,
		streams:  make([]*stream, 0),
		pingPong: make(chan chan struct{}),
		parCxt:   cxt,
	}, nil
}

func (s *Streamer) delStream(i int) {
	if i > -1 && len(s.streams) > i {
		s.streams = append(s.streams[:i], s.streams[i+1:]...)
	}
}

func (s *Streamer) Subscribe(pg Playground, conn *websocket.Conn) {
	if conn == nil {
		return
	}

	defer func() {
		if !s.running() {
			s.start()
		}
	}()

	for _, stream := range s.streams {
		if stream.playground == pg {
			if !stream.connExists(conn) {
				if glog.V(3) {
					glog.Infoln("Creating new subscriber to stream")
				}
				stream.addSubscriber(conn)
			}
			return
		}
	}

	if glog.V(3) {
		glog.Infoln("Creating new subscriber to NEW stream")
	}
	s.streams = append(
		s.streams,
		newStream(pg, conn),
	)
}

func (s *Streamer) Unsubscribe(pg Playground, conn *websocket.Conn) {
	if glog.V(3) {
		glog.Infoln("Unsubscribe connection from stream")
	}

	for i := range s.streams {
		if s.streams[i].playground == pg {
			if glog.V(4) {
				glog.Infoln("Necessary stream was found")
			}

			if j := s.streams[i].connIndex(conn); j > -1 {
				if glog.V(4) {
					glog.Infoln("Subscriber was found")
				}
				if glog.V(3) {
					glog.Infoln("Removing subscriber from stream")
				}
				s.streams[i].delSubscriber(j)
				if len(s.streams[i].subscribers) == 0 {
					if glog.V(3) {
						glog.Infoln(
							"Stream has no subscribers.",
							"Removing stream",
						)
					}
					s.delStream(i)
				}
				if len(s.streams) == 0 && s.running() {
					if glog.V(3) {
						glog.Infoln(
							"Streamer is empty.",
							"Stoping streamer",
						)
					}
					s.stop()
				}
				return
			}
		}
	}
}

func (s *Streamer) start() {
	if !s.running() {
		if glog.V(3) {
			glog.Infoln("Starting streamer")
		}

		var cxt context.Context
		cxt, s.cancel = context.WithCancel(s.parCxt)

		s.run(cxt)
	}
}

func (s *Streamer) stop() {
	if s.running() && s.cancel != nil {
		s.cancel()
	}
}

func (s *Streamer) running() bool {
	var ch = make(chan struct{})
	go func() {
		s.pingPong <- ch
	}()
	select {
	case <-ch:
		return true
	case <-time.After(s.delay):
		<-s.pingPong
	}
	return false
}

func (s *Streamer) run(cxt context.Context) {
	if len(s.streams) == 0 {
		return
	}

	if s.running() {
		return
	}

	go func() {
		var t = time.Tick(s.delay)

		for {
			select {
			case <-cxt.Done():
				if glog.V(2) {
					glog.Infoln(
						"Stopping streamer:",
						"context was canceled",
					)
				}
				return
			case ch := <-s.pingPong:
				ch <- struct{}{}
				continue
			case <-t:
			}

			if len(s.streams) == 0 {
				if glog.V(2) {
					glog.Infoln(
						"Stopping streamer:",
						"there is no one stream",
					)
				}
				return
			}

			for i := 0; i < len(s.streams); {
				if len(s.streams[i].subscribers) == 0 {
					s.delStream(i)
					continue
				}

				if s.streams[i].playground.Updated() {
					s.streams[i].push()
				}

				i++
			}
		}
	}()
}
