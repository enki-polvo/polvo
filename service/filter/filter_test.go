package filter_test

import (
	"encoding/json"
	"os"
	"polvo/service/filter"
	"polvo/service/model"
	"testing"

	"gopkg.in/yaml.v3"
)

const sampleFilter = `
version: 1.0
allow:
  "filter_test":
    "eventname|contains":
      - "bashReadline"
      - "process"
    "Commandline|contains":
      - "ls"
      - "cat"
deny:
  "!filter_NOT":
    "eventname|startswith": "process"
    "Commandline|endswith":
      - "bash"
      - "-al"
`

var (
	parser   filter.Parser
	filterOP filter.FilterOperator
)

func TestMain(m *testing.M) {
	var err error
	parser, err = filter.NewParser()
	if err != nil {
		panic(err)
	}
	filterOP, err = filter.NewFilterOperator([]byte(sampleFilter))
	if err != nil {
		panic(err)
	}
	code := m.Run()
	os.Exit(code)
}

func TestParseRuleField(t *testing.T) {
	sample := "field|startswith"

	// test
	result, err := filter.ParseRuleField(parser.RuleFieldParser(), sample)
	if err != nil {
		t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
	}
	if result.Field != "field" {
		t.Errorf("NewRuleOperate(%s).Field = %s, want field", sample, result.Field)
	}
	if result.Op == nil || *result.Op != "startswith" {
		t.Errorf("NewRuleOperate(%s).Op = %s, want startswith", sample, *result.Op)
	}
	if result.Cond != nil {
		t.Errorf("NewRuleOperate(%s).Cond = %s, want nil", sample, *result.Cond)
	}
}

func TestParseRuleFieldWithAll(t *testing.T) {
	sample := "field|contains|all"

	// test
	result, err := filter.ParseRuleField(parser.RuleFieldParser(), sample)
	if err != nil {
		t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
	}
	if result.Field != "field" {
		t.Errorf("NewRuleOperate(%s).Field = %s, want field", sample, result.Field)
	}
	if result.Op == nil || *result.Op != "contains" {
		t.Errorf("NewRuleOperate(%s).Op = %s, want contains", sample, *result.Op)
	}
	if result.Cond == nil || *result.Cond != "all" {
		t.Errorf("NewRuleOperate(%s).Cond = %s, want all", sample, *result.Cond)
	}
}

func TestParseRuleFieldWithEmpty(t *testing.T) {
	sample := "field|"

	// test
	_, err := filter.ParseRuleField(parser.RuleFieldParser(), sample)
	if err == nil {
		t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewRuleOperate(%s) = %v, want not nil", sample, err)
}

func TestParseRuleFieldWithOnlyAll(t *testing.T) {
	sample := "field|all"
	// test
	result, err := filter.ParseRuleField(parser.RuleFieldParser(), sample)
	if err != nil {
		t.Fatalf("NewRuleOperate(%s) != %v, want nil", sample, err)
	}
	if result.Field != "field" {
		t.Errorf("NewRuleOperate(%s).Field = %s, want field", sample, result.Field)
	}
	if result.Op != nil {
		t.Errorf("NewRuleOperate(%s).Op = %s, want nil", sample, *result.Op)
	}
	if result.Cond == nil || *result.Cond != "all" {
		t.Errorf("NewRuleOperate(%s).Cond = %s, want all", sample, *result.Cond)
	}
}

func TestParseRuleFieldWithInvalid(t *testing.T) {
	sample := "field|invalid"

	// test
	_, err := filter.ParseRuleField(parser.RuleFieldParser(), sample)
	if err == nil {
		t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewRuleOperate(%s) = %v, want not nil", sample, err)
}

func TestParseRuleFieldWithInvalidCond(t *testing.T) {
	sample := "field|endsswith|invalid"
	// test
	_, err := filter.ParseRuleField(parser.RuleFieldParser(), sample)
	if err == nil {
		t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewRuleOperate(%s) = %v, want not nil", sample, err)
}

func TestParseRuleFieldWithInvalidParser(t *testing.T) {
	sample := "field|startswith"

	// test
	_, err := filter.ParseRuleField(nil, sample)
	if err == nil {
		t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewRuleOperate(%s) = %v, want not nil", sample, err)
}

func TestNewRuleSelectionOperatorWithStartsWith(t *testing.T) {
	sample := `
allow:
  "filter_null":
    "eventname|startswith": "bash"
    "source": "eBPF"

`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		selection, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err != nil {
			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
		}
		out := selection.Operation(log)
		if !out {
			t.Fatalf("NewRuleOperate(%s) = %v, want true", selectionName, out)
		}
		t.Logf("NewRuleOperate(%s) = %v, want true", selectionName, out)
	}
}

func TestNewRuleSelectionOperatorWithContains(t *testing.T) {
	sample := `
allow:
  "filter_null":
    "eventname": "bashReadline"
    "Commandline|contains":
      - "echo"
      - "ls"
`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		selection, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err != nil {
			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
		}
		out := selection.Operation(log)
		if !out {
			t.Fatalf("NewRuleOperate(%s) = %v, want true", selectionName, out)
		}
		t.Logf("NewRuleOperate(%s) = %v, want true", selectionName, out)
	}
}

func TestNewRuleSelectionOperatorWithEndsWith(t *testing.T) {
	sample := `
allow:
  "filter_null":
    "eventname|endswith": "Readline"
    "Username": "shhong"
`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`
	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		selection, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err != nil {
			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
		}
		out := selection.Operation(log)
		if !out {
			t.Fatalf("NewRuleOperate(%s) = %v, want true", selectionName, out)
		}
		t.Logf("NewRuleOperate(%s) = %v, want true", selectionName, out)
	}
}

func TestNewRuleSelectionOperatorWithAll(t *testing.T) {
	sample := `
allow:
  "filter_null":
    "Commandline|contains|all":
      - "echo"
      - "hello"
    "source|endswith":
      - "BPF"
      - "PF"
`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`
	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		selection, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err != nil {
			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
		}
		out := selection.Operation(log)
		if !out {
			t.Fatalf("NewRuleOperate(%s) = %v, want true", selectionName, out)
		}
		t.Logf("NewRuleOperate(%s) = %v, want true", selectionName, out)
	}
}

func TestNewRuleSelectionOperatorWithAllError(t *testing.T) {
	sample := `
allow:
  "filter_null":
    "eventname": "bashReadline"
    "Commandline|contains|all":
      - "echo"
      - "cat"
`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		selection, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err != nil {
			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
		}
		out := selection.Operation(log)
		if out {
			t.Fatalf("NewRuleOperate(%s) = %v, want false", selectionName, out)
		}
		t.Logf("NewRuleOperate(%s) = %v, want false", selectionName, out)
	}

}

