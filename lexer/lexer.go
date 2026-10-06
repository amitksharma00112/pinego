package lexer

import (
	"fmt"
	"strings"

	"github.com/amitksharma00112/pinego/token"
)

const tabWidth = 4

// Lexer converts Pine Script source code into lexical tokens.
//
// It is intentionally independent of the parser. The lexer is responsible
// only for:
//   - tokenization
//   - source positions
//   - indentation / dedentation
//   - logical line wrapping
//   - grouped-expression tracking
type Lexer struct {
	source       string
	pos          int
	line         int
	column       int
	tokens       []token.Token
	indentStack  []int
	atLineStart  bool
	parenDepth   int
	bracketDepth int
	braceDepth   int
	pendingWrap  *token.WrapInfo
}

// New creates a Pine Script lexer.
func New(source string) *Lexer {
	return &Lexer{
		source:      source,
		line:        1,
		column:      1,
		indentStack: []int{0},
		atLineStart: true,
		tokens:      make([]token.Token, 0, 128),
	}
}

// Tokenize lexes the complete source.
func (l *Lexer) Tokenize() ([]token.Token, error) {
	for l.pos < len(l.source) {
		ch := l.peek(0)

		// Windows line ending.
		if ch == '\r' {
			if l.peek(1) == '\n' {
				l.advance()
				continue
			}

			l.advance()
			l.emit(token.NEWLINE, "\n", "\r", l.currentIndent(), "")
			l.atLineStart = true
			l.line++
			l.column = 1
			continue
		}

		// Newline.
		if ch == '\n' {
			l.handleNewline()
			continue
		}

		// Indentation / wrapping at logical line start.
		if l.atLineStart {
			l.handleIndentation()
			l.atLineStart = false
			continue
		}

		// Inline whitespace.
		if ch == ' ' || ch == '\t' {
			l.advance()
			continue
		}

		// Comments.
		if ch == '/' && l.peek(1) == '/' {
			l.readComment()
			continue
		}

		// Strings.
		if ch == '"' || ch == '\'' {
			if err := l.readString(); err != nil {
				return nil, err
			}
			continue
		}

		// Color literal.
		if ch == '#' {
			if err := l.readColor(); err != nil {
				return nil, err
			}
			continue
		}

		// Numbers.
		if l.isDigit(ch) {
			if err := l.readNumber(); err != nil {
				return nil, err
			}
			continue
		}

		// Leading-dot number.
		if ch == '.' && l.isDigit(l.peek(1)) {
			if err := l.readNumber(); err != nil {
				return nil, err
			}
			continue
		}

		// Identifier / keyword / literal.
		if l.isIdentifierStart(ch) {
			l.readIdentifier()
			continue
		}

		// Operators / punctuation.
		if l.readOperatorOrPunctuation() {
			continue
		}

		return nil, fmt.Errorf(
			"unexpected character %q at %d:%d",
			ch,
			l.line,
			l.column,
		)
	}

	// Close open indentation levels.
	for len(l.indentStack) > 1 {
		l.indentStack = l.indentStack[:len(l.indentStack)-1]
		l.emit(token.DEDENT, "", "", l.currentIndent(), "")
	}

	l.emit(token.EOF, "", "", l.currentIndent(), "")

	return l.tokens, nil
}

func (l *Lexer) handleNewline() {
	// Newlines inside grouped expressions are suppressed.
	if l.parenDepth > 0 || l.bracketDepth > 0 || l.braceDepth > 0 {
		l.advance()
		l.line++
		l.column = 1
		return
	}

	start := l.position()

	l.advance()

	l.tokens = append(l.tokens, token.Token{
		Type:  token.NEWLINE,
		Value: "\n",
		Raw:   "\n",
		Span: token.Span{
			Start: start,
			End:   l.position(),
		},
		Indent: l.currentIndent(),
	})

	l.atLineStart = true
	l.line++
	l.column = 1
}

