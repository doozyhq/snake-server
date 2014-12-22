package main

import (
	"errors"
	"io"

	"bitbucket.org/pushkin_ivan/clever-snake/game"
	"github.com/golang/glog"
	"github.com/ivan1993spb/pwshandler"
	"golang.org/x/net/context"
	"golang.org/x/net/websocket"
)

const INPUT_MAX_LENGTH = 512

type PoolFeatures struct {
	// Starts common game pool stream for passed connection
	startStream StartStreamFunc
	startPlayer game.StartPlayerFunc
	// Context of current pool
	poolContext context.Context
}

type errConnProcessing struct {
	err error
}

func (e *errConnProcessing) Error() string {
	return "Error of connection processing in connection manager: " +
		e.err.Error()
}

type ConnManager struct{}

func NewConnManager() pwshandler.ConnManager {
	return &ConnManager{}
}

// Implementing pwshandler.ConnManager interface
func (m *ConnManager) Handle(ws *websocket.Conn,
	data pwshandler.Environment) error {
	if glog.V(INFOLOG_LEVEL_CONNS) {
		glog.Infoln("Websocket handler was started")
		defer glog.Infoln("Websocket handler was finished")
	}

	game, ok := data.(*PoolFeatures)
	if !ok {
		return &errConnProcessing{
			errors.New("Pool data was not received"),
		}
	}

	if glog.V(INFOLOG_LEVEL_CONNS) {
		glog.Infoln("Creating connection to common game stream")
	}
	// Game data which is common for all players in current pool
	if err := game.startStream(ws); err != nil {
		return &errConnProcessing{err}
	}

	cxt, cancel := context.WithCancel(game.poolContext)

	// input is channel for transferring information from client to
	// player goroutine, for example: player commands
	input := make(chan []byte)

	if glog.V(INFOLOG_LEVEL_CONNS) {
		glog.Infoln("Starting player")
	}

	// output is channel for transferring private game information
	// for only one player. This information are useful only for
	// current player
	output, err := game.startPlayer(cxt, input)
	if err != nil {
		return &errConnProcessing{err}
	}

	if glog.V(INFOLOG_LEVEL_CONNS) {
		glog.Infoln("Starting private game stream")
	}
	// Send game data which are useful only for current player
	go func() {
		if glog.V(INFOLOG_LEVEL_CONNS) {
			defer glog.Infoln("Private game stream stops")
		}
		for {
			select {
			case <-cxt.Done():
				return
			case data := <-output:
				if _, err := ws.Write(data); err != nil {
					if glog.V(INFOLOG_LEVEL_CONNS) {
						glog.Errorln(
							"Cannot send private game data:",
							err,
						)
					}
					return
				}
			}
		}
	}()

	if glog.V(INFOLOG_LEVEL_CONNS) {
		glog.Infoln("Starting player listener")
	}
	// Listen for player commands
	go func() {
		if glog.V(INFOLOG_LEVEL_CONNS) {
			defer glog.Infoln("Player listener stops")
		}

		buffer := make([]byte, INPUT_MAX_LENGTH)
		for {
			n, err := ws.Read(buffer)
			if err != nil {
				if err != io.EOF {
					glog.Errorln("Cannot read data:", err)
				}

				if cxt.Err() == nil {
					cancel()
				}

				return
			}
			input <- buffer[:n]
		}
	}()

	<-cxt.Done()

	close(input)

	return nil

}

// Implementing pwshandler.ConnManager interface
func (m *ConnManager) HandleError(_ *websocket.Conn, err error) {
	if err == nil {
		err = errors.New("Passed nil errer for reporting")
	}
	glog.Errorln(err)
}
