package token

// Keywords contains Pine Script words that have syntactic meaning.
var Keywords = map[string]struct{}{
	// Control flow
	"if":       {},
	"else":     {},
	"for":      {},
	"while":    {},
	"switch":   {},
	"break":    {},
	"continue": {},

	// Declarations
	"var":   {},
	"varip": {},
	"type":  {},

	// Logical
	"and": {},
	"or":  {},
	"not": {},

	// Other language keywords
	"to":     {},
	"by":     {},
	"in":     {},
	"as":     {},
	"import": {},
	"export": {},
	"method": {},
	"enum":   {},
}

// ContextualKeywords are words that are keywords only in
// specific syntactic contexts.
var ContextualKeywords = map[string]struct{}{
	"type":   {},
	"method": {},
	"enum":   {},
}

// ReservedWords are names that Pine should reject at
// declaration/name-binding positions.
//
// They are still lexed as identifiers.
var ReservedWords = map[string]struct{}{
	"catch":   {},
	"class":   {},
	"do":      {},
	"ellipse": {},
	"is":      {},
	"polygon": {},
	"range":   {},
	"return":  {},
	"struct":  {},
	"text":    {},
	"throw":   {},
	"try":     {},
}

// IsKeyword reports whether s is a language keyword.
func IsKeyword(s string) bool {
	_, ok := Keywords[s]
	return ok
}

// IsContextualKeyword reports whether s has contextual keyword behavior.
func IsContextualKeyword(s string) bool {
	_, ok := ContextualKeywords[s]
	return ok
}

// IsReservedWord reports whether s is a reserved identifier name.
func IsReservedWord(s string) bool {
	_, ok := ReservedWords[s]
	return ok
}
