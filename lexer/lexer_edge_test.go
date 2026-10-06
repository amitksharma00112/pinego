package lexer

import (
	"math"
	"testing"

	"github.com/amitksharma00112/pinego/token"
)

func TestEmptySource(t *testing.T) {
	tokens, err := SafeTokenize("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tokens) != 1 {
		t.Fatalf("got %d tokens, want 1", len(tokens))
	}

	if tokens[0].Type != token.EOF {
		t.Fatalf("got %v, want EOF", tokens[0].Type)
	}
}

func TestTabsAndIndentation(t *testing.T) {
	source := "if close > open\n\tx = 10\n\tif x > 5\n\t\ty = 20\n\tz = 30\nresult = 40"

	tokens, err := SafeTokenize(source)
	if err != nil {
		t.Fatalf("lex failed: %v", err)
	}

	indentCount := 0
	dedentCount := 0

	for _, tok := range tokens {
		switch tok.Type {
		case token.INDENT:
			indentCount++
		case token.DEDENT:
			dedentCount++
		}
	}

	if indentCount != 2 {
		t.Fatalf("got %d INDENT tokens, want 2", indentCount)
	}

	if dedentCount != 2 {
		t.Fatalf("got %d DEDENT tokens, want 2", dedentCount)
	}
}

func TestIndentationJumpRejected(t *testing.T) {
	source := "if close > open\n        x = 10"

	_, err := SafeTokenize(source)
	if err == nil {
		t.Fatal("expected indentation jump error")
	}
}

func TestMisalignedDedentRejected(t *testing.T) {
	source := "if close > open\n    x = 10\n  y = 20"

	_, err := SafeTokenize(source)
	if err == nil {
		t.Fatal("expected misaligned dedent error")
	}
}

func TestUnmatchedClosingDelimiters(t *testing.T) {
	cases := []string{
		")",
		"]",
		"}",
	}

	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := SafeTokenize(source)
			if err == nil {
				t.Fatalf("expected error for %q", source)
			}
		})
	}
}

func TestUnclosedDelimiters(t *testing.T) {
	cases := []string{
		"(",
		"[",
		"{",
		"foo(bar",
		"foo([1, 2)",
	}

	for _, source := range cases {
		t.Run(source, func(t *testing.T) {
			_, err := SafeTokenize(source)
			if err == nil {
				t.Fatalf("expected unclosed delimiter error for %q", source)
			}
		})
	}
}

func TestStringEscapes(t *testing.T) {
	source := `"a\nb\tc\r\\d\""`

	tokens, err := SafeTokenize(source)
	if err != nil {
		t.Fatalf("lex failed: %v", err)
	}

	if tokens[0].Type != token.STRING {
		t.Fatalf("got %v, want STRING", tokens[0].Type)
	}

	want := "a\nb\tc\r\\d\""

	if tokens[0].Value != want {
		t.Fatalf("got %q, want %q", tokens[0].Value, want)
	}
}

func TestSingleQuotedString(t *testing.T) {
	tokens, err := SafeTokenize(`'hello world'`)
	if err != nil {
		t.Fatalf("lex failed: %v", err)
	}

	if tokens[0].Type != token.STRING {
		t.Fatalf("got %v, want STRING", tokens[0].Type)
	}

	if tokens[0].Value != "hello world" {
		t.Fatalf("got %q, want %q", tokens[0].Value, "hello world")
	}
}

func TestNumberValues(t *testing.T) {
	tokens, err := SafeTokenize("123 123.5 .5 10. 1e3 1.5e-2")
	if err != nil {
		t.Fatalf("lex failed: %v", err)
	}

	want := []float64{
		123,
		123.5,
		0.5,
		10,
		1000,
		0.015,
	}

	for i, wantValue := range want {
		got, ok := tokens[i].Value.(float64)
		if !ok {
			t.Fatalf("token %d: value type %T, want float64", i, tokens[i].Value)
		}

		if math.Abs(got-wantValue) > 1e-12 {
			t.Fatalf("token %d: got %v, want %v", i, got, wantValue)
		}
	}
}

func TestWrappedAfterOperator(t *testing.T) {
	tokens := lex(t, "x = 10 +\n  20")

	for _, tok := range tokens {
		if tok.Type == token.NEWLINE {
			t.Fatal("unexpected NEWLINE in wrapped expression")
		}
	}

	found := false
	for _, tok := range tokens {
		if tok.Type == token.NUMBER && tok.Value == float64(20) {
			found = true

			if tok.Wrapped == nil {
				t.Fatal("expected Wrapped metadata on continuation token")
			}
		}
	}

	if !found {
		t.Fatal("continuation number token not found")
	}
}

func TestWrappedAfterComma(t *testing.T) {
	tokens := lex(t, "f(1,\n  2)")

	for _, tok := range tokens {
		if tok.Type == token.NEWLINE {
			t.Fatal("unexpected NEWLINE in grouped expression")
		}
	}

	if tokens[len(tokens)-2].Type != token.RPAREN {
		t.Fatalf("got %v, want RPAREN", tokens[len(tokens)-2].Type)
	}
}

func TestWrappedAfterAnd(t *testing.T) {
	tokens := lex(t, "x = close > open and\n  high > low")

	for _, tok := range tokens {
		if tok.Type == token.NEWLINE {
			t.Fatal("unexpected NEWLINE in wrapped boolean expression")
		}
	}
}

func TestGroupedBracketMetadata(t *testing.T) {
	tokens := lex(t, "x = foo([1, 2])")

	var found bool

	for _, tok := range tokens {
		if tok.Type == token.LBRACKET {
			found = true

			if !tok.Grouped {
				t.Fatal("expected LBRACKET.Grouped=true inside parentheses")
			}
		}
	}

	if !found {
		t.Fatal("LBRACKET token not found")
	}
}

func TestSourceSpans(t *testing.T) {
	tokens := lex(t, "x = 42")

	if tokens[0].Type != token.IDENTIFIER {
		t.Fatalf("token 0: got %v", tokens[0].Type)
	}

	if tokens[0].Span.Start.Line != 1 || tokens[0].Span.Start.Column != 1 {
		t.Fatalf("token 0 start: %+v", tokens[0].Span.Start)
	}

	if tokens[0].Span.End.Line != 1 || tokens[0].Span.End.Column != 2 {
		t.Fatalf("token 0 end: %+v", tokens[0].Span.End)
	}

	if tokens[2].Type != token.NUMBER {
		t.Fatalf("token 2: got %v", tokens[2].Type)
	}

	if tokens[2].Span.Start.Column != 5 || tokens[2].Span.End.Column != 7 {
		t.Fatalf("number span: %+v", tokens[2].Span)
	}
}

func TestEOFClosesIndentation(t *testing.T) {
	tokens := lex(t, "if close > open\n    x = 10")

	if len(tokens) < 2 {
		t.Fatal("expected at least two tokens")
	}

	if tokens[len(tokens)-2].Type != token.DEDENT {
		t.Fatalf("penultimate token: got %v, want DEDENT", tokens[len(tokens)-2].Type)
	}

	if tokens[len(tokens)-1].Type != token.EOF {
		t.Fatalf("last token: got %v, want EOF", tokens[len(tokens)-1].Type)
	}
}

func TestCRLFSource(t *testing.T) {
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
