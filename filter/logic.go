package filter

import (
	"fmt"
	perror "polvo/error"
	"reflect"

	"github.com/alecthomas/participle/v2"
	"gopkg.in/yaml.v3"
)

type Logic[T any] interface {
	operation(log T) bool
}

type DetectionOP int

const (
	STARTSWITH DetectionOP = iota
	ENDSWITH
	CONTAINS
)

type DetectionOper[T any] struct {
	op        DetectionOP
	isAnd     bool
	fieldName string
	val       []string
}

func NewDetectionOperate[T any](parser Parser, key string, val *Detection) (Logic[T], error) {
	field, err := ParseDetectionField(parser.DetectionFieldParser(), key)
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrDetectionField,
			Msg:    "error while NewDetectionOperate",
			Origin: err,
		}
	}
	// set the operation
	dOP := new(DetectionOper[T])

	// set opcode
	switch *field.Op {
	case "startswith":
		dOP.op = STARTSWITH
	case "endswith":
		dOP.op = ENDSWITH
	case "contains":
		dOP.op = CONTAINS
	default:
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrDetectionField,
			Msg:    "error while NewDetectionOperate",
			Origin: fmt.Errorf("invalid detection operation"),
		}
	}
	// set the condition
	dOP.isAnd = field.Cond != nil
	// set the field
	dOP.fieldName = field.Field
	// set the value with kind
	dOP.val = make([]string, 0)

	switch val.Node.Kind {
	case yaml.ScalarNode:
		dOP.val = append(dOP.val, val.Node.Value)
	case yaml.SequenceNode:
		for _, v := range val.Node.Content {
			dOP.val = append(dOP.val, v.Value)
		}
	default:
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrDetectionField,
			Msg:    "error while NewDetectionOperate",
			Origin: fmt.Errorf("invalid detection value. detection must be scalar or sequence"),
		}
	}
	return dOP, nil
}

func (d DetectionOper[T]) operation(log T) bool {
	// set boolean with isAnd
	return false
}

type ValueOper[T any] struct {
	op                      DetectionOP
	fieldName               string
	comp                    string
	isFieldNameCommonHeader bool
}

func NewValueOperate[T any](fieldName string, comp string, op DetectionOP) Logic[T] {
	// set the operation
	vOP := new(ValueOper[T])
	// set the field
	vOP.fieldName = fieldName
	// set the value
	vOP.comp = comp
	// set the operation
	vOP.op = op
	// precheck field Name is common header
	var instance T
	_, vOP.isFieldNameCommonHeader = reflect.TypeOf(instance).FieldByName(fieldName)
	return vOP
}

func (v ValueOper[T]) operation(log T) bool {
	// check log has field & get value

	switch v.op {
	case STARTSWITH:
	}
	return false
}

func ParseDetectionField(parser *participle.Parser[DetectionFieldOper], key string) (*DetectionFieldOper, error) {
	// null check
	if parser == nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrDetectionField,
			Origin: fmt.Errorf("parser is nil"),
			Msg:    "error in ParseDetectionField",
		}
	}
	// interpret the field
	detectOpr, err := parser.ParseString(key, key)
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrDetectionField,
			Msg:    "Failed to parse detection field",
			Origin: err,
		}
	}
	return detectOpr, nil
}
