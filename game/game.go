package game

import (
	"encoding/json"
	"fmt"
	"time"

	// "bitbucket.org/pushkin_ivan/clever-snake/game/playground"
	"github.com/golang/glog"
	"golang.org/x/net/context"
)

type Game struct {
	cxt context.Context
}

type errStartingGame struct {
	err error
}

func (e *errStartingGame) Error() string {
	return "cannot create game: " + e.err.Error()
}

func NewGame(cxt context.Context, pgW, pgH uint8) (*Game, error) {
	if err := cxt.Err(); err != nil {
		return nil, &errStartingGame{err}
	}

	return &Game{cxt}, nil
}

func (g *Game) StartGame() chan interface{} {
	output := make(chan interface{})

	go func() {
		defer close(output)
		defer glog.Infoln("finishing game")

		// all running objects work like this code:
		for {
			select {
			case <-g.cxt.Done():
				return
			case <-time.Tick(time.Second * 3):
				output <- "test"
			}
		}
	}()

	return output
}

type StartPlayerFunc func(cxt context.Context, input <-chan *Command,
) (<-chan interface{}, error)

func (*Game) StartPlayer(cxt context.Context, input <-chan *Command,
) (<-chan interface{}, error) {
	if err := cxt.Err(); err != nil {
		return nil, fmt.Errorf("cannot start player: %s", err)
	}

	output := make(chan interface{})

	go func() {

		defer close(output)
		defer glog.Infoln("finishing player")

		select {
		case <-cxt.Done():
		case <-time.After(time.Second):
		}

		for {
			select {
			case <-cxt.Done():
				return
			case cmd := <-input:
				if cmd == nil {
					return
				}
				output <- "received cmd =)"
			}
		}
	}()

	return output, nil
}

/////////////////////////////////////////////////////////////////////

type Object json.Marshaler

type ObjectSet map[uint16]Object

type Command struct {
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params"`
}

type Notice struct {
}

// type ObjectContainer map[uint16]interface{}

// func (c *ObjectContainer) MarshalJSON() ([]byte, error) {
// 	return json.Marshal(struct {
// 		Id     uint16      `json:"id"`
// 		Type   string      `json:"type"`
// 		Object interface{} `json:"object"`
// 	}{})
// }

// func (*Game) API_CMD()
