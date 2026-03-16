package alerts

import (
	"testing"
)

func TestCompileOperands_SingleBucket(t *testing.T) {
	operandsJSON := []byte(`[{"type":"bucket","ref":"liquid","label":"Liquid Balance","operator":"+"}]`)
	expr, err := CompileOperands(operandsJSON, "<", "5000")
	if err != nil {
		t.Fatalf("CompileOperands failed: %v", err)
	}
	want := "liquid < 5000"
	if expr != want {
		t.Errorf("got %q, want %q", expr, want)
	}
}

func TestCompileOperands_MultipleMixed(t *testing.T) {
	operandsJSON := []byte(`[
		{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"},
		{"type":"account","ref":"acct-1","label":"Acct 1","operator":"-"}
	]`)
	expr, err := CompileOperands(operandsJSON, "<", "5000")
	if err != nil {
		t.Fatalf("CompileOperands failed: %v", err)
	}
	want := `(liquid - accounts["acct-1"]) < 5000`
	if expr != want {
		t.Errorf("got %q, want %q", expr, want)
	}
}

func TestCompileOperands_GroupRef(t *testing.T) {
	operandsJSON := []byte(`[{"type":"group","ref":"1","label":"My Group","operator":"+"}]`)
	expr, err := CompileOperands(operandsJSON, ">", "1000")
	if err != nil {
		t.Fatalf("CompileOperands failed: %v", err)
	}
	want := `groups["1"] > 1000`
	if expr != want {
		t.Errorf("got %q, want %q", expr, want)
	}
}

func TestCompileOperands_NetWorth(t *testing.T) {
	operandsJSON := []byte(`[{"type":"bucket","ref":"net_worth","label":"Net Worth","operator":"+"}]`)
	expr, err := CompileOperands(operandsJSON, "<", "50000")
	if err != nil {
		t.Fatalf("CompileOperands failed: %v", err)
	}
	want := "net_worth < 50000"
	if expr != want {
		t.Errorf("got %q, want %q", expr, want)
	}
}

func TestCompileOperands_AllComparisons(t *testing.T) {
	operandsJSON := []byte(`[{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}]`)
	tests := []struct {
		cmp  string
		want string
	}{
		{"<", "liquid < 100"},
		{"<=", "liquid <= 100"},
		{">", "liquid > 100"},
		{">=", "liquid >= 100"},
		{"==", "liquid == 100"},
	}
	for _, tc := range tests {
		t.Run(tc.cmp, func(t *testing.T) {
			expr, err := CompileOperands(operandsJSON, tc.cmp, "100")
			if err != nil {
				t.Fatalf("CompileOperands(%q) failed: %v", tc.cmp, err)
			}
			if expr != tc.want {
				t.Errorf("got %q, want %q", expr, tc.want)
			}
		})
	}
}

func TestCompileOperands_Empty(t *testing.T) {
	_, err := CompileOperands([]byte(`[]`), "<", "100")
	if err == nil {
		t.Error("expected error for empty operands, got nil")
	}
}

func TestCompileOperands_InvalidComparison(t *testing.T) {
	operandsJSON := []byte(`[{"type":"bucket","ref":"liquid","label":"Liquid","operator":"+"}]`)
	_, err := CompileOperands(operandsJSON, "!=", "100")
	if err == nil {
		t.Error("expected error for invalid comparison '!=', got nil")
	}
}

func TestEvaluate_True(t *testing.T) {
	env := Environment{
		Liquid:   4000,
		Accounts: map[string]float64{},
		Groups:   map[string]float64{},
	}
	result, err := Evaluate("liquid < 5000", env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !result {
		t.Error("expected true, got false")
	}
}

func TestEvaluate_False(t *testing.T) {
	env := Environment{
		Liquid:   6000,
		Accounts: map[string]float64{},
		Groups:   map[string]float64{},
	}
	result, err := Evaluate("liquid < 5000", env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result {
		t.Error("expected false, got true")
	}
}

func TestEvaluate_AccountRef(t *testing.T) {
	env := Environment{
		Liquid:   10000,
		Accounts: map[string]float64{"acct-1": 3000},
		Groups:   map[string]float64{},
	}
	// (liquid - accounts["acct-1"]) < 5000 => (10000 - 3000) = 7000 < 5000 => false
	result, err := Evaluate(`(liquid - accounts["acct-1"]) < 5000`, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if result {
		t.Error("expected false (7000 < 5000), got true")
	}
}

func TestEvaluate_GroupRef(t *testing.T) {
	env := Environment{
		Groups:   map[string]float64{"1": 1500},
		Accounts: map[string]float64{},
	}
	result, err := Evaluate(`groups["1"] > 1000`, env)
	if err != nil {
		t.Fatalf("Evaluate failed: %v", err)
	}
	if !result {
		t.Error("expected true (1500 > 1000), got false")
	}
}

func TestValidate_Valid(t *testing.T) {
	err := Validate("liquid < 5000")
	if err != nil {
		t.Errorf("expected valid expression, got error: %v", err)
	}
}

func TestValidate_Invalid(t *testing.T) {
	err := Validate("liquid < < 5000")
	if err == nil {
		t.Error("expected error for invalid expression, got nil")
	}
}
