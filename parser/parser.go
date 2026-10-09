package parser

import (
	"fmt"

	"github.com/amitksharma00112/pinego/ast"
	"github.com/amitksharma00112/pinego/lexer"
	"github.com/amitksharma00112/pinego/token"
)

// Parser converts lexer tokens into an AST.
type Parser struct {
	tokens []token.Token
	pos    int
}

// New creates a parser from an existing token stream.
func New(tokens []token.Token) *Parser {
	return &Parser{
		tokens: tokens,
	}
}

// ParseSource lexes and parses Pine source in one call.
func ParseSource(source string) (*ast.Program, error) {
	tokens, err := lexSource(source)
	if err != nil {
		return nil, err
	}

	return New(tokens).Parse()
}

// Parse parses the complete token stream.
func (p *Parser) Parse() (*ast.Program, error) {
	program := &ast.Program{}

	for {
		p.skipTrivia()

		if p.current().Type == token.EOF {
			break
		}

		if p.current().Type == token.DEDENT {
			return nil, p.errorf("unexpected DEDENT")
		}

		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}

		p.consumeStatementSeparators()
	}

	if len(program.Statements) > 0 {
		program.SourceSpan = token.Span{
			Start: program.Statements[0].Span().Start,
			End:   program.Statements[len(program.Statements)-1].Span().End,
		}
	}

	return program, nil
}

func lexSource(source string) ([]token.Token, error) {
	return lexer.SafeTokenize(source)
}

// -----------------------------------------------------------------------------
// Statements
// -----------------------------------------------------------------------------

func (p *Parser) parseStatement() (ast.Stmt, error) {
	p.skipTrivia()

	// Function declaration:
	//
	// foo(x) => x + 1
	// foo(float x) => x + 1
	// foo(x)
	// =>
	//     x + 1
	if p.isFunctionDeclarationStart() {
		return p.parseFunctionDeclaration()
	}

	// Typed variable declaration:
	//
	// float x = 10
	// int length = 20
	//
	// var float x = 10 is handled by parseVariableDeclaration().
	if p.isTypedVariableDeclarationStart() {
		return p.parseTypedVariableDeclaration()
	}

	tok := p.current()

	if tok.Type == token.KEYWORD {
		switch tok.Value {

		case "var", "varip":
			return p.parseVariableDeclaration()

		case "if":
			return p.parseIf()
		case "while":
			return p.parseWhile()
		case "for":
			return p.parseFor()

		case "return":
			return p.parseReturn()

		case "break":
			return p.parseBreak()

		case "continue":
			return p.parseContinue()

		default:
			return nil, p.errorf(
				"unexpected keyword %q",
				tok.Value,
			)
		}
	}

	// Normal expression.
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	// Assignment:
	//
	// x = 10
	// x := 10
	// x += 5
	if p.isAssignmentOperator(p.current()) {
		op := p.current()
		p.advance()

		value, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		return &ast.AssignmentStmt{
			Target:   expr,
			Operator: opString(op),
			Value:    value,
			SourceSpan: token.Span{
				Start: expr.Span().Start,
				End:   value.Span().End,
			},
		}, nil
	}

	return &ast.ExpressionStmt{
		Expression: expr,
		SourceSpan: expr.Span(),
	}, nil
}

func (p *Parser) isFunctionDeclarationStart() bool {
	if p.current().Type != token.IDENTIFIER {
		return false
	}

	i := p.pos + 1

	if i >= len(p.tokens) || p.tokens[i].Type != token.LPAREN {
		return false
	}

	depth := 1
	i++

	for i < len(p.tokens) {
		switch p.tokens[i].Type {
		case token.LPAREN:
			depth++

		case token.RPAREN:
			depth--

			if depth == 0 {
				i++

				if i < len(p.tokens) &&
					p.tokens[i].Type == token.OPERATOR &&
					stringValue(p.tokens[i].Value) == "=>" {
					return true
				}

				return false
			}
		}

		i++
	}

	return false
}