func (l *Lexer) handleIndentation() {
	width := 0
	startColumn := l.column

	for l.pos < len(l.source) {
		switch l.peek(0) {
		case ' ':
			width++
			l.advance()

		case '\t':
			width += tabWidth
			l.advance()

		default:
			goto indentationDone
		}
	}

indentationDone:

	// Blank line.
	if l.peek(0) == '\n' || l.peek(0) == '\r' || l.peek(0) == 0 {
		return
	}

	// Comment-only line.
	if l.peek(0) == '/' && l.peek(1) == '/' {
		return
	}

	// Non-multiple-of-four indentation is a wrapped continuation.
	if width%tabWidth != 0 {
		l.joinWithPreviousLine(width, startColumn)
		return
	}

	// A full-width indentation can still be a continuation when the
	// previous logical token requires a right-hand side.
	if l.isContinuationFromPreviousToken() {
		l.joinWithPreviousLine(width, startColumn)
		return
	}

	level := width / tabWidth
	currentLevel := len(l.indentStack) - 1

	if level > currentLevel {
		// Pine block nesting increases one level at a time.
		if level > currentLevel+1 {
			panicError := fmt.Sprintf(
				"indentation error at %d:%d: indentation %d columns jumps more than one level",
				l.line,
				l.column,
				width,
			)

			l.fail(panicError)
			return
		}

		l.indentStack = append(l.indentStack, level)
		l.emit(token.INDENT, "", "", level, "")
		return
	}

	if level < currentLevel {
		for len(l.indentStack) > 1 &&
			l.indentStack[len(l.indentStack)-1] > level {

			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			l.emit(token.DEDENT, "", "", l.currentIndent(), "")
		}

		if l.currentIndent() != level {
			l.fail(
				fmt.Sprintf(
					"indentation error at %d:%d: misaligned dedent",
					l.line,
					l.column,
				),
			)
		}
	}
}

func (l *Lexer) joinWithPreviousLine(width, column int) {
	// Remove layout tokens between the previous logical statement and
	// the continuation.
	for len(l.tokens) > 0 {
		last := l.tokens[len(l.tokens)-1]

		if last.Type != token.NEWLINE && last.Type != token.COMMENT {
			break
		}

		l.tokens = l.tokens[:len(l.tokens)-1]
	}

	if len(l.tokens) == 0 {
		return
	}

	fromLine := l.tokens[len(l.tokens)-1].Span.End.Line

	l.pendingWrap = &token.WrapInfo{
		Width:    width,
		FromLine: fromLine,
		Column:   column,
	}
}

func (l *Lexer) isContinuationFromPreviousToken() bool {
	for i := len(l.tokens) - 1; i >= 0; i-- {
		t := l.tokens[i]

		if t.Type == token.NEWLINE || t.Type == token.COMMENT {
			continue
		}

		// Any operator except => requires another expression.
		if t.Type == token.OPERATOR {
			return t.Value != "=>"
		}

		if t.Type == token.COMMA || t.Type == token.COLON {
			return true
		}

		if t.Type == token.KEYWORD {
			switch t.Value {
			case "and", "or":
				return true
			}
		}

		return false
	}

	return false
}

func (l *Lexer) readComment() {
	start := l.position()
	startPos := l.pos

	l.advance()
	l.advance()

	for l.pos < len(l.source) {
		if l.peek(0) == '\n' || l.peek(0) == '\r' {
			break
		}

		l.advance()
	}

	raw := l.source[startPos:l.pos]
	value := strings.TrimSpace(strings.TrimPrefix(raw, "//"))

	l.emitAt(
		token.COMMENT,
		value,
		raw,
		l.currentIndent(),
		"",
		start,
	)
}

func (l *Lexer) readString() error {
	quote := l.peek(0)

	// Triple quoted string.
	if l.peek(1) == quote && l.peek(2) == quote {
		return l.readMultilineString()
	}

	start := l.position()
	startColumn := l.column
	startPos := l.pos

	l.advance()

	var value strings.Builder

	for l.pos < len(l.source) {
		ch := l.peek(0)

		if ch == quote {
			l.advance()

			l.emitAt(
				token.STRING,
				value.String(),
				l.source[startPos:l.pos],
				l.currentIndent(),
				"",
				start,
			)

			l.tokens[len(l.tokens)-1].StartColumn = startColumn
			return nil
		}

		if ch == '\n' || ch == '\r' {
			return fmt.Errorf(
				"unterminated string at %d:%d",
				start.Line,
				start.Column,
			)
		}

		if ch == '\\' {
			escaped, err := l.readEscape()
			if err != nil {
				return err
			}

			value.WriteByte(escaped)
			continue
		}

		value.WriteByte(l.advance())
	}

	return fmt.Errorf(
		"unterminated string at %d:%d",
		start.Line,
		start.Column,
	)
}

func (l *Lexer) readMultilineString() error {
	quote := l.peek(0)

	start := l.position()
	startPos := l.pos

	l.advance()
	l.advance()
	l.advance()

	var value strings.Builder

	for l.pos < len(l.source) {
		if l.peek(0) == quote &&
			l.peek(1) == quote &&
			l.peek(2) == quote {

			l.advance()
			l.advance()
			l.advance()

			l.emitAt(
				token.STRING,
				value.String(),
				l.source[startPos:l.pos],
				l.currentIndent(),
				"",
				start,
			)

			return nil
		}

		if l.peek(0) == '\r' {
			if l.peek(1) == '\n' {
				l.advance()
			}

			l.advance()

			value.WriteByte('\n')
			l.line++
			l.column = 1
			continue
		}

		if l.peek(0) == '\n' {
			l.advance()

			value.WriteByte('\n')
			l.line++
			l.column = 1
			continue
		}

		if l.peek(0) == '\\' {
			escaped, err := l.readEscape()
			if err != nil {
				return err
			}

			value.WriteByte(escaped)
			continue
		}

		value.WriteByte(l.advance())
	}

	return fmt.Errorf(
		"unterminated multiline string at %d:%d",
		start.Line,
		start.Column,
	)
}

