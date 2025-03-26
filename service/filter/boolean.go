package filter

import "polvo/service/model"

type And []Logic

func (logic And) Operation(log *model.CommonLogWrapper) bool {
	if len(logic) == 1 {
		return logic[0].Operation(log)
	} else {
		conv := true
		for _, l := range logic {
			conv = conv && l.Operation(log)
		}
		return conv
	}
}

type Or []Logic

func (logic Or) Operation(log *model.CommonLogWrapper) bool {
	if len(logic) == 1 {
		return logic[0].Operation(log)
	} else {
		conv := false
		for _, e := range logic {
			conv = conv || e.Operation(log)
		}
		return conv
	}
}

type Not struct {
	logic Logic
}

func (logic *Not) Operation(log *model.CommonLogWrapper) bool {
	return !logic.logic.Operation(log)
}