func TestNewRuleSelectionOperatorWithNOT(t *testing.T) {
	sample := `
allow:
  "!filter_null":
    "eventname": "bashReadline"
    "Commandline|contains|all":
      - "echo"
      - "cat"
`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		selection, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err != nil {
			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
		}
		out := selection.Operation(log)
		if !out {
			t.Fatalf("NewRuleOperate(%s) = %v, want true", selectionName, out)
		}
		t.Logf("NewRuleOperate(%s) = %v, want true", selectionName, out)
	}

}

func TestNewRuleSelectionOperatorWithWrongCollectionName(t *testing.T) {
	sample := `
allow:
  "fil!ter_null":
    "eventname": "bashReadline"
    "Commandline|contains|all":
      - "echo"
      - "cat"
`
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "echo hello world",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	var sampleRule filter.Filter
	// unmarshal yaml
	err := yaml.Unmarshal([]byte(sample), &sampleRule)
	if err != nil {
		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
	}
	// unmarshal log
	log := new(model.CommonLogWrapper)
	err = json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}
	// test
	for selectionName, selection := range sampleRule.Allow {
		_, err := filter.NewRuleSelectionOperator(parser, selectionName, &selection)
		if err == nil {
			t.Fatalf("NewRuleOperate(%s) != nil, want nil", selectionName)
		}
		t.Logf("selection name '%s' returns err: %v, want true", selectionName, err)
	}

}

func TestFilterOperation(t *testing.T) {
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "ls -al",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	// unmarshal log
	log := new(model.CommonLogWrapper)
	err := json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}

	// test
	out := filterOP.Operation(log)
	if !out {
		t.Fatalf("FilterOperation() = %v, want true", out)
	}
	t.Logf("FilterOperation() = %v, want true", out)
}

func TestFilterOperationWithBothDenyAndAllow(t *testing.T) {
	sampleLog := `
{
	"eventname": "bashReadline",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user has entered a command in the bash shell",
	"metadata": {
		"Commandline": "ls -al",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	// unmarshal log
	log := new(model.CommonLogWrapper)
	err := json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}

	// test
	out := filterOP.Operation(log)
	if !out {
		t.Fatalf("FilterOperation() = %v, want true", out)
	}
	t.Logf("FilterOperation() = %v, want true", out)
}

func TestFilterOperationWithDeny(t *testing.T) {
	sampleLog := `
{
	"eventname": "fileCreate",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user created file",
	"metadata": {
		"fileName": "hello.txt",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	// unmarshal log
	log := new(model.CommonLogWrapper)
	err := json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}

	// test
	out := filterOP.Operation(log)
	if out {
		t.Fatalf("FilterOperation() = %v, want false", out)
	}
	t.Logf("FilterOperation() = %v, want false", out)
}

func TestFilterOperationWithNothing(t *testing.T) {
	sampleLog := `
{
	"eventname": "processCreate",
	"source": "eBPF",
	"timestamp": "2025-03-11T15:29:34+09:00",
	"log": "A user created process",
	"metadata": {
		"Commandline": "/bin/bash",
		"PID": 191998,
		"UID": 1000,
		"Username": "shhong"
	}
}
	`

	// unmarshal log
	log := new(model.CommonLogWrapper)
	err := json.Unmarshal([]byte(sampleLog), log)
	if err != nil {
		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
	}

	// test
	out := filterOP.Operation(log)
	if !out {
		t.Fatalf("FilterOperation() = %v, want true", out)
	}
	t.Logf("FilterOperation() = %v, want true", out)
}
