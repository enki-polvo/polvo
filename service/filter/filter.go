package filter

import (
	perror "polvo/error"
	"polvo/service/model"

	"gopkg.in/yaml.v3"
)

// # FilterOperator
//
// FilterOperator implements filter operations based on a filter yaml file.
// If the Deny operation is true, the result is false.
type FilterOperator interface {
	Operation(log *model.CommonLogWrapper) bool
}

type filterOperator struct {
	parser    Parser
	filterObj *Filter
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
	// construct deny selections
	for denySelectionName, denySellections := range newFilterOP.filterObj.Deny {
		denySelection, err := NewDenyOperator(newFilterOP.parser, denySelectionName, &denySellections)
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
// 1. If Operation returns true, log will be denied
// 2. If Operation returns false, log will be allowed
// 3. If exception returns true, log will be allowed
func (f *filterOperator) Operation(log *model.CommonLogWrapper) bool {
	// check Deny First, then Allow
	return Or(f.deny).Operation(log)
}

type DenyOperator struct {
	selectionName string
	condition     []Logic
	exception     []Logic
}

func NewDenyOperator(parser Parser, selectionName string, rules *Deny) (Logic, error) {
	dOP := new(DenyOperator)
	dOP.selectionName = selectionName

	// read rules from condition map
	dOP.condition = make([]Logic, 0)
	dOP.exception = make([]Logic, 0)
	for key, val := range rules.Condition {
		rOP, err := NewRuleOperator(parser, key, &val)
		if err != nil {
			return nil, perror.PolvoFilterError{
				Code:   perror.ErrRuleField,
				Msg:    "error while NewDenyOperator",
				Origin: err,
			}
		}
		dOP.condition = append(dOP.condition, rOP)
	}
	// read rules from exception map
	for key, val := range rules.Exception {
		rOP, err := NewRuleOperator(parser, key, &val)
		if err != nil {
			return nil, perror.PolvoFilterError{
				Code:   perror.ErrRuleField,
				Msg:    "error while NewDenyOperator",
				Origin: err,
			}
		}
		dOP.exception = append(dOP.exception, rOP)
	}
	return dOP, nil
}

// Operation works as Follows:
// 1. If Operation returns true, log will be denied
// 2. If Operation returns false, log will be allowed
// 3. If exception returns true, log will be allowed
func (dOP *DenyOperator) Operation(log *model.CommonLogWrapper) bool {
	var (
		denyResult      bool
		exceptionResult bool
	)
	denyResult = Or(dOP.condition).Operation(log)
	exceptionResult = Or(dOP.exception).Operation(log)
	return denyResult && !exceptionResult
}
