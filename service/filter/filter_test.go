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
deny:
  "filter_bash":
    "condition":
      "eventname|startswith": "process"
      "Command|contains":
        - "bash"
        - "sleep"
    "exception":
      "eventname": "bashReadline"
      "Commandline|contains":
        - "ls"
        - "cat"
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

func TestNewDenyOperatorWithStartsWith(t *testing.T) {
	sample := `
deny:
  "filter_null":
    "condition":
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
	for selectionName, selection := range sampleRule.Deny {
		selection, err := filter.NewDenyOperator(parser, selectionName, &selection)
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

func TestNewDenyOperatorWithContains(t *testing.T) {
	sample := `
deny:
  "filter_null":
    "condition":
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
	for selectionName, selection := range sampleRule.Deny {
		selection, err := filter.NewDenyOperator(parser, selectionName, &selection)
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

func TestNewDenyOperatorWithEndsWith(t *testing.T) {
	sample := `
deny:
  "filter_null":
    "condition":
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
	for selectionName, selection := range sampleRule.Deny {
		selection, err := filter.NewDenyOperator(parser, selectionName, &selection)
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

func TestNewDenyOperatorWithAll(t *testing.T) {
	sample := `
deny:
  "filter_null":
    "condition":
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
	for selectionName, selection := range sampleRule.Deny {
		selection, err := filter.NewDenyOperator(parser, selectionName, &selection)
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

func TestNewDenyOperatorWithAllError(t *testing.T) {
	sample := `
deny:
  "filter_null":
    "condition":
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
	for selectionName, selection := range sampleRule.Deny {
		selection, err := filter.NewDenyOperator(parser, selectionName, &selection)
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

// (DEPRECATED)
//
// func TestNewDenyOperatorWithNOT(t *testing.T) {
// 	sample := `
// deny:
//   "!filter_null":
//     "eventname": "bashReadline"
//     "Commandline|contains|all":
//       - "echo"
//       - "cat"
// `
// 	sampleLog := `
// {
// 	"eventname": "bashReadline",
// 	"source": "eBPF",
// 	"timestamp": "2025-03-11T15:29:34+09:00",
// 	"log": "A user has entered a command in the bash shell",
// 	"metadata": {
// 		"Commandline": "echo hello world",
// 		"PID": 191998,
// 		"UID": 1000,
// 		"Username": "shhong"
// 	}
// }
// 	`

// 	var sampleRule filter.Filter
// 	// unmarshal yaml
// 	err := yaml.Unmarshal([]byte(sample), &sampleRule)
// 	if err != nil {
// 		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
// 	}
// 	// unmarshal log
// 	log := new(model.CommonLogWrapper)
// 	err = json.Unmarshal([]byte(sampleLog), log)
// 	if err != nil {
// 		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
// 	}
// 	// test
// 	for selectionName, selection := range sampleRule.Deny {
// 		selection, err := filter.NewDenyOperator(parser, selectionName, &selection)
// 		if err != nil {
// 			t.Fatalf("NewRuleOperate(%s) = nil, want not nil", sample)
// 		}
// 		out := selection.Operation(log)
// 		if !out {
// 			t.Fatalf("NewRuleOperate(%s) = %v, want true", selectionName, out)
// 		}
// 		t.Logf("NewRuleOperate(%s) = %v, want true", selectionName, out)
// 	}

// }

// func TestNewDenyOperatorWithWrongCollectionName(t *testing.T) {
// 	sample := `
// deny:
//   "fil!ter_null":
//     "eventname": "bashReadline"
//     "Commandline|contains|all":
//       - "echo"
//       - "cat"
// `
// 	sampleLog := `
// {
// 	"eventname": "bashReadline",
// 	"source": "eBPF",
// 	"timestamp": "2025-03-11T15:29:34+09:00",
// 	"log": "A user has entered a command in the bash shell",
// 	"metadata": {
// 		"Commandline": "echo hello world",
// 		"PID": 191998,
// 		"UID": 1000,
// 		"Username": "shhong"
// 	}
// }
// 	`

// 	var sampleRule filter.Filter
// 	// unmarshal yaml
// 	err := yaml.Unmarshal([]byte(sample), &sampleRule)
// 	if err != nil {
// 		t.Fatalf("yaml.Unmarshal(%s) = %v, want nil", sample, err)
// 	}
// 	// unmarshal log
// 	log := new(model.CommonLogWrapper)
// 	err = json.Unmarshal([]byte(sampleLog), log)
// 	if err != nil {
// 		t.Fatalf("json.Unmarshal(%s) = %v, want nil", sampleLog, err)
// 	}
// 	// test
// 	for selectionName, selection := range sampleRule.Deny {
// 		_, err := filter.NewDenyOperator(parser, selectionName, &selection)
// 		if err == nil {
// 			t.Fatalf("NewRuleOperate(%s) != nil, want nil", selectionName)
// 		}
// 		t.Logf("selection name '%s' returns err: %v, want true", selectionName, err)
// 	}

// }

func TestFilterOperation(t *testing.T) {
	sampleLog := `
{
	"eventname":"processCreate",
	"source":"eBPF",
	"timestamp":"2025-03-27T20:54:22.051986+09:00",
	"log":"A process has been created",
	"metadata":{
		"Argc":2,
		"Args":["sleep","1","","","","","","","",""],
		"BpfTimestamp":"2025-03-27T20:54:22.059441750+09:00",
		"Command":"cpuUsage.sh",
		"Envc":10,
		"Envs":["SHELL=/bin/bash","QT_ACCESSIBILITY=1","COLORTERM=truecolor","VSCODE_VERBOSE_LOGGING=true","XDG_CONFIG_DIRS=/etc/xdg/xdg-ubuntu:/etc/xdg","NVM_INC=/home/shhong/.nvm/versions/node/v22.12.0/include/node","XDG_MENU_PREFIX=gnome-","GNOME_DESKTOP_SESSION_ID=this-is-deprecated","LANGUAGE=en"],
		"Filename":"/usr/bin/sleep",
		"PID":92303,
		"PPID":92300,
		"TGID":92303,
		"UID":1000,
		"Username":"shhong"
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

func TestFilterOperationWithException(t *testing.T) {
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
	if out {
		t.Fatalf("FilterOperation() = %v, want false", out)
	}
	t.Logf("FilterOperation() = %v, want false", out)
}

func TestFilterOperationWithPassFilter(t *testing.T) {
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
	t.Logf("FilterOperation() = %v, want true", out)
}

func TestFilterOperationWithDrop(t *testing.T) {
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