func (l *Lexer) readEscape() (byte, error) {
	l.advance() // backslash

	if l.pos >= len(l.source) {
		return 0, fmt.Errorf(
			"unterminated escape at %d:%d",
			l.line,
			l.column,
		)
	}

	switch ch := l.advance(); ch {
	case 'n':
		return '\n', nil

	case 't':
		return '\t', nil

	case 'r':
		return '\r', nil

	case '\\':
		return '\\', nil

	case '\'':
		return '\'', nil

	case '"':
		return '"', nil

	default:
		return ch, nil
	}
}

func (l *Lexer) readColor() error {
	start := l.position()
	startPos := l.pos

	l.advance() // #

	digits := 0

	for l.pos < len(l.source) && digits < 8 {
		ch := l.peek(0)

		if l.isHexDigit(ch) {
			l.advance()
			digits++
			continue
		}

		break
	}

	if digits != 6 && digits != 8 {
		raw := l.source[startPos:l.pos]

		return fmt.Errorf(
			"invalid color literal %q at %d:%d; expected #RRGGBB or #RRGGBBAA",
			raw,
			start.Line,
			start.Column,
		)
	}

	raw := l.source[startPos:l.pos]

	l.emitAt(
		token.COLOR,
		raw,
		raw,
		l.currentIndent(),
		"",
		start,
	)

	return nil
}

func (l *Lexer) readNumber() error {
	start := l.position()
	startPos := l.pos
	hasDecimal := false

	// .5
	if l.peek(0) == '.' {
		hasDecimal = true
		l.advance()
	}

	// Integer / decimal digits.
	for l.pos < len(l.source) && l.isDigit(l.peek(0)) {
		l.advance()
	}

	// Decimal fraction.
	if l.peek(0) == '.' && !hasDecimal {
		next := l.peek(1)

		// Only consume the dot when it is actually part of the number.
		if l.isDigit(next) || !l.isIdentifierStart(next) {
			hasDecimal = true
			l.advance()

			for l.pos < len(l.source) && l.isDigit(l.peek(0)) {
				l.advance()
			}
		}
	}

	// Scientific notation.
	if ch := l.peek(0); ch == 'e' || ch == 'E' {
		next := l.peek(1)

		if l.isDigit(next) {
			l.advance()

			for l.pos < len(l.source) && l.isDigit(l.peek(0)) {
				l.advance()
			}
		} else if next == '+' || next == '-' {
			if l.isDigit(l.peek(2)) {
				l.advance()
				l.advance()

				for l.pos < len(l.source) && l.isDigit(l.peek(0)) {
					l.advance()
				}
			}
		}
	}

	raw := l.source[startPos:l.pos]

	var value float64

	if _, err := fmt.Sscan(raw, &value); err != nil {
		return fmt.Errorf(
			"invalid number %q at %d:%d",
			raw,
			start.Line,
			start.Column,
		)
	}

	l.emitAt(
		token.NUMBER,
		value,
		raw,
		l.currentIndent(),
		raw,
		start,
	)

	return nil
}

func (l *Lexer) readIdentifier() {
	start := l.position()
	startPos := l.pos

	for l.pos < len(l.source) && l.isIdentifierChar(l.peek(0)) {
		l.advance()
	}

	value := l.source[startPos:l.pos]

	switch {
	case value == "true":
		l.emitAt(
			token.BOOLEAN,
			true,
			value,
			l.currentIndent(),
			"",
			start,
		)

	case value == "false":
		l.emitAt(
			token.BOOLEAN,
			false,
			value,
			l.currentIndent(),
			"",
			start,
		)

	case value == "na":
		l.emitAt(
			token.NA,
			nil,
			value,
			l.currentIndent(),
			"",
			start,
		)

	case token.IsKeyword(value):
		l.emitAt(
			token.KEYWORD,
			value,
			value,
			l.currentIndent(),
			"",
			start,
		)

	default:
		l.emitAt(
			token.IDENTIFIER,
			value,
			value,
			l.currentIndent(),
			"",
			start,
		)
	}
}

