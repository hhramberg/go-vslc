package frontend

import "unicode/utf8"

// lexGlobal starts the lexing process and serves as the default state.
func lexGlobal(l *lexer) stateFunc {
	for {
		r := l.next()
		switch {
		case isAlpha(r):
			// Keyword or identifier.
			return lexWord
		case isDigit(r):
			// Number.
			return lexNumber
		case r == '\n':
			// Newline.
			l.ignore()
			l.line++
			l.startOnLine = 1
		case isSpace(r):
			// Ignore whitespace. Newlines are caught before whitespaces.
			// Based on Google's RE2 WHITESPACE class: [\t\n\f\r ]
			l.ignore()
		case r == '"':
			// String.
			return lexString
		case r == ':' && l.peek() == '=':
			// Assignment operator.
			l.next()
			l.emit(ASSIGN)
		case r == '<' && l.peek() == '<':
			// Left shift operator.
			l.next()
			l.emit(LSHIFT)
		case r == '>' && l.peek() == '>':
			// Right shift operator.
			l.next()
			l.emit(RSHIFT)
		case r == '/' && l.peek() == '/':
			// Ignore comments.
			for c := l.next(); c != '\n'; c = l.next() {
			}
			l.ignore()
			l.line++
			l.startOnLine = 1
		case r == eof:
			// End of file: stop the state machine.
			l.emit(itemEOF)
			return nil
		default:
			// Let parser use character as is.
			l.emit(itemType(r))
		}
	}
}

// lexWord scans the input string for keywords and identifiers.
func lexWord(l *lexer) stateFunc {
	// We know that the currently scanned rune is an alphabetic character.
	for {
		r := l.next()

		// Check if character is valid character.
		if !isAlpha(r) && !isDigit(r) && r != '_' {
			l.backup()
			kw, typ := isKeyword(l.input[l.start:l.pos])
			if kw {
				l.emit(typ)
			} else {
				l.emit(IDENTIFIER)
			}
			return lexGlobal
		}
	}
}

// lexNumber scans the input stream for an integer number.
// This function accepts zero leading numbers and numbers consisting of all zeros.
func lexNumber(l *lexer) stateFunc {
	// We've scanned the first digit already. We don't scan negative numbers.
	// We instead let the parser handle negative numbers by grammar rules.

	// Scan integer part.
	r := l.next()
	for ; isDigit(r); r = l.next() {
	}

	// Check for decimal.
	if r == '.' {
		// Decimal delimiter found.
		for r = l.next(); isDigit(r); r = l.next() {
		}
		l.backup()
		l.emit(FLOAT)
	}
	l.backup()
	l.emit(INTEGER)
	return lexGlobal
}

// lexString scans a string literal from the input stream.
func lexString(l *lexer) stateFunc {
	// By this point we're int the string. Accept anything until the next '"' appears.
	// Escaped '"' (\") are ignored.
	l.ignore()
	prev, _ := utf8.DecodeRuneInString(l.input[l.pos-1:]) // Safe, because we must scan at least one rune to get here.
	for {
		r := l.next()
		if r == eof {
			return l.errorf("unclosed string literal at line %d:%d", l.line, l.startOnLine)
		}
		// Check for escaped string termination (\").
		if r == '"' && prev != '\\' {
			// Found string termination.
			l.backup()
			l.emit(STRING)
			l.next()
			l.ignore()
			return lexGlobal
		}
		prev = r
	}
}

// ----------------------------
// ----- Helper functions -----
// ----------------------------

// isAlpha return true if rune r is an alphabetic character in the set [a-zA-Z].
func isAlpha(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// isDigit return true if rune r is a digit in the range [0-9].
func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

// isSpace return true if rune r is a whitespace character.
func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\f' || r == '\r'
}
