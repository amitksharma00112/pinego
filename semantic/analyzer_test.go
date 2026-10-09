package semantic

import (
	"strings"
	"testing"

	"github.com/amitksharma00112/pinego/ast"
	"github.com/amitksharma00112/pinego/parser"
)

func TestAnalyzeUndefinedVariable(t *testing.T) {
	program, err := parser.ParseSource("y = x + 5")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error")
	}
}

func TestAnalyzeReassignmentBeforeDeclaration(t *testing.T) {
	program, err := parser.ParseSource("x := 10")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected error for reassignment of undefined variable")
	}
}

func TestAnalyzeBlockScope(t *testing.T) {
	source := `x = 10
if close > open
    y = x + 5
z = y + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for y outside block")
	}
}

func TestAnalyzeUndefinedVariableInsideBlock(t *testing.T) {
	source := `x = 10
if x > 0
    y = unknown + 5`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for unknown")
	}
}
func TestAnalyzeVariableDoesNotEscapeBlock(t *testing.T) {
	source := `x = 10
if x > 0
    y = 20
z = y + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for y outside block")
	}
}

func TestAnalyzeBuiltinVariables(t *testing.T) {
	program, err := parser.ParseSource("x = close + open")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}

func TestAnalyzeBuiltinNamespace(t *testing.T) {
	program, err := parser.ParseSource("x = ta.sma(close, 20)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}

func TestAnalyzeUnknownBuiltinFunction(t *testing.T) {
	program, err := parser.ParseSource("x = ta.unknown(close, 20)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected unknown builtin function error")
	}
}

func TestAnalyzeBuiltinFunctionArgumentCount(t *testing.T) {
	program, err := parser.ParseSource("x = ta.sma(close)")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected argument count error")
	}
}

func TestAnalyzeUnknownType(t *testing.T) {
	program, err := parser.ParseSource("banana x = 10")
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected unknown type error")
	}
}

func TestAnalyzeTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`int x = "hello"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}

func TestAnalyzeIntFloatMismatch(t *testing.T) {
	program, err := parser.ParseSource(`int x = 10.5`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}

func TestAnalyzeReassignmentTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`int x = 10
x := "hello"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected reassignment type mismatch error")
	}
}
func TestAnalyzeBinaryExpressionTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`int x = 10 + "hello"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected binary expression type mismatch error")
	}
}

func TestAnalyzeFunctionParameters(t *testing.T) {
	source := `add(x, y) =>
    x + y`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}

func TestAnalyzeFunctionBody(t *testing.T) {
	source := `add(x, y) =>
    x + unknown`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for unknown")
	}
}

func TestAnalyzeFunctionParameterTypeMismatch(t *testing.T) {
	source := `add(string x) =>
    x + 10`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected function parameter type mismatch error")
	}
}

func TestAnalyzeUserFunctionArgumentCount(t *testing.T) {
	source := `add(x, y) =>
    x + y

z = add(10)`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected user function argument count error")
	}

	if !strings.Contains(err.Error(), "expects 2 arguments") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAnalyzeUserFunctionArgumentTypeMismatch(t *testing.T) {
	source := `add(int x, int y) =>
    x + y

z = add("hello", 20)`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected user function argument type mismatch")
	}
}

func TestInferFunctionReturnType(t *testing.T) {
	source := `add(int x, int y) =>
    x + y`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	got, ok := analyzer.functionReturnTypes["add"]
	if !ok {
		t.Fatal("expected return type for function add")
	}

	if got != "int" {
		t.Fatalf("got return type %q, want %q", got, "int")
	}
}

func TestInferFunctionCallType(t *testing.T) {
	source := `add(int x, int y) =>
    x + y

float result = add(10, 20)`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	if got := analyzer.inferExprType(program.Statements[1].(*ast.VariableDeclStmt).Value); got != "int" {
		t.Fatalf("got function call type %q, want %q", got, "int")
	}
}

func TestAnalyzeComparisonTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`x = 10 > "hello"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected comparison type mismatch error")
	}
}

func TestAnalyzeLogicalTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`x = 10 and true`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected logical operator type mismatch error")
	}
}
func TestAnalyzeTernaryTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`x = true ? 10 : "hello"`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected ternary type mismatch error")
	}
}