func (p *Parser) parseFunctionDeclaration() (ast.Stmt, error) {
	start := p.current()

	nameTok := p.current()
	p.advance()

	if p.current().Type != token.LPAREN {
		return nil, p.errorf("expected '(' after function name")
	}
	p.advance()

	parameters := make([]*ast.Parameter, 0)

	if p.current().Type != token.RPAREN {
		for {
			param, err := p.parseParameter()
			if err != nil {
				return nil, err
			}

			parameters = append(parameters, param)

			if p.current().Type != token.COMMA {
				break
			}

			p.advance()

			if p.current().Type == token.RPAREN {
				break
			}
		}
	}

	if p.current().Type != token.RPAREN {
		return nil, p.errorf("expected ')' after parameters")
	}
	p.advance()

	if p.current().Type != token.OPERATOR ||
		stringValue(p.current().Value) != "=>" {
		return nil, p.errorf("expected '=>' after function signature")
	}
	p.advance()

	body, err := p.parseFunctionBody()
	if err != nil {
		return nil, err
	}

	return &ast.FunctionDeclStmt{
		Name: &ast.Identifier{
			Name:       stringValue(nameTok.Value),
			SourceSpan: nameTok.Span,
		},
		Parameters: parameters,
		Body:       body,
		SourceSpan: token.Span{
			Start: start.Span.Start,
			End:   body.Span().End,
		},
	}, nil
}

func (p *Parser) parseParameter() (*ast.Parameter, error) {
	start := p.current()

	var typeName string

	if p.current().Type == token.IDENTIFIER {
		first := p.current()
		p.advance()

		// Typed parameter: float x
		if p.current().Type == token.IDENTIFIER {
			typeName = stringValue(first.Value)

			nameTok := p.current()
			p.advance()

			return &ast.Parameter{
				TypeName: typeName,
				Name: &ast.Identifier{
					Name:       stringValue(nameTok.Value),
					SourceSpan: nameTok.Span,
				},
				SourceSpan: token.Span{
					Start: start.Span.Start,
					End:   nameTok.Span.End,
				},
			}, nil
		}

		// Untyped parameter: x
		return &ast.Parameter{
			Name: &ast.Identifier{
				Name:       stringValue(first.Value),
				SourceSpan: first.Span,
			},
			SourceSpan: first.Span,
		}, nil
	}

	return nil, p.errorf("expected parameter name")
}

func (p *Parser) parseFunctionBody() (*ast.BlockStmt, error) {
	// Multi-line function:
	//
	// f(x) =>
	//     x + 1
	if p.current().Type == token.NEWLINE {
		return p.parseIndentedBlock()
	}

	// Single-line function:
	//
	// f(x) => x + 1
	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	stmt := &ast.ExpressionStmt{
		Expression: expr,
		SourceSpan: expr.Span(),
	}

	return &ast.BlockStmt{
		Statements: []ast.Stmt{stmt},
		SourceSpan: expr.Span(),
	}, nil
}

func (p *Parser) parseVariableDeclaration() (ast.Stmt, error) {
	keyword := p.current()
	p.advance()

	typeName := ""

	// Support:
	//
	// var float x = 10
	// var int x = 20
	if p.current().Type == token.IDENTIFIER &&
		p.peek(1).Type == token.IDENTIFIER &&
		p.isAssignmentOperator(p.peek(2)) {

		typeName = stringValue(p.current().Value)
		p.advance()
	}

	if p.current().Type != token.IDENTIFIER {
		return nil, p.errorf(
			"expected identifier after %q",
			keyword.Value,
		)
	}

	nameTok := p.current()

	name := &ast.Identifier{
		Name:       stringValue(nameTok.Value),
		SourceSpan: nameTok.Span,
	}

	p.advance()

	if !p.isAssignmentOperator(p.current()) {
		return nil, p.errorf(
			"expected assignment operator after variable %q",
			name.Name,
		)
	}

	p.advance()

	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &ast.VariableDeclStmt{
		Keyword:  stringValue(keyword.Value),
		TypeName: typeName,
		Name:     name,
		Value:    value,
		SourceSpan: token.Span{
			Start: keyword.Span.Start,
			End:   value.Span().End,
		},
	}, nil
}

func (p *Parser) isTypedVariableDeclarationStart() bool {
	if p.current().Type != token.IDENTIFIER {
		return false
	}

	if p.peek(1).Type != token.IDENTIFIER {
		return false
	}

	return p.isAssignmentOperator(p.peek(2))
}

