// Tests the lexer type by verifying that a sample VSL program 'bitops.vsl' is tokenized properly.
//
// The sample file was manually transformed into a slice of item types holding both token
// numerical type, string value and line position. It is expected that the lexer output tokens in the same order as the
// tuple slice, as it traverses the source string from start to finish.

package frontend

import (
	"testing"
	"vslc/src/util"
)

// TestLexer tests the lexing state functions to verify that it correctly scans a sample VSL file for tokens.
func TestLexer(t *testing.T) {
	opt := util.Options{Src: "../../resources/vsl/bitops.vsl"}
	s, err := util.ReadSource(opt)
	if err != nil {
		t.Fatalf("failed to open file %q: %s", opt.Src, err)
	}

	// Line numbers and position indices were manually captured from Jetbrains GoLand IDE.
	exp := []item{
		{val: "def", typ: DEF, line: 4, pos: 1},
		{val: "bitwise_operators", typ: IDENTIFIER, line: 4, pos: 5},
		{val: "(", typ: '(', line: 4, pos: 23},
		{val: "a", typ: IDENTIFIER, line: 4, pos: 25},
		{val: ",", typ: ',', line: 4, pos: 26},
		{val: "b", typ: IDENTIFIER, line: 4, pos: 28},
		{val: ")", typ: ')', line: 4, pos: 30},
		{val: "begin", typ: BEGIN, line: 5, pos: 1},
		{val: "var", typ: VAR, line: 6, pos: 5},
		{val: "c", typ: IDENTIFIER, line: 6, pos: 9},
		{val: "print", typ: PRINT, line: 7, pos: 5},
		{val: "a is", typ: STRING, line: 7, pos: 12},
		{val: ",", typ: ',', line: 7, pos: 17},
		{val: "a", typ: IDENTIFIER, line: 7, pos: 19},
		{val: ",", typ: ',', line: 7, pos: 20},
		{val: "and b is", typ: STRING, line: 7, pos: 23},
		{val: ",", typ: ',', line: 7, pos: 32},
		{val: "b", typ: IDENTIFIER, line: 7, pos: 34},
		{val: "c", typ: IDENTIFIER, line: 8, pos: 5},
		{val: ":=", typ: ASSIGN, line: 8, pos: 7},
		{val: "~", typ: '~', line: 8, pos: 10},
		{val: "a", typ: IDENTIFIER, line: 8, pos: 12},
		{val: "print", typ: PRINT, line: 9, pos: 5},
		{val: "~", typ: STRING, line: 9, pos: 12},
		{val: ",", typ: ',', line: 9, pos: 14},
		{val: "a", typ: IDENTIFIER, line: 9, pos: 16},
		{val: ",", typ: ',', line: 9, pos: 17},
		{val: "=", typ: STRING, line: 9, pos: 19},
		{val: ",", typ: ',', line: 9, pos: 21},
		{val: "c", typ: IDENTIFIER, line: 9, pos: 23},
		{val: "c", typ: IDENTIFIER, line: 10, pos: 5},
		{val: ":=", typ: ASSIGN, line: 10, pos: 7},
		{val: "a", typ: IDENTIFIER, line: 10, pos: 10},
		{val: "|", typ: '|', line: 10, pos: 12},
		{val: "b", typ: IDENTIFIER, line: 10, pos: 14},
		{val: "print", typ: PRINT, line: 11, pos: 5},
		{val: "a", typ: IDENTIFIER, line: 11, pos: 11},
		{val: ",", typ: ',', line: 11, pos: 12},
		{val: "|", typ: STRING, line: 11, pos: 14},
		{val: ",", typ: ',', line: 11, pos: 16},
		{val: "b", typ: IDENTIFIER, line: 11, pos: 17},
		{val: ",", typ: ',', line: 11, pos: 18},
		{val: "=", typ: STRING, line: 11, pos: 20},
		{val: ",", typ: ',', line: 11, pos: 22},
		{val: "c", typ: IDENTIFIER, line: 11, pos: 23},
		{val: "c", typ: IDENTIFIER, line: 12, pos: 5},
		{val: ":=", typ: ASSIGN, line: 12, pos: 7},
		{val: "a", typ: IDENTIFIER, line: 12, pos: 10},
		{val: "^", typ: '^', line: 12, pos: 12},
		{val: "b", typ: IDENTIFIER, line: 12, pos: 14},
		{val: "print", typ: PRINT, line: 13, pos: 5},
		{val: "a", typ: IDENTIFIER, line: 13, pos: 11},
		{val: ",", typ: ',', line: 13, pos: 12},
		{val: "^", typ: STRING, line: 13, pos: 14},
		{val: ",", typ: ',', line: 13, pos: 16},
		{val: "b", typ: IDENTIFIER, line: 13, pos: 17},
		{val: ",", typ: ',', line: 13, pos: 18},
		{val: "=", typ: STRING, line: 13, pos: 20},
		{val: ",", typ: ',', line: 13, pos: 22},
		{val: "c", typ: IDENTIFIER, line: 13, pos: 23},
		{val: "c", typ: IDENTIFIER, line: 14, pos: 5},
		{val: ":=", typ: ASSIGN, line: 14, pos: 7},
		{val: "a", typ: IDENTIFIER, line: 14, pos: 10},
		{val: "&", typ: '&', line: 14, pos: 12},
		{val: "b", typ: IDENTIFIER, line: 14, pos: 14},
		{val: "print", typ: PRINT, line: 15, pos: 5},
		{val: "a", typ: IDENTIFIER, line: 15, pos: 11},
		{val: ",", typ: ',', line: 15, pos: 12},
		{val: "&", typ: STRING, line: 15, pos: 14},
		{val: ",", typ: ',', line: 15, pos: 16},
		{val: "b", typ: IDENTIFIER, line: 15, pos: 17},
		{val: ",", typ: ',', line: 15, pos: 18},
		{val: "=", typ: STRING, line: 15, pos: 20},
		{val: ",", typ: ',', line: 15, pos: 22},
		{val: "c", typ: IDENTIFIER, line: 15, pos: 23},
		{val: "c", typ: IDENTIFIER, line: 16, pos: 5},
		{val: ":=", typ: ASSIGN, line: 16, pos: 7},
		{val: "a", typ: IDENTIFIER, line: 16, pos: 10},
		{val: "<<", typ: LSHIFT, line: 16, pos: 12},
		{val: "b", typ: IDENTIFIER, line: 16, pos: 15},
		{val: "print", typ: PRINT, line: 17, pos: 5},
		{val: "a", typ: IDENTIFIER, line: 17, pos: 11},
		{val: ",", typ: ',', line: 17, pos: 12},
		{val: "<<", typ: STRING, line: 17, pos: 14},
		{val: ",", typ: ',', line: 17, pos: 17},
		{val: "b", typ: IDENTIFIER, line: 17, pos: 18},
		{val: ",", typ: ',', line: 17, pos: 19},
		{val: "=", typ: STRING, line: 17, pos: 21},
		{val: ",", typ: ',', line: 17, pos: 23},
		{val: "c", typ: IDENTIFIER, line: 17, pos: 24},
		{val: "c", typ: IDENTIFIER, line: 18, pos: 5},
		{val: ":=", typ: ASSIGN, line: 18, pos: 7},
		{val: "a", typ: IDENTIFIER, line: 18, pos: 10},
		{val: ">>", typ: RSHIFT, line: 18, pos: 12},
		{val: "b", typ: IDENTIFIER, line: 18, pos: 15},
		{val: "print", typ: PRINT, line: 19, pos: 5},
		{val: "a", typ: IDENTIFIER, line: 19, pos: 11},
		{val: ",", typ: ',', line: 19, pos: 12},
		{val: ">>", typ: STRING, line: 19, pos: 14},
		{val: ",", typ: ',', line: 19, pos: 17},
		{val: "b", typ: IDENTIFIER, line: 19, pos: 18},
		{val: ",", typ: ',', line: 19, pos: 19},
		{val: "=", typ: STRING, line: 19, pos: 21},
		{val: ",", typ: ',', line: 19, pos: 23},
		{val: "c", typ: IDENTIFIER, line: 19, pos: 24},
		{val: "return", typ: RETURN, line: 20, pos: 5},
		{val: "0", typ: INTEGER, line: 20, pos: 12},
		{val: "end", typ: END, line: 21, pos: 1},
	}

	l := newLexer(s, lexGlobal)
	go l.run()

	for i1 := 0; ; i1++ {
		tok := l.nextItem()

		// Check for
		if tok.typ == itemEOF {
			if len(exp)-1 > i1 {
				t.Fatalf("expected %d tokens, got %d", len(exp), i1+1)
			}
			break
		}
		if i1 >= len(exp) {
			t.Fatalf("expected %d tokens, got more", len(exp))
		}
		if tok.typ != exp[i1].typ || tok.val != exp[i1].val {
			t.Errorf("(token %d): expected %q, got %q", i1+1, exp[i1].val, tok.String())
		} else if tok.line != exp[i1].line || tok.pos != exp[i1].pos {
			t.Errorf("(token %d): expected %q to be on line %d:%d, got line %d:%d",
				i1+1, exp[i1].val, exp[i1].line, exp[i1].pos, tok.line, tok.pos)
		}
	}
}
