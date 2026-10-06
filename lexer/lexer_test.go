package lexer

import (
	"testing"

	"github.com/amitksharma00112/pinego/token"
)

func lex(t *testing.T, source string) []token.Token {
	t.Helper()

	tokens, err := SafeTokenize(source)
	if err != nil {
		t.Fatalf("lex failed: %v", err)
	}

	return tokens
}

func types(tokens []token.Token) []token.TokenType {
	out := make([]token.TokenType, 0, len(tokens))
	for _, tok := range tokens {
		out = append(out, tok.Type)
	}
	return out
}

func TestBasicExpression(t *testing.T) {
	got := lex(t, "x = 10 + 20")

	want := []token.TokenType{
		token.IDENTIFIER,
		token.OPERATOR,
		token.NUMBER,
		token.OPERATOR,
		token.NUMBER,
		token.EOF,
	}

	gotTypes := types(got)

	if len(gotTypes) != len(want) {
		t.Fatalf("got %v, want %v", gotTypes, want)
	}

	for i := range want {
		if gotTypes[i] != want[i] {
			t.Fatalf("token %d: got %v, want %v", i, gotTypes[i], want[i])
		}
	}
}

func TestLiterals(t *testing.T) {
	tokens := lex(t, `123 123.45 .5 10. 1e10 1.5e-5 "hello" 'world' true false na #FF0000 #FF000080`)

	want := []token.TokenType{
		token.NUMBER,
		token.NUMBER,
		token.NUMBER,
		token.NUMBER,
		token.NUMBER,
		token.NUMBER,
		token.STRING,
		token.STRING,
		token.BOOLEAN,
		token.BOOLEAN,
		token.NA,
		token.COLOR,
		token.COLOR,
		token.EOF,
	}

	got := types(tokens)

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestKeywords(t *testing.T) {
	tokens := lex(t, "if else for while switch break continue var varip type and or not import export method enum")

	for _, tok := range tokens[:len(tokens)-1] {
		if tok.Type != token.KEYWORD {
			t.Fatalf("expected KEYWORD, got %v (%v)", tok.Type, tok.Value)
		}
	}
}

func TestOperators(t *testing.T) {
	tokens := lex(t, `+ - * / % < > <= >= == != = := += -= *= /= %= => ?`)

	want := []string{
		"+",
		"-",
		"*",
		"/",
		"%",
		"<",
		">",
		"<=",
		">=",
		"==",
		"!=",
		"=",
		":=",
		"+=",
		"-=",
		"*=",
		"/=",
		"%=",
		"=>",
		"?",
	}

	if len(tokens) != len(want)+1 {
		t.Fatalf("got %d tokens, want %d", len(tokens), len(want)+1)
	}

	for i, wantValue := range want {
		if tokens[i].Type != token.OPERATOR {
			t.Fatalf("token %d: got type %v", i, tokens[i].Type)
		}

		if tokens[i].Value != wantValue {
			t.Fatalf("token %d: got %v, want %v", i, tokens[i].Value, wantValue)
		}
	}
}

func TestPunctuation(t *testing.T) {
	tokens := lex(t, "( ) [ ] { } , . : ;")

	want := []token.TokenType{
		token.LPAREN,
		token.RPAREN,
		token.LBRACKET,
		token.RBRACKET,
		token.LBRACE,
		token.RBRACE,
		token.COMMA,
		token.DOT,
		token.COLON,
		token.SEMICOLON,
		token.EOF,
	}

	got := types(tokens)

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestComments(t *testing.T) {
	tokens := lex(t, "// hello\nx = 10 // inline")

	if tokens[0].Type != token.COMMENT {
		t.Fatalf("expected COMMENT, got %v", tokens[0].Type)
	}

	if tokens[0].Value != "hello" {
		t.Fatalf("unexpected comment value: %v", tokens[0].Value)
	}

	if tokens[1].Type != token.NEWLINE {
		t.Fatalf("expected NEWLINE, got %v", tokens[1].Type)
	}

	if tokens[2].Type != token.IDENTIFIER {
		t.Fatalf("expected IDENTIFIER, got %v", tokens[2].Type)
	}
}

func TestIndentation(t *testing.T) {
	source := `if close > open
    x = 10
    if x > 5
        y = 20
    z = 30
result = 40`

	tokens := lex(t, source)
	got := types(tokens)

	want := []token.TokenType{
		token.KEYWORD,
		token.IDENTIFIER,
		token.OPERATOR,
		token.IDENTIFIER,
		token.NEWLINE,

		token.INDENT,
		token.IDENTIFIER,
		token.OPERATOR,
		token.NUMBER,
		token.NEWLINE,

		token.KEYWORD,
		token.IDENTIFIER,
		token.OPERATOR,
		token.NUMBER,
		token.NEWLINE,

		token.INDENT,
		token.IDENTIFIER,
		token.OPERATOR,
		token.NUMBER,
		token.NEWLINE,

		token.DEDENT,
		token.IDENTIFIER,
		token.OPERATOR,
		token.NUMBER,
		token.NEWLINE,

		token.DEDENT,
		token.IDENTIFIER,
		token.OPERATOR,
		token.NUMBER,
		token.EOF,
	}

	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("token %d: got %v, want %v", i, got[i], want[i])
		}
	}
}

func TestLineWrapping(t *testing.T) {
	tokens := lex(t, "x = 1 +\n  2")

	for _, tok := range tokens {
		if tok.Type == token.NEWLINE {
			t.Fatalf("unexpected NEWLINE in wrapped expression")
		}
	}

	foundWrapped := false

	for _, tok := range tokens {
		if tok.Type == token.NUMBER && tok.Value == float64(2) {
			foundWrapped = true

			if tok.Wrapped == nil {
				t.Fatal("expected wrapped metadata")
			}
		}
	}

	if !foundWrapped {
		t.Fatal("wrapped number token not found")
	}
}

func TestGroupedExpressionsSuppressNewline(t *testing.T) {
	source := `x = (
    10 +
    20
)`

	tokens := lex(t, source)

	newlineCount := 0

	for _, tok := range tokens {
		if tok.Type == token.NEWLINE {
			newlineCount++
		}
	}

	// Only the newline after the closing ')' is emitted.
	if newlineCount != 0 {
		t.Fatalf("unexpected NEWLINE tokens: %d", newlineCount)
	}
}

func TestMultilineString(t *testing.T) {
	source := "\"\"\"hello\nworld\"\"\""

	tokens := lex(t, source)

	if tokens[0].Type != token.STRING {
		t.Fatalf("expected STRING, got %v", tokens[0].Type)
	}

	if tokens[0].Value != "hello\nworld" {
		t.Fatalf("unexpected value: %q", tokens[0].Value)
	}
}

func TestCRLF(t *testing.T) {
	tokens := lex(t, "x = 10\r\ny = 20\r\n")

	newlineCount := 0

	for _, tok := range tokens {
		if tok.Type == token.NEWLINE {
			newlineCount++
		}
	}

	if newlineCount != 2 {
		t.Fatalf("got %d NEWLINE tokens, want 2", newlineCount)
	}
}

func TestInvalidColor(t *testing.T) {
	_, err := SafeTokenize("#FF00")

	if err == nil {
		t.Fatal("expected invalid color error")
	}
}

func TestUnterminatedString(t *testing.T) {
	_, err := SafeTokenize(`"hello`)

	if err == nil {
		t.Fatal("expected unterminated string error")
	}
}

func TestInvalidCharacter(t *testing.T) {
	_, err := SafeTokenize("@")

	if err == nil {
		t.Fatal("expected invalid character error")
	}
}