func (p *Parser) parseTypedVariableDeclaration() (ast.Stmt, error) {
	typeTok := p.current()
	p.advance()

	if p.current().Type != token.IDENTIFIER {
		return nil, p.errorf(
			"expected variable name after type %q",
			typeTok.Value,
		)
	}

	nameTok := p.current()

	name := &ast.Identifier{
		Name:       stringValue(nameTok.Value),
		SourceSpan: nameTok.Span,
	}

	p.advance()

	if !p.isAssignmentOperator(p.current()) {
		return nil, p.errorf(
			"expected assignment operator after variable %q",
			name.Name,
		)
	}

	p.advance()

	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &ast.VariableDeclStmt{
		TypeName: stringValue(typeTok.Value),
		Name:     name,
		Value:    value,
		SourceSpan: token.Span{
			Start: typeTok.Span.Start,
			End:   value.Span().End,
		},
	}, nil
}

func (p *Parser) parseIf() (ast.Stmt, error) {
	ifTok := p.current()
	p.advance()

	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	thenBlock, err := p.parseIndentedBlock()
	if err != nil {
		return nil, err
	}

	var elseBlock *ast.BlockStmt
	var elseIf *ast.IfStmt

	p.skipComments()

	if p.current().Type == token.KEYWORD &&
		stringValue(p.current().Value) == "else" {

		p.advance()
		p.skipComments()

		// else if
		if p.current().Type == token.KEYWORD &&
			stringValue(p.current().Value) == "if" {

			stmt, err := p.parseIf()
			if err != nil {
				return nil, err
			}

			var ok bool
			elseIf, ok = stmt.(*ast.IfStmt)
			if !ok {
				return nil, p.errorf("expected if statement after else")
			}

		} else {
			// normal else
			elseBlock, err = p.parseIndentedBlock()
			if err != nil {
				return nil, err
			}
		}
	}

	end := thenBlock.Span().End

	if elseIf != nil {
		end = elseIf.Span().End
	} else if elseBlock != nil {
		end = elseBlock.Span().End
	}

	return &ast.IfStmt{
		Condition: condition,
		Then:      thenBlock,
		ElseIf:    elseIf,
		Else:      elseBlock,
		SourceSpan: token.Span{
			Start: ifTok.Span.Start,
			End:   end,
		},
	}, nil
}

func (p *Parser) parseWhile() (ast.Stmt, error) {
	whileTok := p.current()
	p.advance()

	condition, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	body, err := p.parseIndentedBlock()
	if err != nil {
		return nil, err
	}

	return &ast.WhileStmt{
		Condition: condition,
		Body:      body,
		SourceSpan: token.Span{
			Start: whileTok.Span.Start,
			End:   body.Span().End,
		},
	}, nil
}
func (p *Parser) parseFor() (ast.Stmt, error) {
	forTok := p.current()
	p.advance()

	if p.current().Type != token.IDENTIFIER {
		return nil, p.errorf("expected loop variable after 'for'")
	}

	varTok := p.current()
	p.advance()

	variable := &ast.Identifier{
		Name:       stringValue(varTok.Value),
		SourceSpan: varTok.Span,
	}

	if !p.isAssignmentOperator(p.current()) ||
		stringValue(p.current().Value) != "=" {
		return nil, p.errorf("expected '=' after loop variable")
	}
	p.advance()

	from, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.current().Type != token.KEYWORD ||
		stringValue(p.current().Value) != "to" {
		return nil, p.errorf("expected 'to' in for loop")
	}
	p.advance()

	to, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	var step ast.Expr

	if p.current().Type == token.KEYWORD &&
		stringValue(p.current().Value) == "by" {
		p.advance()

		step, err = p.parseExpression(0)
		if err != nil {
			return nil, err
		}
	}

	body, err := p.parseIndentedBlock()
	if err != nil {
		return nil, err
	}

	return &ast.ForStmt{
		Variable: variable,
		From:     from,
		To:       to,
		Step:     step,
		Body:     body,
		SourceSpan: token.Span{
			Start: forTok.Span.Start,
			End:   body.Span().End,
		},
	}, nil
}

