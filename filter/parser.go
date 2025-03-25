package filter

import (
	perror "polvo/error"

	"github.com/alecthomas/participle/v2"
)

type DetectionFieldOper struct {
	Field string  `@Ident`
	Op    *string `("|" @("startswith" | "endswith" | "contains"))?`
	Cond  *string `("|" @("all"))?`
}

type Parser interface {
	DetectionFieldParser() *participle.Parser[DetectionFieldOper]
}

type parser struct {
	detectionFieldParser *participle.Parser[DetectionFieldOper]
}

func (p *parser) DetectionFieldParser() *participle.Parser[DetectionFieldOper] {
	return p.detectionFieldParser
}

func NewParser() (Parser, error) {
	var err error

	p := new(parser)

	p.detectionFieldParser, err = participle.Build[DetectionFieldOper]()
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrDetectionField,
			Msg:    "Failed to build parser in ParseDetectionField",
			Origin: err,
		}
	}

	return p, nil
}
