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

	tok := p.current()

	if tok.Type == token.KEYWORD {
		switch tok.Value {
		case "var", "varip":
			return p.parseVariableDeclaration()

		case "if":
			return p.parseIf()

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

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

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

func (p *Parser) parseVariableDeclaration() (ast.Stmt, error) {
	keyword := p.current()
	p.advance()

	if p.current().Type != token.IDENTIFIER {
		return nil, p.errorf("expected identifier after %q", keyword.Value)
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
		Keyword: stringValue(keyword.Value),
		Name:    name,
		Value:   value,
		SourceSpan: token.Span{
			Start: keyword.Span.Start,
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

	p.skipComments()

	if p.current().Type == token.KEYWORD &&
		stringValue(p.current().Value) == "else" {

		p.advance()

		elseBlock, err = p.parseIndentedBlock()
		if err != nil {
			return nil, err
		}
	}

	end := thenBlock.Span().End

	if elseBlock != nil {
		end = elseBlock.Span().End
	}

	return &ast.IfStmt{
		Condition: condition,
		Then:      thenBlock,
		Else:      elseBlock,
		SourceSpan: token.Span{
			Start: ifTok.Span.Start,
			End:   end,
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
			return nil, p.errorf("unterminated block: expected DEDENT")
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

		// Postfix: member access.
		if tok.Type == token.DOT {
			left, err = p.parseMember(left)
			if err != nil {
				return nil, err
			}

			continue
		}

		// Postfix: function call.
		if tok.Type == token.LPAREN {
			left, err = p.parseCall(left)
			if err != nil {
				return nil, err
			}

			continue
		}

		// Postfix: history/index.
		if tok.Type == token.LBRACKET {
			left, err = p.parseIndex(left)
			if err != nil {
				return nil, err
			}

			continue
		}

		// Ternary operator.
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
				return nil, p.errorf("expected ':' in conditional expression")
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
			return nil, p.errorf("invalid numeric token value")
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
			return nil, p.errorf("invalid boolean token value")
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
	start := p.current()
	p.advance()

	expr, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.current().Type != token.RPAREN {
		return nil, p.errorf("expected ')'")
	}

	end := p.current()
	p.advance()

	_ = start
	_ = end

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
	p.advance() // .

	if p.current().Type != token.IDENTIFIER {
		return nil, p.errorf("expected member name after '.'")
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
	p.advance() // (

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
		return nil, p.errorf("expected ')' after arguments")
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
	p.advance() // [

	index, err := p.parseExpression(0)
	if err != nil {
		return nil, err
	}

	if p.current().Type != token.RBRACKET {
		return nil, p.errorf("expected ']' after index")
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
		case token.COMMENT, token.NEWLINE, token.SEMICOLON:
			p.advance()

		default:
			return
		}
	}
}

func (p *Parser) consumeStatementSeparators() {
	for {
		switch p.current().Type {
		case token.COMMENT, token.NEWLINE, token.SEMICOLON:
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