func (p *Parser) parseIndentedBlock() (*ast.BlockStmt, error) {
	p.skipComments()

	if p.current().Type != token.NEWLINE {
		return nil, p.errorf("expected NEWLINE before block")
	}

	p.advance()

	p.skipComments()

	if p.current().Type != token.INDENT {
		return nil, p.errorf("expected INDENT before block")
	}

	start := p.current().Span.Start
	p.advance()

	block := &ast.BlockStmt{}

	for {
		p.skipComments()

		if p.current().Type == token.DEDENT {
			end := p.current().Span.End
			p.advance()

			block.SourceSpan = token.Span{
				Start: start,
				End:   end,
			}

			return block, nil
		}

		if p.current().Type == token.EOF {
			return nil, p.errorf(
				"unterminated block: expected DEDENT",
			)
		}

		if p.current().Type == token.NEWLINE ||
			p.current().Type == token.SEMICOLON {

			p.advance()
			continue
		}

		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}

		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}

		p.consumeStatementSeparators()
	}
}

func (p *Parser) parseReturn() (ast.Stmt, error) {
	start := p.current()
	p.advance()

	p.skipComments()

	if p.current().Type == token.NEWLINE ||
		p.current().Type == token.DEDENT ||
		p.current().Type == token.EOF ||
		p.current().Type == token.SEMICOLON {

		return &ast.ReturnStmt{
			SourceSpan: start.Span,
		}, nil
	}

	value, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	return &ast.ReturnStmt{
		Value: value,
		SourceSpan: token.Span{
			Start: start.Span.Start,
			End:   value.Span().End,
		},
	}, nil
}

func (p *Parser) parseBreak() (ast.Stmt, error) {
	tok := p.current()
	p.advance()

	return &ast.BreakStmt{
		SourceSpan: tok.Span,
	}, nil
}

func (p *Parser) parseContinue() (ast.Stmt, error) {
	tok := p.current()
	p.advance()

	return &ast.ContinueStmt{
		SourceSpan: tok.Span,
	}, nil
}

// -----------------------------------------------------------------------------
// Expressions
// -----------------------------------------------------------------------------

func (p *Parser) parseExpression(minPrecedence int) (ast.Expr, error) {
	left, err := p.parsePrefix()
	if err != nil {
		return nil, err
	}

	for {
		p.skipComments()

		tok := p.current()

		// Member access:
		//
		// ta.sma
		if tok.Type == token.DOT {
			left, err = p.parseMember(left)
			if err != nil {
				return nil, err
			}

			continue
		}

		// Function call:
		//
		// foo(a, b)
		if tok.Type == token.LPAREN {
			left, err = p.parseCall(left)
			if err != nil {
				return nil, err
			}

			continue
		}

		// History/index:
		//
		// close[1]
		if tok.Type == token.LBRACKET {
			left, err = p.parseIndex(left)
			if err != nil {
				return nil, err
			}

			continue
		}

		// Ternary:
		//
		// condition ? a : b
		if tok.Type == token.OPERATOR &&
			stringValue(tok.Value) == "?" {

			if minPrecedence > 0 {
				break
			}

			p.advance()

			thenExpr, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}

			if p.current().Type != token.COLON {
				return nil, p.errorf(
					"expected ':' in conditional expression",
				)
			}

			p.advance()

			elseExpr, err := p.parseExpression(0)
			if err != nil {
				return nil, err
			}

			left = &ast.ConditionalExpr{
				Condition: left,
				Then:      thenExpr,
				Else:      elseExpr,
				SourceSpan: token.Span{
					Start: left.Span().Start,
					End:   elseExpr.Span().End,
				},
			}

			continue
		}

		precedence, ok := p.binaryPrecedence(tok)
		if !ok || precedence <= minPrecedence {
			break
		}

		operator := stringValue(tok.Value)
		p.advance()

		right, err := p.parseExpression(precedence)
		if err != nil {
			return nil, err
		}

		left = &ast.BinaryExpr{
			Left:     left,
			Operator: operator,
			Right:    right,
			SourceSpan: token.Span{
				Start: left.Span().Start,
				End:   right.Span().End,
			},
		}
	}

	return left, nil
}

