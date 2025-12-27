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

	line   int
	column int
}

func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}

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

	if l.ch == '\n' {
		l.line++
		l.column = 0
	} else {
		l.column++
	}
}

func isLetter(ch byte) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isIdentStart(ch byte) bool {
    return isLetter(ch) || ch == '_'
}

func isIdentPart(ch byte) bool {
    return isLetter(ch) || isDigit(ch) || ch == '_'
}


func (l *Lexer) readIdentifier() string {
    pos := l.position
    for isIdentPart(l.ch) {
        l.readChar()
    }
    return l.input[pos:l.position]
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
			tok = token.Token{Type: token.EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = token.Token{Type: token.ASSIGN, Literal: "=", Line: l.line, Column: l.column}
		}
	case '+':
		tok = token.Token{Type: token.PLUS, Literal: "+", Line: l.line, Column: l.column}
	case '-':
		tok = token.Token{Type: token.MINUS, Literal: "-", Line: l.line, Column: l.column}
	case ';':
		tok = token.Token{Type: token.SEMICOLON, Literal: ";", Line: l.line, Column: l.column}
	case '/':
		tok = token.Token{Type: token.SLASH, Literal: "/", Line: l.line, Column: l.column}
	case '"':
		tok = token.Token{Type: token.STRING, Literal: l.readString(), Line: l.line, Column: l.column}
		return tok
	case ',':
		tok = token.Token{Type: token.COMMA, Literal: ",", Line: l.line, Column: l.column}
	case ':':
		tok = token.Token{Type: token.COLON, Literal: ":", Line: l.line, Column: l.column}
	case '*':
		tok = token.Token{Type: token.ASTERISK, Literal: "*", Line: l.line, Column: l.column}
	case '<':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = token.Token{Type: token.LT, Literal: "<", Line: l.line, Column: l.column}
		}
	case '>':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = token.Token{Type: token.GT, Literal: ">", Line: l.line, Column: l.column}
		}
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		} else {
			tok = token.Token{Type: token.BANG, Literal: "!", Line: l.line, Column: l.column}
		}
	case '&':
		if l.peekChar() == '&' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		}
	case '|':
		if l.peekChar() == '|' {
			ch := l.ch
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: string(ch) + string(l.ch), Line: l.line, Column: l.column}
		}
	case 0:
		tok = token.Token{Type: token.EOF, Literal: "", Line: l.line, Column: l.column}
	case '(':
		tok = token.Token{Type: token.LPAREN, Literal: "(", Line: l.line, Column: l.column}
	case ')':
		tok = token.Token{Type: token.RPAREN, Literal: ")", Line: l.line, Column: l.column}
	case '{':
		tok = token.Token{Type: token.LBRACE, Literal: "{", Line: l.line, Column: l.column}
	case '}':
		tok = token.Token{Type: token.RBRACE, Literal: "}", Line: l.line, Column: l.column}
	case '[':
		tok = token.Token{Type: token.LBRACKET, Literal: "[", Line: l.line, Column: l.column}
	case ']':
		tok = token.Token{Type: token.RBRACKET, Literal: "]", Line: l.line, Column: l.column}
	default:
		if isIdentStart(l.ch) {
			literal := l.readIdentifier()
			tok.Type = token.LookupIdent(literal)
			tok.Literal = literal
			tok.Line = l.line
			tok.Column = l.column
			return tok
		} else if isDigit(l.ch) {
			num := l.readNumber()
			if strings.Contains(num, ".") {
				return token.Token{Type: token.FLOAT, Literal: num, Line: l.line, Column: l.column}
			}
			return token.Token{Type: token.INT, Literal: num, Line: l.line, Column: l.column}
		} else {
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Line: l.line, Column: l.column}
		}
	}

	l.readChar()
	return tok
}
