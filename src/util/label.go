// label.go provides a thread safe way of generating assembly labels for jumps.

package util

import (
	"fmt"
	"sync"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// ---------------------
// ----- Constants -----
// ---------------------

// Labels for conditionals.
const (
	LabelWhileHead = iota
	LabelWhileEnd
	LabelIf
	LabelIfElse
	LabelIfEnd
	LabelIfElseEnd
	LabelJump
)

// -------------------
// ----- Globals -----
// -------------------

var mx sync.Mutex // Mutex for synchronising worker threads.

// labelIndices stores the numerical suffix for generated labels of types.
var labelIndices [LabelJump + 1]int

// labelPrefixes stores the string literal prefixes for labels of types.
var labelPrefixes = [LabelJump + 1]string{
	"LWhileHead",
	"LWhileEnd",
	"LIf",
	"LIfElse",
	"LIfEnd",
	"LIfElseEnd",
	"LJump",
}

// ---------------------
// ----- Functions -----
// ---------------------

// NewLabel returns a new label of type typ.
func NewLabel(typ int) string {
	mx.Lock()
	mx.Unlock()
	if typ >= 0 && typ < len(labelIndices) {
		s := fmt.Sprintf("%s_%03d", labelPrefixes[typ], labelIndices[typ])
		labelIndices[typ]++
		return s
	} else {
		return "# LABEL ERROR"
	}
}
