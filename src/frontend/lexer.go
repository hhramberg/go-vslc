// This lexer is based on, and copied from, Rob Pike's excellent talk on Go scanners.
// Link to the talk on YouTube: https://www.youtube.com/watch?v=HxaD_trXwRE
// Link to presentation slides: https://talks.golang.org/2011/lex.slide#1
//
// The lexer uses state functions stateFunc to define the lexer state. States allow the lexer to treat same runes
// differently. State transitions happens in the current states and appearance of key runes, or transition runes if you
// would. The lexer uses the Go 'character' type 'rune' which enables native UTF-8 support for the source being scanned.

package frontend

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// stateFunc defines the state of the lexer.
type stateFunc func(*lexer) stateFunc

// itemType is used to differentiate different tokens scanned by the lexer.
type itemType int

// item contains a lexeme scanned by the lexer and its position in the source stream.
type item struct {
	typ  itemType // Token type to emit.
	val  string   // Value of token.
	line int      // Line of token in source stream.
	pos  int      // Start position on current line of token in source stream.
}

// lexer is a lexical type that traverse a source stream character by character and emits lexemes.
type lexer struct {
	input       string     // The source stream of characters to scan for lexemes.
	start       int        // The starting position of the current token.
	pos         int        // The current position of the scanner in the source stream.
	width       int        // The width of the currently scanned rune/character in bytes.
	line        int        // The current line in the source stream. Not zero-indexed.
	startOnLine int        // The start position of the current token on the current line. Not zero-indexed.
	state       stateFunc  // The start state of the lexer.
	err         chan error // A channel for reporting errors.
	items       chan item  // A channel for emitting item tokens.
}

// ---------------------
// ----- Constants -----
// ---------------------

const eof = 0 // Same as '\0' for null-terminated C strings.

const (
	itemEOF itemType = iota
	itemError
)

// --------------------------
// ----- Item functions -----
// --------------------------

// String returns a print friendly string representation of the item.
func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return fmt.Sprintf("%s [ERROR]", i.val)
	}
	if len(i.val) > 10 {
		return fmt.Sprintf("%.10q... (line %d:%d)", i.val, i.line, i.pos)
	}
	return fmt.Sprintf("%q (line %d:%d)", i.val, i.line, i.pos)
}

// ---------------------------
// ----- Lexer functions -----
// ---------------------------

// Lex is called by the parser and awaits tokens emitted by the concurrent lexer.
// A token compatible with the goyacc parser is put in the lval argument, and the token type is returned.
func (l *lexer) Lex(lval *yySymType) int {
	i := l.nextItem()
	lval.typ = int(i.typ)
	lval.val = i.val
	lval.line = i.line
	lval.pos = i.pos
	return int(i.typ)
}

// Called by the parser when a parse error is encountered.
func (l *lexer) Error(e string) {
	l.err <- errors.New(e)
}

// newLexer creates and returns a pointer to a new lexer.
func newLexer(src string, start stateFunc) *lexer {
	return &lexer{
		input:       src,
		start:       0,
		pos:         0,
		width:       0,
		line:        1,
		startOnLine: 1,
		state:       start,
		err:         make(chan error),
		items:       make(chan item, 2),
	}
}

// run initiates the traversal of the input stream of the lexer, resulting in tokens being emitted
// on the lexer's items channel.
func (l *lexer) run() {
	defer close(l.items)
	defer close(l.err)
	for state := l.state; state != nil; {
		select {
		case err := <-l.err:
			fmt.Printf("Syntax error: %s\n", err)
			return
		default:
			state = state(l)
		}
	}
}

// emit sends an item of type typ back to the caller.
func (l *lexer) emit(typ itemType) {
	defer func() {
		if r := recover(); r != nil {
			// Send on closed channel.
			l.state = nil // Stop the lexer.
		}
	}()

	l.items <- item{
		typ:  typ,
		val:  l.input[l.start:l.pos],
		line: l.line,
		pos:  l.startOnLine,
	}
	l.startOnLine += len(l.input[l.start:l.pos])
	l.start = l.pos
}

// next returns the next rune in the input. The use of runes makes the lexer UTF-8 compatible.
func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.startOnLine += len(l.input[l.start:l.pos])
	l.start = l.pos
}

// backup steps back one rune. Should only be called once per call of next.
func (l *lexer) backup() {
	if l.pos > l.start {
		l.pos -= l.width
	}
}

// peek returns, but does not consume, the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next rune if it's from the set of valid characters defined by the valid string.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a sequence of runes from the set of valid characters defined by the valid string.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

// nextItem returns the next item from the input.
func (l *lexer) nextItem() item {
	item := <-l.items
	return item
}

// errorf returns an error token and terminates the scan by passing back a nil pointer
// that will be the next state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFunc {
	l.items <- item{
		typ:  itemError,
		val:  fmt.Sprintf(format, args...),
		line: 0,
		pos:  0,
	}
	return nil
}
