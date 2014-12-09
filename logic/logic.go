package logic

import (
	"errors"

	"bitbucket.org/pushkin_ivan/clever-snake/playground"
)

const _STRENGTH_FACTOR float32 = 1.3

var (
	ErrRecognizingObject = errors.New("Cannot recognize object")
	ErrExecuteCommand    = errors.New("Cannot execute command")
)

// Object game characteristics
type (
	Object interface{}

	Resistant interface {
		Object
		Strength() float32
	}

	Living interface {
		Object
		Resistant

		// Every living thing ever dies
		Die()
		// Living things are hungry
		Feed(int8)
	}

	Notalive interface {
		Object
		Break(*playground.Dot)
	}

	Food interface {
		Object
		// Nutritional value
		NutritionalValue(*playground.Dot) int8
	}

	Controlled interface {
		Object
		Command(string) error
	}
)

// Clash implements logic of clash of two objects (first and second)
// in dot dot
func Clash(first Living, second Object, dot *playground.Dot) error {
	switch second := second.(type) {
	case Food:
		// Feed if second object is food
		first.Feed(second.NutritionalValue(dot))

	case Resistant:
		if first.Strength() > second.Strength()*_STRENGTH_FACTOR {
			switch second := second.(type) {
			case Living:
				// Living dies
				second.Die()

			case Notalive:
				// Not living breaks
				second.Break(dot)

			default:
				goto cannot_recognize_object

			}
		} else {
			// Clash with any hard object will result to dying
			first.Die()
		}

	default:
		goto cannot_recognize_object
	}

	return nil

cannot_recognize_object:

	return ErrRecognizingObject
}
