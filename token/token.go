package token

type TokenType string

type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL"
	EOF     = "EOF"

	// identifiers
	IDENT  = "IDENT"
	INT    = "INT"
	STRING = "STRING"
	// operators
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	SLASH    = "/"
	ASTERISK = "*"

	EQ     = "EQ"
	NOT_EQ = "NOT_EQ"
	LT     = "<"
	GT     = ">"
	LTE    = "LTE"
	GTE    = "GTE"

	BANG = "!"
	AND  = "AND"
	OR   = "OR"

	COMMA     = ","
	SEMICOLON = ";"
	COLON     = ":"

	LPAREN   = "("
	RPAREN   = ")"
	LBRACE   = "{"
	RBRACE   = "}"
	LBRACKET = "["
	RBRACKET = "]"

	// keywords
	VAR   = "VAR"
	PRINT = "PRINT"
	IF    = "IF"
	ELSE  = "ELSE"
	FOR   = "FOR"
	WHILE = "WHILE"
	CONST = "CONST"
	TRUE  = "TRUE"
	FALSE = "FALSE"

	SCANLN = "SCANLN"

	INT_TYPE    = "INT_TYPE"
	STRING_TYPE = "STRING_TYPE"
	BOOL_TYPE   = "BOOL_TYPE"
)

var keywords = map[string]TokenType{
	"egg":     VAR,
	"rock":    CONST,
	"explode": PRINT,
	"ayla":    IF,
	"elen":    ELSE,
	"int":     INT_TYPE,
	"string":  STRING_TYPE,
	"bool":    BOOL_TYPE,
	"four":    FOR,
	"why":     WHILE,
	"true":    TRUE,
	"false":   FALSE,
	"scanln":  SCANLN,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
