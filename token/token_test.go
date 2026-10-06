package token

import "testing"

func TestKeywordLookup(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"if", true},
		{"else", true},
		{"for", true},
		{"while", true},
		{"switch", true},
		{"var", true},
		{"varip", true},
		{"and", true},
		{"or", true},
		{"not", true},
		{"method", true},
		{"enum", true},
		{"pinego", false},
		{"close", false},
	}

	for _, tt := range tests {
		got := IsKeyword(tt.name)
		if got != tt.want {
			t.Fatalf("IsKeyword(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestContextualKeywords(t *testing.T) {
	for _, word := range []string{"type", "method", "enum"} {
		if !IsContextualKeyword(word) {
			t.Fatalf("expected %q to be contextual", word)
		}
	}
}

func TestReservedWords(t *testing.T) {
	for _, word := range []string{
		"catch",
		"class",
		"do",
		"ellipse",
		"polygon",
		"return",
		"struct",
		"text",
		"throw",
		"try",
	} {
		if !IsReservedWord(word) {
			t.Fatalf("expected %q to be reserved", word)
		}
	}
}

func TestTokenConstruction(t *testing.T) {
	tok := Token{
		Type:  IDENTIFIER,
		Value: "close",
		Raw:   "close",
		Span: Span{
			Start: Position{
				Line:   2,
				Column: 4,
				Offset: 8,
			},
			End: Position{
				Line:   2,
				Column: 9,
				Offset: 13,
			},
		},
		Indent: 1,
	}

	if tok.Type != IDENTIFIER {
		t.Fatalf("unexpected token type: %v", tok.Type)
	}

	if tok.Value != "close" {
		t.Fatalf("unexpected token value: %v", tok.Value)
	}

	if tok.Span.Start.Line != 2 {
		t.Fatalf("unexpected start line: %d", tok.Span.Start.Line)
	}
}
