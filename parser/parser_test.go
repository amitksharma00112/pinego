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
		t.Fatalf("got %d statements, want 1", len(program.Statements))
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
			"got operator %q, want %q",
			stmt.Operator,
			"=",
		)
	}

	target, ok := stmt.Target.(*ast.Identifier)
	if !ok {
		t.Fatalf("got target %T, want *ast.Identifier", stmt.Target)
	}

	if target.Name != "x" {
		t.Fatalf("got target %q, want %q", target.Name, "x")
	}

	value, ok := stmt.Value.(*ast.NumberLiteral)
	if !ok {
		t.Fatalf("got value %T, want *ast.NumberLiteral", stmt.Value)
	}

	if value.Value != 10 {
		t.Fatalf("got %v, want 10", value.Value)
	}
}

func TestParseOperatorPrecedence(t *testing.T) {
	program, err := ParseSource("x = 10 + 20 * 5")

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)
	add, ok := stmt.Value.(*ast.BinaryExpr)

	if !ok {
		t.Fatalf("got %T, want *ast.BinaryExpr", stmt.Value)
	}

	if add.Operator != "+" {
		t.Fatalf("got operator %q, want +", add.Operator)
	}

	mul, ok := add.Right.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf(
			"got right operand %T, want *ast.BinaryExpr",
			add.Right,
		)
	}

	if mul.Operator != "*" {
		t.Fatalf("got operator %q, want *", mul.Operator)
	}
}

func TestParseMemberAndCall(t *testing.T) {
	program, err := ParseSource("x = ta.sma(close, 20)")

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)
	call, ok := stmt.Value.(*ast.CallExpr)

	if !ok {
		t.Fatalf("got %T, want *ast.CallExpr", stmt.Value)
	}

	member, ok := call.Callee.(*ast.MemberExpr)
	if !ok {
		t.Fatalf("got %T, want *ast.MemberExpr", call.Callee)
	}

	if member.Member != "sma" {
		t.Fatalf("got member %q, want sma", member.Member)
	}

	if len(call.Arguments) != 2 {
		t.Fatalf(
			"got %d arguments, want 2",
			len(call.Arguments),
		)
	}
}

func TestParseHistoryOperator(t *testing.T) {
	program, err := ParseSource("x = close[1]")

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	index, ok := stmt.Value.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("got %T, want *ast.IndexExpr", stmt.Value)
	}

	if index.Index.(*ast.NumberLiteral).Value != 1 {
		t.Fatalf("unexpected index")
	}
}

func TestParseTernary(t *testing.T) {
	program, err := ParseSource("x = close > open ? high : low")

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

	if len(program.Statements) != 1 {
		t.Fatalf(
			"got %d statements, want 1",
			len(program.Statements),
		)
	}

	ifStmt, ok := program.Statements[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf(
			"got %T, want *ast.IfStmt",
			program.Statements[0],
		)
	}

	if len(ifStmt.Then.Statements) != 1 {
		t.Fatalf("unexpected then block")
	}

	if ifStmt.Else == nil {
		t.Fatal("expected else block")
	}

	if len(ifStmt.Else.Statements) != 1 {
		t.Fatalf("unexpected else block")
	}
}

func TestParseUnaryAndLogical(t *testing.T) {
	program, err := ParseSource(
		`x = not close > open and volume > 100000`,
	)

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	root, ok := stmt.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("got %T, want *ast.BinaryExpr", stmt.Value)
	}

	if root.Operator != "and" {
		t.Fatalf("got root operator %q, want and", root.Operator)
	}
}

func TestParseVariableDeclaration(t *testing.T) {
	program, err := ParseSource("var x = 10")

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
		t.Fatalf("got %q, want var", stmt.Keyword)
	}

	if stmt.Name.Name != "x" {
		t.Fatalf("got %q, want x", stmt.Name.Name)
	}
}

func TestParseCompoundAssignment(t *testing.T) {
	program, err := ParseSource("x += 5")

	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	stmt := program.Statements[0].(*ast.AssignmentStmt)

	if stmt.Operator != "+=" {
		t.Fatalf("got %q, want +=", stmt.Operator)
	}
}

func TestParseTuple(t *testing.T) {
	program, err := ParseSource("[high, low]")

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
