package player

import (
	"github.com/sirupsen/logrus"

	"github.com/ivan1993spb/snake-server/game"
	"github.com/ivan1993spb/snake-server/objects/snake"
)

type Player struct {
	game   *game.Game
	logger logrus.FieldLogger
}

func NewPlayer(logger logrus.FieldLogger, game *game.Game) *Player {
	return &Player{
		game:   game,
		logger: logger,
	}
}

func (p *Player) Start(stop <-chan struct{}) {
	s, _ := snake.NewSnake(p.game.World())
	// TODO: Pass stop channel?
	s.Run(stop)

	go func() {
		<-stop
		s.Die()
	}()
}
