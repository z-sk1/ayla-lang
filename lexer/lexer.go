package lexer

import (
	"strings"

	"github.com/z-sk1/ayla-lang/token"
)

type Lexer struct {
	input        string
	position     int
	readPosition int
	ch           byte
}

func New(input string) *Lexer {
	l := &Lexer{input: input}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func (l *Lexer) readIdentifier() string {
	start := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[start:l.position]
}

// read numbers
func (l *Lexer) readNumber() string {
	start := l.position
	hasDot := false

	for isDigit(l.ch) || l.ch == '.' {
		if l.ch == '.' {
			if hasDot { // second dot, invalid
				break
			}
			hasDot = true
		}

		l.readChar()
	}
	return l.input[start:l.position]
}

func (l *Lexer) readString() string {
	// skip the opening quote
	l.readChar()
	start := l.position
	for l.ch != '"' && l.ch != 0 {
		l.readChar()
	}
	str := l.input[start:l.position]
	l.readChar() // skip closing quote
	return str
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	} else {
		return l.input[l.readPosition]
	}
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()

	var tok token.Token

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = token.Token{Type: token.ASSIGN, Literal: "="}
		}
	case '+':
		tok = token.Token{Type: token.PLUS, Literal: "+"}
	case '-':
		tok = token.Token{Type: token.MINUS, Literal: "-"}
	case ';':
		tok = token.Token{Type: token.SEMICOLON, Literal: ";"}
	case '/':
		tok = token.Token{Type: token.SLASH, Literal: "/"}
	case '"':
		tok = token.Token{Type: token.STRING, Literal: l.readString()}
		return tok
	case ',':
		tok = token.Token{Type: token.COMMA, Literal: ","}
	case ':':
		tok = token.Token{Type: token.COLON, Literal: ":"}
	case '*':
		tok = token.Token{Type: token.ASTERISK, Literal: "*"}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch)}
		} else {
			tok = token.Token{Type: token.LT, Literal: "<"}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch)}
		} else {
			tok = token.Token{Type: token.GT, Literal: ">"}
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch)}
		} else {
			tok = token.Token{Type: token.BANG, Literal: "!"}
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch)}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch)}
		}
	case 0:
		tok = token.Token{Type: token.EOF, Literal: ""}
	case '(':
		tok = token.Token{Type: token.LPAREN, Literal: "("}
	case ')':
		tok = token.Token{Type: token.RPAREN, Literal: ")"}
	case '{':
		tok = token.Token{Type: token.LBRACE, Literal: "{"}
	case '}':
		tok = token.Token{Type: token.RBRACE, Literal: "}"}
	case '[':
		tok = token.Token{Type: token.LBRACKET, Literal: "["}
	case ']':
		tok = token.Token{Type: token.RBRACKET, Literal: "]"}
	default:
		if isLetter(l.ch) {
			literal := l.readIdentifier()
			tok.Type = token.LookupIdent(literal) // <-- keyword check
			tok.Literal = literal
			return tok // return early to avoid readChar() below
		} else if isDigit(l.ch) {
			num := l.readNumber()
			if strings.Contains(num, ".") {
				return token.Token{Type: token.FLOAT, Literal: num}
			}
			return token.Token{Type: token.INT, Literal: num} // return early
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch)}
		}
	}

	l.readChar()
	return tok
}