func (p *Parser) parsePrefix() (ast.Expr, error) {
	tok := p.current()

	switch tok.Type {

	case token.IDENTIFIER:
		p.advance()

		return &ast.Identifier{
			Name:       stringValue(tok.Value),
			SourceSpan: tok.Span,
		}, nil

	case token.NUMBER:
		p.advance()

		value, ok := tok.Value.(float64)
		if !ok {
			return nil, p.errorf(
				"invalid numeric token value",
			)
		}

		return &ast.NumberLiteral{
			Value:      value,
			Raw:        tok.Raw,
			SourceSpan: tok.Span,
		}, nil

	case token.STRING:
		p.advance()

		return &ast.StringLiteral{
			Value:      stringValue(tok.Value),
			Raw:        tok.Raw,
			SourceSpan: tok.Span,
		}, nil

	case token.BOOLEAN:
		p.advance()

		value, ok := tok.Value.(bool)
		if !ok {
			return nil, p.errorf(
				"invalid boolean token value",
			)
		}

		return &ast.BoolLiteral{
			Value:      value,
			SourceSpan: tok.Span,
		}, nil

	case token.NA:
		p.advance()

		return &ast.NALiteral{
			SourceSpan: tok.Span,
		}, nil

	case token.COLOR:
		p.advance()

		return &ast.ColorLiteral{
			Value:      stringValue(tok.Value),
			SourceSpan: tok.Span,
		}, nil

	case token.OPERATOR:
		operator := stringValue(tok.Value)

		if operator == "+" ||
			operator == "-" ||
			operator == "!" {

			p.advance()

			operand, err := p.parseExpression(7)
			if err != nil {
				return nil, err
			}

			return &ast.UnaryExpr{
				Operator: operator,
				Operand:  operand,
				SourceSpan: token.Span{
					Start: tok.Span.Start,
					End:   operand.Span().End,
				},
			}, nil
		}

	case token.KEYWORD:
		if stringValue(tok.Value) == "not" {
			p.advance()

			operand, err := p.parseExpression(7)
			if err != nil {
				return nil, err
			}

			return &ast.UnaryExpr{
				Operator: "not",
				Operand:  operand,
				SourceSpan: token.Span{
					Start: tok.Span.Start,
					End:   operand.Span().End,
				},
			}, nil
		}

	case token.LPAREN:
		return p.parseGroupedExpression()

	case token.LBRACKET:
		return p.parseTupleExpression()
	}

	return nil, p.errorf(
		"unexpected token %s (%v)",
		tok.Type,
		tok.Value,
	)
}

func (p *Parser) parseGroupedExpression() (ast.Expr, error) {
	p.advance()

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.current().Type != token.RPAREN {
		return nil, p.errorf("expected ')'")
	}

	p.advance()

	return expr, nil
}

func (p *Parser) parseTupleExpression() (ast.Expr, error) {
	start := p.current()
	p.advance()

	tuple := &ast.TupleExpr{
		Elements: make([]ast.Expr, 0),
	}

	if p.current().Type == token.RBRACKET {
		end := p.current()
		p.advance()

		tuple.SourceSpan = token.Span{
			Start: start.Span.Start,
			End:   end.Span.End,
		}

		return tuple, nil
	}

	for {
		expr, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		tuple.Elements = append(tuple.Elements, expr)

		if p.current().Type != token.COMMA {
			break
		}

		p.advance()

		if p.current().Type == token.RBRACKET {
			break
		}
	}

	if p.current().Type != token.RBRACKET {
		return nil, p.errorf("expected ']'")
	}

	end := p.current()
	p.advance()

	tuple.SourceSpan = token.Span{
		Start: start.Span.Start,
		End:   end.Span.End,
	}

	return tuple, nil
}

func (p *Parser) parseMember(object ast.Expr) (ast.Expr, error) {
	p.advance()

	if p.current().Type != token.IDENTIFIER {
		return nil, p.errorf(
			"expected member name after '.'",
		)
	}

	member := p.current()

	result := &ast.MemberExpr{
		Object: object,
		Member: stringValue(member.Value),
		SourceSpan: token.Span{
			Start: object.Span().Start,
			End:   member.Span.End,
		},
	}

	p.advance()

	return result, nil
}

