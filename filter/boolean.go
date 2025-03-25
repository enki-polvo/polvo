package filter

type And[T any] []Logic[T]

func (logic And[T]) operation(log T) bool {
	if len(logic) == 1 {
		return logic[0].operation(log)
	} else {
		conv := false
		for _, l := range logic {
			conv = conv && l.operation(log)
		}
		return conv
	}
}

type Or[T any] []Logic[T]

func (logic Or[T]) operation(log T) bool {
	if len(logic) == 1 {
		return logic[0].operation(log)
	} else {
		conv := false
		for _, e := range logic {
			conv = conv || e.operation(log)
		}
		return conv
	}
}

type Not[T any] struct {
	logic Logic[T]
}

func (logic Not[T]) operation(log T) bool {
	return !logic.logic.operation(log)
}
