// label.go provides a thread safe way of generating assembly labels for jumps.

package util

import "fmt"

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

var cll chan string // Label channel; results:
var clr chan int    // Request channel.
var clc chan error  // Close channel.

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

// ListenLabel listens for label requests and returns labels to requesting worker threads.
func ListenLabel() {
	cll = make(chan string)
	clr = make(chan int)
	clc = make(chan error)

	defer close(clr)
	defer close(cll)
	defer close(clc)

	for {
		select {
		case <-clc:
			return
		case i := <-clr:
			if i >= 0 && i < len(labelIndices) {
				cll <- fmt.Sprintf("%s_%03d", labelPrefixes[i], labelIndices[i])
				labelIndices[i]++
			} else {
				cll <- "# LABEL ERROR"
			}
		}
	}
}

// NewLabel returns a new label of type typ.
func NewLabel(typ int) string {
	clr <- typ
	s := <-cll
	return s
}

// CloseLabel sends the termination signal to the thread safe label generator. Must only be called once and after
// assembly code generation has finished, successful or not.
func CloseLabel() {
	clc <- nil
}
