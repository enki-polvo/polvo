package filter

import (
	perror "polvo/error"
	"polvo/service/model"

	"gopkg.in/yaml.v3"
)

type FilterOperator interface {
	Operation(log *model.CommonLogWrapper) bool
}

type filterOperator struct {
	parser    Parser
	filterObj *Filter
	allow     []Logic
	deny      []Logic
}

func NewFilterOperator(filterData []byte) (FilterOperator, error) {
	var err error
	newFilterOP := new(filterOperator)

	newFilterOP.parser, err = NewParser()
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrFilterConstructor,
			Origin: err,
			Msg:    "error while NewFilterOperator",
		}
	}
	// parse Filter object from yaml byte slice
	newFilterOP.filterObj = new(Filter)
	err = yaml.Unmarshal(filterData, newFilterOP.filterObj)
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrFilterConstructor,
			Origin: err,
			Msg:    "error while NewFilterOperator",
		}
	}
	// construct allow selections and deny selections
	// construct allow selections
	for allowSelectionName, allowSellections := range newFilterOP.filterObj.Allow {
		allowSelection, err := NewRuleSelectionOperator(newFilterOP.parser, allowSelectionName, &allowSellections)
		if err != nil {
			return nil, perror.PolvoFilterError{
				Code:   perror.ErrFilterConstructor,
				Origin: err,
				Msg:    "error while NewFilterOperator",
			}
		}
		newFilterOP.allow = append(newFilterOP.allow, allowSelection)
	}
	// construct deny selections
	for denySelectionName, denySellections := range newFilterOP.filterObj.Deny {
		denySelection, err := NewRuleSelectionOperator(newFilterOP.parser, denySelectionName, &denySellections)
		if err != nil {
			return nil, perror.PolvoFilterError{
				Code:   perror.ErrFilterConstructor,
				Origin: err,
				Msg:    "error while NewFilterOperator",
			}
		}
		newFilterOP.deny = append(newFilterOP.deny, denySelection)
	}
	return newFilterOP, nil
}

// FilterOperation works as Follows:
// 1. Check Deny First, then Allow
// 2. If Deny is true, then return false
// 3. If Allow is true, then return true
// 4. If Deny is false and Allow is false, then return false
// 5. If Deny is false and Allow is true, then return true
func (f *filterOperator) Operation(log *model.CommonLogWrapper) bool {
	var (
		denyResult  bool
		allowResult bool
	)
	// check Deny First, then Allow
	denyResult = Or(f.deny).Operation(log)
	allowResult = Or(f.allow).Operation(log)
	return !denyResult || allowResult
}
