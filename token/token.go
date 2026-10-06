package token

// TokenType identifies the lexical category of a token.
type TokenType string

const (
	// Special
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	// Identifiers / keywords
	IDENTIFIER TokenType = "IDENTIFIER"
	KEYWORD    TokenType = "KEYWORD"

	// Literals
	NUMBER  TokenType = "NUMBER"
	STRING  TokenType = "STRING"
	BOOLEAN TokenType = "BOOLEAN"
	NA      TokenType = "NA"
	COLOR   TokenType = "COLOR"

	// Operators
	OPERATOR TokenType = "OPERATOR"

	// Punctuation
	LPAREN    TokenType = "LPAREN"    // (
	RPAREN    TokenType = "RPAREN"    // )
	LBRACKET  TokenType = "LBRACKET"  // [
	RBRACKET  TokenType = "RBRACKET"  // ]
	LBRACE    TokenType = "LBRACE"    // {
	RBRACE    TokenType = "RBRACE"    // }
	COMMA     TokenType = "COMMA"     // ,
	DOT       TokenType = "DOT"       // .
	COLON     TokenType = "COLON"     // :
	SEMICOLON TokenType = "SEMICOLON" // ;

	// Layout
	NEWLINE TokenType = "NEWLINE"
	INDENT  TokenType = "INDENT"
	DEDENT  TokenType = "DEDENT"

	// Comments
	COMMENT TokenType = "COMMENT"
)

// Position identifies a location in the source.
type Position struct {
	Line   int
	Column int
	Offset int
}

// Span identifies the source range occupied by a token.
type Span struct {
	Start Position
	End   Position
}

// WrapInfo stores metadata for a Pine line that was joined to
// the previous logical line.
type WrapInfo struct {
	Width    int
	FromLine int
	Column   int
}

// Token represents one lexical unit produced by the lexer.
type Token struct {
	Type  TokenType
	Value any
	Raw   string

	Span Span

	// Logical indentation level at which this token occurs.
	Indent int

	// Present when the token starts a wrapped continuation line.
	Wrapped *WrapInfo

	// True when '[' occurs inside a grouped expression.
	Grouped bool

	// Opening quote column for a string literal.
	StartColumn int
}