func TestInferHistoryExpressionType(t *testing.T) {
	program, err := parser.ParseSource(`x = close[1]`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	got := analyzer.inferExprType(stmt.Value)
	if got != "float" {
		t.Fatalf("got history expression type %q, want %q", got, "float")
	}
}

func TestInferComparisonExpressionType(t *testing.T) {
	program, err := parser.ParseSource(`x = close > open`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	got := analyzer.inferExprType(stmt.Value)
	if got != "bool" {
		t.Fatalf("got comparison expression type %q, want %q", got, "bool")
	}
}

func TestInferUnaryExpressionType(t *testing.T) {
	program, err := parser.ParseSource(`x = not true`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	got := analyzer.inferExprType(stmt.Value)
	if got != "bool" {
		t.Fatalf("got unary expression type %q, want %q", got, "bool")
	}
}

func TestInferTernaryExpressionType(t *testing.T) {
	program, err := parser.ParseSource(`x = true ? 10 : 20.5`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	got := analyzer.inferExprType(stmt.Value)
	if got != "float" {
		t.Fatalf("got ternary expression type %q, want %q", got, "float")
	}
}

func TestAnalyzeReturnOutsideFunction(t *testing.T) {
	program, err := parser.ParseSource(`return 10`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected return outside function error")
	}
}

func TestAnalyzeBreakOutsideLoop(t *testing.T) {
	program, err := parser.ParseSource(`break`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected break outside loop error")
	}
}

func TestAnalyzeContinueOutsideLoop(t *testing.T) {
	program, err := parser.ParseSource(`continue`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected continue outside loop error")
	}
}
func TestAnalyzeBreakInsideWhile(t *testing.T) {
	source := `while close > open
    break`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}

func TestAnalyzeUndefinedVariableInsideWhile(t *testing.T) {
	source := `while x < 10
    y = unknown + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for unknown")
	}
}

func TestAnalyzeContinueInsideWhile(t *testing.T) {
	source := `while close > open
    continue`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}
func TestAnalyzeForLoop(t *testing.T) {
	source := `for i = 0 to 10
    x = i
    continue`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}

func TestAnalyzeUndefinedVariableInsideFor(t *testing.T) {
	source := `for i = 0 to 10
    x = unknown + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for unknown")
	}
}
func TestAnalyzeForVariableDoesNotEscape(t *testing.T) {
	source := `for i = 0 to 10
    x = i
y = i + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for i outside loop")
	}
}
func TestAnalyzeNestedFunction(t *testing.T) {
	source := `outer(x) =>
    inner(y) =>
        y + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected nested function declaration error")
	}
}

func TestAnalyzeIfConditionTypeMismatch(t *testing.T) {
	source := `if 10
    x = 20`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected boolean condition error")
	}
}
func TestAnalyzeWhileConditionTypeMismatch(t *testing.T) {
	source := `while 10
    break`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected boolean while condition error")
	}
}
func TestAnalyzeForRangeTypeMismatch(t *testing.T) {
	source := `for i = "hello" to 10
    x = i`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected for range type mismatch error")
	}
}

func TestInferFunctionReturnTypeFromBranches(t *testing.T) {
	source := `getValue(int x) =>
    if x > 0
        return 10
    else
        return 20`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}

	got, ok := analyzer.functionReturnTypes["getValue"]
	if !ok {
		t.Fatal("expected return type for function getValue")
	}

	if got != "int" {
		t.Fatalf("got return type %q, want %q", got, "int")
	}
}
func TestAnalyzeFunctionReturnTypeMismatch(t *testing.T) {
	source := `getValue(int x) =>
    if x > 0
        return 10
    else
        return "hello"`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected function return type mismatch")
	}
}
func TestAnalyzeInvalidAssignmentTarget(t *testing.T) {
	program, err := parser.ParseSource(`10 = x`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected invalid assignment target error")
	}
}

func TestAnalyzeFunctionLocalVariableDoesNotEscape(t *testing.T) {
	source := `makeValue() =>
    local = 10
    return local

x = local`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for local")
	}
}
func TestAnalyzeWhileVariableDoesNotEscape(t *testing.T) {
	source := `while close > open
    y = 20

z = y + 1`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected undefined variable error for y outside while")
	}
}
func TestAnalyzeFunctionReturnTypeAtCallSite(t *testing.T) {
	source := `add(int x, int y) =>
    x + y

int result = add(10, 20)`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("unexpected semantic error: %v", err)
	}
}
func TestAnalyzeUnknownFunctionParameterType(t *testing.T) {
	source := `add(banana x) =>
    x`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected unknown parameter type error")
	}
}
func TestAnalyzeBuiltinFunctionArgumentTypeMismatch(t *testing.T) {
	program, err := parser.ParseSource(`x = ta.sma("hello", 20)`)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	err = analyzer.Analyze(program)
	if err == nil {
		t.Fatal("expected builtin function argument type mismatch")
	}
}
func TestAnalyzeNACompatibility(t *testing.T) {
	source := `
float x = na
int y = na
bool z = na
string s = na
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("expected na to be compatible, got error: %v", err)
	}
}

func TestAnalyzeRecursiveFunction(t *testing.T) {
	source := `
countdown(int n) =>
	if n > 0
		return countdown(n - 1)
	else
		return 0
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err != nil {
		t.Fatalf("expected recursive function to be valid, got error: %v", err)
	}
}
func TestAnalyzeDuplicateFunction(t *testing.T) {
	source := `
add(int x) =>
	x

add(int x) =>
	x + 1
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err == nil {
		t.Fatal("expected duplicate function declaration error")
	}
}
func TestAnalyzeDuplicateVariable(t *testing.T) {
	source := `
int x = 10
int x = 20
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err == nil {
		t.Fatal("expected duplicate variable declaration error")
	}
}
func TestAnalyzeCompoundAssignmentTypeMismatch(t *testing.T) {
	source := `
x = 10
x += "hello"
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err == nil {
		t.Fatal("expected compound assignment type mismatch")
	}
}
func TestAnalyzeCompoundAssignmentNonNumeric(t *testing.T) {
	source := `
x = true
x += false
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err == nil {
		t.Fatal("expected compound assignment on bool to fail")
	}
}
func TestAnalyzeDuplicateAssignmentDeclaration(t *testing.T) {
	source := `
x = 10
x = 20
`

	program, err := parser.ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	analyzer := New()

	if err := analyzer.Analyze(program); err == nil {
		t.Fatal("expected duplicate variable declaration error")
	}
}
