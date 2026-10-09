package parser

import (
	"testing"

	"github.com/amitksharma00112/pinego/ast"
)

func TestParseAssignment(t *testing.T) {
	program, err := ParseSource("x = 10")

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(program.Statements) != 1 {
		t.Fatalf(
			"got %d statements, want 1",
			len(program.Statements),
		)
	}

	stmt, ok := program.Statements[0].(*ast.AssignmentStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.AssignmentStmt",
			program.Statements[0],
		)
	}

	if stmt.Operator != "=" {
		t.Fatalf(
			"got operator %q, want =",
			stmt.Operator,
		)
	}
}

func TestParseOperatorPrecedence(t *testing.T) {
	program, err := ParseSource(
		"x = 10 + 20 * 5",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	add, ok := stmt.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.BinaryExpr",
			stmt.Value,
		)
	}

	if add.Operator != "+" {
		t.Fatalf(
			"got operator %q, want +",
			add.Operator,
		)
	}

	mul, ok := add.Right.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.BinaryExpr",
			add.Right,
		)
	}

	if mul.Operator != "*" {
		t.Fatalf(
			"got operator %q, want *",
			mul.Operator,
		)
	}
}

func TestParseMemberAndCall(t *testing.T) {
	program, err := ParseSource(
		"x = ta.sma(close, 20)",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	call, ok := stmt.Value.(*ast.CallExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.CallExpr",
			stmt.Value,
		)
	}

	member, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.MemberExpr",
			call.Callee,
		)
	}

	if member.Member != "sma" {
		t.Fatalf(
			"got member %q, want sma",
			member.Member,
		)
	}

	if len(call.Arguments) != 2 {
		t.Fatalf(
			"got %d arguments, want 2",
			len(call.Arguments),
		)
	}
}

func TestParseHistoryOperator(t *testing.T) {
	program, err := ParseSource(
		"x = close[1]",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	index, ok := stmt.Value.(*ast.IndexExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.IndexExpr",
			stmt.Value,
		)
	}

	number, ok := index.Index.(*ast.NumberLiteral)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.NumberLiteral",
			index.Index,
		)
	}

	if number.Value != 1 {
		t.Fatalf(
			"got %v, want 1",
			number.Value,
		)
	}
}

func TestParseTernary(t *testing.T) {
	program, err := ParseSource(
		"x = close > open ? high : low",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	conditional, ok := stmt.Value.(*ast.ConditionalExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.ConditionalExpr",
			stmt.Value,
		)
	}

	if conditional.Then.(*ast.Identifier).Name != "high" {
		t.Fatalf("unexpected then expression")
	}

	if conditional.Else.(*ast.Identifier).Name != "low" {
		t.Fatalf("unexpected else expression")
	}
}

func TestParseIfElse(t *testing.T) {
	source := `if close > open
    x = close
else
    x = open`

	program, err := ParseSource(source)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	root, ok := program.Statements[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.IfStmt",
			program.Statements[0],
		)
	}

	if len(root.Then.Statements) != 1 {
		t.Fatalf("unexpected then block")
	}

	if root.Else == nil {
		t.Fatal("expected else block")
	}

	if len(root.Else.Statements) != 1 {
		t.Fatalf("unexpected else block")
	}
}

func TestParseElseIf(t *testing.T) {
	source := `if close > open
    x = close
else if close < open
    x = open
else
    x = na`

	program, err := ParseSource(source)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	root, ok := program.Statements[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.IfStmt",
			program.Statements[0],
		)
	}

	if root.ElseIf == nil {
		t.Fatal("expected else-if")
	}

	if root.ElseIf.Condition == nil {
		t.Fatal("expected else-if condition")
	}

	if root.ElseIf.Else == nil {
		t.Fatal("expected final else")
	}
}

func TestParseUnaryAndLogical(t *testing.T) {
	program, err := ParseSource(
		"x = not close > open and volume > 100000",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	root, ok := stmt.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.BinaryExpr",
			stmt.Value,
		)
	}

	if root.Operator != "and" {
		t.Fatalf(
			"got root operator %q, want and",
			root.Operator,
		)
	}
}

func TestParseVariableDeclaration(t *testing.T) {
	program, err := ParseSource(
		"var x = 10",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt, ok := program.Statements[0].(*ast.VariableDeclStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.VariableDeclStmt",
			program.Statements[0],
		)
	}

	if stmt.Keyword != "var" {
		t.Fatalf(
			"got %q, want var",
			stmt.Keyword,
		)
	}

	if stmt.Name.Name != "x" {
		t.Fatalf(
			"got %q, want x",
			stmt.Name.Name,
		)
	}
}

func TestParseVaripDeclaration(t *testing.T) {
	program, err := ParseSource(
		"varip x = 0",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.VariableDeclStmt)

	if stmt.Keyword != "varip" {
		t.Fatalf(
			"got %q, want varip",
			stmt.Keyword,
		)
	}
}