func (l *Lexer) readOperatorOrPunctuation() bool {
	start := l.position()

	ch := l.peek(0)
	next := l.peek(1)

	two := string([]byte{ch, next})

	switch two {
	case "==", "!=", "<=", ">=", ":=", "+=", "-=", "*=", "/=", "%=", "=>":
		l.advance()
		l.advance()

		l.emitAt(
			token.OPERATOR,
			two,
			two,
			l.currentIndent(),
			"",
			start,
		)

		return true
	}

	switch ch {
	case '+', '-', '*', '/', '%', '<', '>', '=', '!', '?':
		l.advance()

		raw := string([]byte{ch})

		l.emitAt(
			token.OPERATOR,
			raw,
			raw,
			l.currentIndent(),
			"",
			start,
		)

		return true

	case '(':
		l.parenDepth++
		l.advance()

		l.emitAt(
			token.LPAREN,
			"(",
			"(",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case ')':
		if l.parenDepth == 0 {
			l.fail("unexpected ')' without matching '('")
			return true
		}

		l.parenDepth--
		l.advance()

		l.emitAt(
			token.RPAREN,
			")",
			")",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case '[':
		grouped := l.parenDepth > 0 ||
			l.bracketDepth > 0 ||
			l.braceDepth > 0

		l.bracketDepth++
		l.advance()

		l.emitAt(
			token.LBRACKET,
			"[",
			"[",
			l.currentIndent(),
			"",
			start,
		)

		l.tokens[len(l.tokens)-1].Grouped = grouped

		return true

	case ']':
		if l.bracketDepth == 0 {
			l.fail("unexpected ']' without matching '['")
			return true
		}

		l.bracketDepth--
		l.advance()

		l.emitAt(
			token.RBRACKET,
			"]",
			"]",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case '{':
		l.braceDepth++
		l.advance()

		l.emitAt(
			token.LBRACE,
			"{",
			"{",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case '}':
		if l.braceDepth == 0 {
			l.fail("unexpected '}' without matching '{'")
			return true
		}

		l.braceDepth--
		l.advance()

		l.emitAt(
			token.RBRACE,
			"}",
			"}",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case ',':
		l.advance()

		l.emitAt(
			token.COMMA,
			",",
			",",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case '.':
		l.advance()

		l.emitAt(
			token.DOT,
			".",
			".",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case ':':
		l.advance()

		l.emitAt(
			token.COLON,
			":",
			":",
			l.currentIndent(),
			"",
			start,
		)

		return true

	case ';':
		l.advance()

		l.emitAt(
			token.SEMICOLON,
			";",
			";",
			l.currentIndent(),
			"",
			start,
		)

		return true
	}

	return false
}

func (l *Lexer) emit(
	typ token.TokenType,
	value any,
	raw string,
	indent int,
	forcedRaw string,
) token.Token {
	start := l.position()

	if forcedRaw != "" {
		raw = forcedRaw
	}

	return l.emitAt(
		typ,
		value,
		raw,
		indent,
		"",
		start,
	)
}

func (l *Lexer) emitAt(
	typ token.TokenType,
	value any,
	raw string,
	indent int,
	_ string,
	start token.Position,
) token.Token {
	t := token.Token{
		Type:  typ,
		Value: value,
		Raw:   raw,
		Span: token.Span{
			Start: start,
			End:   l.position(),
		},
		Indent: indent,
	}

	if l.pendingWrap != nil {
		t.Wrapped = l.pendingWrap
		l.pendingWrap = nil
	}

	l.tokens = append(l.tokens, t)

	return t
}

func (l *Lexer) currentIndent() int {
	return l.indentStack[len(l.indentStack)-1]
}

func (l *Lexer) position() token.Position {
	return token.Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.pos,
	}
}

func (l *Lexer) peek(offset int) byte {
	p := l.pos + offset

	if p < 0 || p >= len(l.source) {
		return 0
	}

	return l.source[p]
}

func (l *Lexer) advance() byte {
	if l.pos >= len(l.source) {
		return 0
	}

	ch := l.source[l.pos]

	l.pos++
	l.column++

	return ch
}

func (l *Lexer) isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func (l *Lexer) isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') ||
		(ch >= 'a' && ch <= 'f') ||
		(ch >= 'A' && ch <= 'F')
}

func (l *Lexer) isIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		ch == '_'
}

func (l *Lexer) isIdentifierChar(ch byte) bool {
	return l.isIdentifierStart(ch) || l.isDigit(ch)
}

// fail records a lexer failure using panic.
//
// Tokenize recovers this panic at the public boundary below.
func (l *Lexer) fail(message string) {
	panic(message)
}

// SafeTokenize converts internal lexer failures into errors.
func SafeTokenize(source string) (tokens []token.Token, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
			tokens = nil
		}
	}()

	return New(source).Tokenize()
}
