package filter

import (
	"fmt"
	perror "polvo/error"
	"polvo/service/model"
	"strings"

	"github.com/alecthomas/participle/v2"
	"gopkg.in/yaml.v3"
)

// # Logic
//
// Logic is an abstraction of the logical operations performed in filters,
// allowing various logical operations to be handled with a single interface.
// You can see the usage philosophy right in Boolean.go
type Logic interface {
	Operation(log *model.CommonLogWrapper) bool
}

// # RuleOpCode
//
// RuleOpCode is an enumeration of the possible operations that can be performed in a rule.
// The operations are:
// - STARTSWITH : checks if the field starts with the condition
// - ENDSWITH : checks if the field ends with the condition
// - CONTAINS : checks if the field contains the condition
type RuleOpCode int

const (
	STARTSWITH RuleOpCode = iota
	ENDSWITH
	CONTAINS
	NONE
)

// # RuleOperator
//
// RuleOperator implements the actual behavior of the Rule written by the user.
// A rule consists of N events.
type RuleOperator struct {
	op        RuleOpCode
	isAnd     bool
	fieldName string
	events    []Logic
}

func NewRuleOperator(parser Parser, key string, val *yaml.Node) (Logic, error) {
	field, err := ParseRuleField(parser.RuleFieldParser(), key)
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrRuleField,
			Msg:    "error while NewRuleOperate",
			Origin: err,
		}
	}
	// set the operation
	rOP := new(RuleOperator)
	if field.Op != nil {
		// set opcode
		switch *field.Op {
		case "startswith":
			rOP.op = STARTSWITH
		case "endswith":
			rOP.op = ENDSWITH
		case "contains":
			rOP.op = CONTAINS
		default:
			return nil, perror.PolvoFilterError{
				Code:   perror.ErrRuleField,
				Msg:    "error while NewRuleOperate",
				Origin: fmt.Errorf("invalid detection operation"),
			}
		}
	} else {
		rOP.op = NONE
	}
	// set the condition
	rOP.isAnd = field.Cond != nil
	// set the field
	rOP.fieldName = field.Field
	// set the value with kind
	rOP.events = make([]Logic, 0)

	// check detection kind is scalar or sequence not map
	switch val.Kind {
	case yaml.ScalarNode:
		rOP.events = append(rOP.events, NewEventOperator(rOP.fieldName, rOP.op, val.Value))
	case yaml.SequenceNode:
		for _, v := range val.Content {
			rOP.events = append(rOP.events, NewEventOperator(rOP.fieldName, rOP.op, v.Value))
		}
	default:
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrRuleField,
			Msg:    "error while NewRuleOperate",
			Origin: fmt.Errorf("invalid detection value. detection must be scalar or sequence"),
		}
	}
	return rOP, nil
}

func (rOP *RuleOperator) Operation(log *model.CommonLogWrapper) bool {
	// set boolean with isAnd
	switch rOP.isAnd {
	case true:
		return And(rOP.events).Operation(log)
	case false:
		return Or(rOP.events).Operation(log)
	}
	return false
}

// # EventOperator
//
// EventOperator is an operator that compares the actual log and the condition and derives the result.
// There is a premise that the log that is the target of all operations uses CommonLogModel.
type EventOperator struct {
	op        RuleOpCode
	fieldName string
	condition string
}

func NewEventOperator(fieldName string, op RuleOpCode, condition string) Logic {
	// set the operation
	eOP := new(EventOperator)
	// set the field
	eOP.fieldName = fieldName
	// set the value
	eOP.condition = condition
	// set the operation
	eOP.op = op
	return eOP
}

func (eOP *EventOperator) Operation(log *model.CommonLogWrapper) bool {
	var strVal string

	// check log has field & get value
	switch eOP.fieldName {
	case "eventname":
		return eOP.compare(log.EventName)
	case "source":
		return eOP.compare(log.Source)
	case "timestamp":
		return eOP.compare(log.Timestmp)
	case "log":
		return eOP.compare(log.Log)
	default:
		// metadata
		if log.MetaData == nil {
			return false
		}
		// check metadata's type is map
		if _, ok := log.MetaData.(map[string]interface{}); !ok {
			return false
		}
		// check metadata has field & get value
		val, ok := log.MetaData.(map[string]interface{})[eOP.fieldName]
		if !ok {
			// not found field
			return false
		}
		// check value's type
		switch val := val.(type) {
		case string:
			strVal = val
		default:
			strVal = fmt.Sprintf("%v", val)
		}
		return eOP.compare(strVal)
	}
}

func (eOP *EventOperator) compare(dest string) bool {
	switch eOP.op {
	case STARTSWITH:
		return strings.HasPrefix(dest, eOP.condition)
	case ENDSWITH:
		return strings.HasSuffix(dest, eOP.condition)
	case CONTAINS:
		return strings.Contains(dest, eOP.condition)
	default:
		return dest == eOP.condition
	}
}

// # ParseRuleField
//
// The condition field of the Rule has the following format:
// - field|operation|condition
//
// # field
//
// The field is the field name of the target log.
//
// # operation
//
// The operation is the operation to be performed on the field.
// The operations are:
// - startswith : checks if the field starts with the condition
// - endswith : checks if the field ends with the condition
// - contains : checks if the field contains the condition
//
// # condition
//
// The condition is the value to be define AND or OR operation.
// If the condition is not defined, the operation is performed as OR.
func ParseRuleField(parser *participle.Parser[RuleFieldOper], key string) (*RuleFieldOper, error) {
	// null check
	if parser == nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrRuleField,
			Origin: fmt.Errorf("parser is nil"),
			Msg:    "error in ParseDetectionField",
		}
	}
	// interpret the field
	detectOpr, err := parser.ParseString(key, key)
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrRuleField,
			Msg:    "Failed to parse detection field",
			Origin: err,
		}
	}
	return detectOpr, nil
}