func TestParseTypedVariable(t *testing.T) {
	program, err := ParseSource(
		"float x = close",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt, ok := program.Statements[0].(*ast.VariableDeclStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.VariableDeclStmt",
			program.Statements[0],
		)
	}

	if stmt.TypeName != "float" {
		t.Fatalf(
			"got type %q, want float",
			stmt.TypeName,
		)
	}

	if stmt.Name.Name != "x" {
		t.Fatalf(
			"got name %q, want x",
			stmt.Name.Name,
		)
	}
}

func TestParseVarTypedVariable(t *testing.T) {
	program, err := ParseSource(
		"var float x = close",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.VariableDeclStmt)

	if stmt.Keyword != "var" {
		t.Fatalf(
			"got keyword %q, want var",
			stmt.Keyword,
		)
	}

	if stmt.TypeName != "float" {
		t.Fatalf(
			"got type %q, want float",
			stmt.TypeName,
		)
	}
}

func TestParseFunctionDeclaration(t *testing.T) {
	source := `add(float x, float y) =>
    result = x + y
    return result`

	program, err := ParseSource(source)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	fn, ok := program.Statements[0].(*ast.FunctionDeclStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.FunctionDeclStmt",
			program.Statements[0],
		)
	}

	if fn.Name.Name != "add" {
		t.Fatalf(
			"got function name %q, want add",
			fn.Name.Name,
		)
	}

	if len(fn.Parameters) != 2 {
		t.Fatalf(
			"got %d parameters, want 2",
			len(fn.Parameters),
		)
	}

	if fn.Parameters[0].TypeName != "float" {
		t.Fatalf("unexpected first parameter type")
	}

	if fn.Parameters[0].Name.Name != "x" {
		t.Fatalf("unexpected first parameter name")
	}

	if fn.Parameters[1].TypeName != "float" {
		t.Fatalf("unexpected second parameter type")
	}

	if fn.Parameters[1].Name.Name != "y" {
		t.Fatalf("unexpected second parameter name")
	}

	if len(fn.Body.Statements) != 2 {
		t.Fatalf(
			"got %d body statements, want 2",
			len(fn.Body.Statements),
		)
	}
}

func TestParseSingleLineFunction(t *testing.T) {
	program, err := ParseSource(
		"add(x, y) => x + y",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	fn, ok := program.Statements[0].(*ast.FunctionDeclStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.FunctionDeclStmt",
			program.Statements[0],
		)
	}

	if len(fn.Parameters) != 2 {
		t.Fatalf(
			"got %d parameters, want 2",
			len(fn.Parameters),
		)
	}

	if len(fn.Body.Statements) != 1 {
		t.Fatalf(
			"got %d body statements, want 1",
			len(fn.Body.Statements),
		)
	}
}

func TestParseReturn(t *testing.T) {
	source := `foo(x) =>
    return x`

	program, err := ParseSource(source)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	fn := program.Statements[0].(*ast.FunctionDeclStmt)

	ret, ok := fn.Body.Statements[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.ReturnStmt",
			fn.Body.Statements[0],
		)
	}

	if ret.Value == nil {
		t.Fatal("expected return value")
	}
}

func TestParseCompoundAssignment(t *testing.T) {
	program, err := ParseSource(
		"x += 5",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	if stmt.Operator != "+=" {
		t.Fatalf(
			"got %q, want +=",
			stmt.Operator,
		)
	}
}

func TestParseTuple(t *testing.T) {
	program, err := ParseSource(
		"[high, low]",
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.ExpressionStmt)

	tuple, ok := stmt.Expression.(*ast.TupleExpr)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.TupleExpr",
			stmt.Expression,
		)
	}

	if len(tuple.Elements) != 2 {
		t.Fatalf(
			"got %d elements, want 2",
			len(tuple.Elements),
		)
	}
}

func TestParseErrors(t *testing.T) {
	cases := []string{
		"x =",
		"x = close[",
		"x = ta.sma(close,",
		"if close > open",
		"float x =",
		"foo(",
		"foo(x",
	}

	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := ParseSource(source)

			if err == nil {
				t.Fatalf(
					"expected parse error for %q",
					source,
				)
			}
		})
	}
}
func TestParseWhile(t *testing.T) {
	source := `while x < 10
    x = x + 1`

	program, err := ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(program.Statements) != 1 {
		t.Fatalf("got %d statements, want 1", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("got %T, want *ast.WhileStmt", program.Statements[0])
	}

	if stmt.Condition == nil {
		t.Fatal("expected while condition")
	}

	if stmt.Body == nil {
		t.Fatal("expected while body")
	}
}

func TestParseFor(t *testing.T) {
	source := `for i = 0 to 10
    x = i`

	program, err := ParseSource(source)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	if len(program.Statements) != 1 {
		t.Fatalf("got %d statements, want 1", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ForStmt)
	if !ok {
		t.Fatalf("got %T, want *ast.ForStmt", program.Statements[0])
	}

	if stmt.Variable == nil {
		t.Fatal("expected loop variable")
	}

	if stmt.From == nil {
		t.Fatal("expected from expression")
	}

	if stmt.To == nil {
		t.Fatal("expected to expression")
	}

	if stmt.Body == nil {
		t.Fatal("expected for body")
	}
}
