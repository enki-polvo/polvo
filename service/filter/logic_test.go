package filter_test

import (
	"encoding/json"
	"os"
	"polvo/service/filter"
	"polvo/service/model"
	"testing"

	"gopkg.in/yaml.v3"
)

var (
	parser filter.Parser
)

func TestMain(m *testing.M) {
	var err error
	parser, err = filter.NewParser()
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

func TestNewRuleOperator(t *testing.T) {
	sample := `
detection:
  "filter_null":
    "eventname|startswith": "bash"
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

	var sampleRule filter.Detection
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
	for _, dval := range sampleRule.Detections {
		for key, val := range dval {
			dOPs, err := filter.NewRuleOperator(parser, key, &val)
			if err != nil {
				t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
			}
			out := dOPs.Operation(log)
			if !out {
				t.Fatalf("NewRuleOperate(%s) = %v, want true", key, out)
			}
			t.Logf("NewRuleOperate(%s) = %v, want true", key, out)
		}
	}
}

func TestNewRuleOperator2(t *testing.T) {
	sample := `
detection:
  "filter_null":
    "eventname|endswith": "Readline"
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

	var sampleRule filter.Detection
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
	for _, dval := range sampleRule.Detections {
		for key, val := range dval {
			dOPs, err := filter.NewRuleOperator(parser, key, &val)
			if err != nil {
				t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
			}
			out := dOPs.Operation(log)
			if !out {
				t.Fatalf("NewRuleOperate(%s) = %v, want true", key, out)
			}
			t.Logf("NewRuleOperate(%s) = %v, want true", key, out)
		}
	}
}

func TestNewRuleOperator3(t *testing.T) {
	sample := `
detection:
  "filter_null":
    "eventname|contains|all":
      - "bash"
      - "hello"
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

	var sampleRule filter.Detection
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
	for _, dval := range sampleRule.Detections {
		for key, val := range dval {
			dOPs, err := filter.NewRuleOperator(parser, key, &val)
			if err != nil {
				t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
			}
			out := dOPs.Operation(log)
			if out {
				t.Fatalf("NewRuleOperate(%s) = %v, want false", key, out)
			}
			t.Logf("NewRuleOperate(%s) = %v, want false", key, out)
		}
	}
}

func TestNewRuleOperator4(t *testing.T) {
	sample := `
detection:
  "filter_null":
    "Commandline|contains|all":
      - "echo"
      - "hello"
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

	var sampleRule filter.Detection
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
	for _, dval := range sampleRule.Detections {
		for key, val := range dval {
			dOPs, err := filter.NewRuleOperator(parser, key, &val)
			if err != nil {
				t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
			}
			out := dOPs.Operation(log)
			if !out {
				t.Fatalf("NewRuleOperate(%s) = %v, want true", key, out)
			}
			t.Logf("NewRuleOperate(%s) = %v, want true", key, out)
		}
	}
}

func TestNewRuleOperator5(t *testing.T) {
	sample := `
detection:
  "filter_null":
    "Commandline|all":
      - "echo"
      - "hello"
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

	var sampleRule filter.Detection
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
	for _, dval := range sampleRule.Detections {
		for key, val := range dval {
			dOPs, err := filter.NewRuleOperator(parser, key, &val)
			if err != nil {
				t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
			}
			out := dOPs.Operation(log)
			if out {
				t.Fatalf("NewRuleOperate(%s) = %v, want false", key, out)
			}
			t.Logf("NewRuleOperate(%s) = %v, want false", key, out)
		}
	}
}