func (p *Parser) parseCall(callee ast.Expr) (ast.Expr, error) {
	p.advance()

	call := &ast.CallExpr{
		Callee:    callee,
		Arguments: make([]ast.Expr, 0),
	}

	if p.current().Type == token.RPAREN {
		end := p.current()
		p.advance()

		call.SourceSpan = token.Span{
			Start: callee.Span().Start,
			End:   end.Span.End,
		}

		return call, nil
	}

	for {
		arg, err := p.parseExpression(0)
		if err != nil {
			return nil, err
		}

		call.Arguments = append(call.Arguments, arg)

		if p.current().Type != token.COMMA {
			break
		}

		p.advance()

		if p.current().Type == token.RPAREN {
			break
		}
	}

	if p.current().Type != token.RPAREN {
		return nil, p.errorf(
			"expected ')' after arguments",
		)
	}

	end := p.current()
	p.advance()

	call.SourceSpan = token.Span{
		Start: callee.Span().Start,
		End:   end.Span.End,
	}

	return call, nil
}

func (p *Parser) parseIndex(object ast.Expr) (ast.Expr, error) {
	p.advance()

	index, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.current().Type != token.RBRACKET {
		return nil, p.errorf(
			"expected ']' after index",
		)
	}

	end := p.current()
	p.advance()

	return &ast.IndexExpr{
		Object: object,
		Index:  index,
		SourceSpan: token.Span{
			Start: object.Span().Start,
			End:   end.Span.End,
		},
	}, nil
}

// -----------------------------------------------------------------------------
// Operators
// -----------------------------------------------------------------------------

func (p *Parser) binaryPrecedence(tok token.Token) (int, bool) {
	value := stringValue(tok.Value)

	if tok.Type == token.KEYWORD {
		switch value {
		case "or":
			return 1, true

		case "and":
			return 2, true
		}

		return 0, false
	}

	if tok.Type != token.OPERATOR {
		return 0, false
	}

	switch value {
	case "==", "!=":
		return 3, true

	case "<", "<=", ">", ">=":
		return 4, true

	case "+", "-":
		return 5, true

	case "*", "/", "%":
		return 6, true
	}

	return 0, false
}

func (p *Parser) isAssignmentOperator(tok token.Token) bool {
	if tok.Type != token.OPERATOR {
		return false
	}

	switch stringValue(tok.Value) {
	case "=", ":=", "+=", "-=", "*=", "/=", "%=":
		return true

	default:
		return false
	}
}

// -----------------------------------------------------------------------------
// Token helpers
// -----------------------------------------------------------------------------

func (p *Parser) current() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{
			Type: token.EOF,
		}
	}

	return p.tokens[p.pos]
}

func (p *Parser) peek(offset int) token.Token {
	index := p.pos + offset

	if index < 0 || index >= len(p.tokens) {
		return token.Token{
			Type: token.EOF,
		}
	}

	return p.tokens[index]
}

func (p *Parser) advance() token.Token {
	tok := p.current()

	if p.pos < len(p.tokens) {
		p.pos++
	}

	return tok
}

func (p *Parser) skipComments() {
	for p.current().Type == token.COMMENT {
		p.advance()
	}
}

func (p *Parser) skipTrivia() {
	for {
		switch p.current().Type {
		case token.COMMENT,
			token.NEWLINE,
			token.SEMICOLON:

			p.advance()

		default:
			return
		}
	}
}

func (p *Parser) consumeStatementSeparators() {
	for {
		switch p.current().Type {
		case token.COMMENT,
			token.NEWLINE,
			token.SEMICOLON:

			p.advance()

		default:
			return
		}
	}
}

func (p *Parser) errorf(format string, args ...any) error {
	tok := p.current()

	return fmt.Errorf(
		"parse error at %d:%d: %s",
		tok.Span.Start.Line,
		tok.Span.Start.Column,
		fmt.Sprintf(format, args...),
	)
}

func opString(tok token.Token) string {
	return stringValue(tok.Value)
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}

	if s, ok := value.(string); ok {
		return s
	}

	return fmt.Sprint(value)
}
