package util

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// Writer buffers output from threads in a strings.Buffer.
// When the Flush or Close method is called the buffer is emptied and sent to
// the assigned output writer through channel c.
type Writer struct {
	sb strings.Builder
	c  chan string
}

// ---------------------
// ----- Constants -----
// ---------------------

var wc chan string // Write channel used for receiving data from worker threads.
var cc chan error  // Close channel used by main thread to signal to end write operations.

// ---------------------
// ----- Functions -----
// ---------------------

// Write writes a format string to the Writer's buffer.
func (w *Writer) Write(format string, args ...interface{}) {
	w.sb.WriteString(fmt.Sprintf(format, args...))
}

// Ins1 writes a one-line instruction using the operator and single operand.
func (w *Writer) Ins1(op, rs1 string) {
	w.sb.WriteString(fmt.Sprintf("\t%s\t%s\n", op, rs1))
}

// Ins2 writes a one-line instruction using the operator, destination register and single source register.
func (w *Writer) Ins2(op, rd, rs1 string) {
	w.sb.WriteString(fmt.Sprintf("\t%s\t%s, %s\n", op, rd, rs1))
}

// Ins2imm writes a one-line instruction using the operator, destination register, single source register and
// signed immediate.
func (w *Writer) Ins2imm(op, rd, rs1 string, imm int) {
	w.sb.WriteString(fmt.Sprintf("\t%s\t%s, %s, %d\n", op, rd, rs1, imm))
}

// Ins3 writes a one-line instruction using the operator, destination register and two source registers.
func (w *Writer) Ins3(op, rd, rs1, rs2 string) {
	w.sb.WriteString(fmt.Sprintf("\t%s\t%s, %s, %s\n", op, rd, rs1, rs2))
}

// Label writes a one-line label with the given name.
func (w *Writer) Label(name string) {
	w.sb.WriteString(fmt.Sprintf("\n%s:\n", name))
}

// Flush empties the Writer's buffer and sends the buffer data to the
// designated output writer over the Writer's channel.
func (w *Writer) Flush() {
	w.c <- w.sb.String()
	w.sb = strings.Builder{}
}

// Close flushes the Writer's buffer and then closes the Writer's channel.
func (w *Writer) Close() {
	w.Flush()
	w.c = nil
}

// NewWriter returns a new Writer to be used by worker threads to write strings concurrently to the output buffer.
// Must not be called before main thread has called ListenWrite.
func NewWriter() Writer {
	return Writer{
		sb: strings.Builder{},
		c:  wc,
	}
}

// ReadSource reads source code from file or stdin.
// If the Options structure holds a string for source the file will be opened and read.
// Else the function waits for a short period for input on stdin. If no input on stdin is
// provided the function returns an error.
func ReadSource(opt Options) (string, error) {
	if len(opt.Src) > 0 {
		// Read from file.
		b, err := ioutil.ReadFile(opt.Src)
		return string(b), err
	} else {
		// Read stdin.
		c := make(chan string)
		cerr := make(chan error)

		// Concurrently wait for input on stdin.
		go func(c chan string, cerr chan error) {
			defer close(c)
			defer close(cerr)
			reader := bufio.NewReader(os.Stdin)
			text, err := reader.ReadString(0)
			if err == nil {
				c <- text
			} else {
				cerr <- err
			}
		}(c, cerr)

		// Select between input from stdin or timer expiry.
		select {
		case <-time.After(500 * time.Millisecond):
			return "", errors.New("expected input from stdin, got none")
		case s := <-c:
			return s, nil
		}
	}
}

// ListenWrite listens for worker thread outputs. The received data is written to either file
// if File pointer f is not nil or stdout if File pointer f is nil. The function loops until
// a termination signal is sent using the Close function.
func ListenWrite(t int, f *os.File) {
	wc = make(chan string, t)
	cc = make(chan error, 1) // Make buffered to catch Close before listener is invoked.
	var w *bufio.Writer
	if f != nil {
		// Write output to file.
		w = bufio.NewWriter(f)
	} else {
		// Write output to stdout.
		w = bufio.NewWriter(os.Stdout)
	}

	// Listen for input and termination signal.
	go func(wc chan string, cc chan error) {
		defer close(wc)
		defer close(cc)
		for {
			select {
			case s := <-wc:
				if _, err := w.WriteString(s); err != nil {
					fmt.Println(err) // TODO: Handle better.
				}
				if err := w.Flush(); err != nil {
					fmt.Println(err) // TODO: Handle better.
				}
			case <-cc:
				return
			}
		}
	}(wc, cc)
}

// Close sends the termination signal to the writer listener.
func Close() {
	cc <- nil
}
