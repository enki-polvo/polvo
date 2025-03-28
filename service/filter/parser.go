package filter

import (
	perror "polvo/error"

	"github.com/alecthomas/participle/v2"
)

type RuleFieldOper struct {
	Field string  `@Ident`
	Op    *string `("|" @("startswith" | "endswith" | "contains"))?`
	Cond  *string `("|" @("all"))?`
}

type Parser interface {
	RuleFieldParser() *participle.Parser[RuleFieldOper]
}

type parser struct {
	ruleFieldParser *participle.Parser[RuleFieldOper]
}

func (p *parser) RuleFieldParser() *participle.Parser[RuleFieldOper] {
	return p.ruleFieldParser
}

func NewParser() (Parser, error) {
	var err error

	p := new(parser)

	p.ruleFieldParser, err = participle.Build[RuleFieldOper]()
	if err != nil {
		return nil, perror.PolvoFilterError{
			Code:   perror.ErrRuleField,
			Msg:    "Failed to build parser in ParseRuleField",
			Origin: err,
		}
	}

	return p, nil
}
