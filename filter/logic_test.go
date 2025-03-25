package filter_test

import (
	"os"
	"polvo/filter"
	"testing"
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

func TestParseDetectionField(t *testing.T) {
	sample := "field|startswith"

	// test
	result, err := filter.ParseDetectionField(parser.DetectionFieldParser(), sample)
	if err != nil {
		t.Fatalf("NewDetectionOperate(%s) = nil, want not nil", sample)
	}
	if result.Field != "field" {
		t.Errorf("NewDetectionOperate(%s).Field = %s, want field", sample, result.Field)
	}
	if result.Op == nil || *result.Op != "startswith" {
		t.Errorf("NewDetectionOperate(%s).Op = %s, want startswith", sample, *result.Op)
	}
	if result.Cond != nil {
		t.Errorf("NewDetectionOperate(%s).Cond = %s, want nil", sample, *result.Cond)
	}
}

func TestParseDetectionFieldWithAll(t *testing.T) {
	sample := "field|contains|all"

	// test
	result, err := filter.ParseDetectionField(parser.DetectionFieldParser(), sample)
	if err != nil {
		t.Fatalf("NewDetectionOperate(%s) = nil, want not nil", sample)
	}
	if result.Field != "field" {
		t.Errorf("NewDetectionOperate(%s).Field = %s, want field", sample, result.Field)
	}
	if result.Op == nil || *result.Op != "contains" {
		t.Errorf("NewDetectionOperate(%s).Op = %s, want contains", sample, *result.Op)
	}
	if result.Cond == nil || *result.Cond != "all" {
		t.Errorf("NewDetectionOperate(%s).Cond = %s, want all", sample, *result.Cond)
	}
}

func TestParseDetectionFieldWithInvalid(t *testing.T) {
	sample := "field|invalid"

	// test
	_, err := filter.ParseDetectionField(parser.DetectionFieldParser(), sample)
	if err == nil {
		t.Fatalf("NewDetectionOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewDetectionOperate(%s) = %v, want not nil", sample, err)
}

func TestParseDetectionFieldWithInvalidCond(t *testing.T) {
	sample := "field|endsswith|invalid"
	// test
	_, err := filter.ParseDetectionField(parser.DetectionFieldParser(), sample)
	if err == nil {
		t.Fatalf("NewDetectionOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewDetectionOperate(%s) = %v, want not nil", sample, err)
}

func TestParseDetectionFieldWithInvalidParser(t *testing.T) {
	sample := "field|startswith"

	// test
	_, err := filter.ParseDetectionField(nil, sample)
	if err == nil {
		t.Fatalf("NewDetectionOperate(%s) = nil, want not nil", sample)
	}
	t.Logf("NewDetectionOperate(%s) = %v, want not nil", sample, err)
}
