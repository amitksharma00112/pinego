package token

type TokenType string

type Token struct {
	Type   TokenType
	Lexeme string
	Line   int
	Column int
}
