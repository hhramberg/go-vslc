package util

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"sync"
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

// syncer is a sync.Mutex synchronised structure that keeps track of two counters. One counter counts the number of
// worker go routines that have registered a Writer. The other counter keeps track of the number of write operations.
type syncer struct {
	active  int // active keeps track of the number of active go worker threads.
	writing int // writing keeps track of the number of write operations.
	sync.Mutex
}

// ---------------------
// ----- Constants -----
// ---------------------

var wc chan string // wc is the writer channel used for receiving data from worker go routines.
var cc chan error  // cc is the close channel used by main thread to signal to end write operations.
var sc syncer

// ---------------------
// ----- functions -----
// ---------------------

// Write writes a format string to the Writer's buffer.
func (w *Writer) Write(format string, args ...interface{}) {
	w.sb.WriteString(fmt.Sprintf(format, args...))
}

// WriteString writes a plain string to the Writer's buffer.
func (w *Writer) WriteString(s string) {
	w.sb.WriteString(s)
}

// Label writes a one-line label with the given name.
func (w *Writer) Label(name string) {
	w.sb.WriteString(fmt.Sprintf("%s:\n", name))
}

// Len returns the result of calling the Len function on the underlying strings.Builder.
func (w *Writer) Len() int {
	return w.sb.Len()
}

// Flush empties the Writer's buffer and sends the buffer data to the
// designated output writer over the Writer's channel.
func (w *Writer) Flush() {
	if w.sb.Len() < 1 {
	}
	sc.addWriteOperation()
	w.c <- w.sb.String()
	w.sb.Reset()
}

// Close flushes the Writer's buffer and then closes the Writer's channel.
func (w *Writer) Close() {
	w.Flush()
	w.c = nil
	sc.subWriter()
}

// NewWriter returns a new Writer to be used by worker threads to write strings concurrently to the output buffer.
// Must not be called before main thread has called ListenWrite.
func NewWriter() Writer {
	sc.addWriter()
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
func ListenWrite(opt Options, f *os.File) {
	if opt.Threads > 1 && !opt.LLVM && !opt.TokenStream {
		// LLVM IR can't be output in parallel.
		wc = make(chan string, opt.Threads+1)
	} else {
		wc = make(chan string, 1)
	}
	cc = make(chan error)
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
		stop := false
		for {
			if stop {
				// Got stop signal. Check for pending jobs.
				sc.Lock()
				if sc.writing == 0 && sc.active == 0 {
					// No more jobs, no active writers: close the listener and tell
					// the main thread over the close channel.
					sc.Unlock()
					cc <- nil
					return // Stop the listener writer go routine.
				}
				sc.Unlock()
			}
			select {
			case s := <-wc:
				if _, err := w.WriteString(s); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				if err := w.Flush(); err != nil {
					fmt.Println(err)
					os.Exit(1)
				}
				sc.subWriteOperation()
			case <-cc:
				stop = true
			}
		}
	}(wc, cc)
}

// ListenWriteBench is equal to ListenWrite, but it doesn't write the contents to the destination file.
// This function is used for benchmarking, where writing multiple gigabytes to disk is undesirable.
func ListenWriteBench(opt Options) {
	if opt.Threads > 1 && !opt.LLVM && !opt.TokenStream {
		// LLVM IR can't be output in parallel.
		wc = make(chan string, opt.Threads+1)
	} else {
		wc = make(chan string, 1)
	}
	cc = make(chan error)

	// Listen for input and termination signal.
	go func(wc chan string, cc chan error) {
		defer close(wc)
		defer close(cc)
		stop := false
		for {
			if stop {
				// Got stop signal. Check for pending jobs.
				sc.Lock()
				if sc.writing == 0 && sc.active == 0 {
					// No more jobs, no active writers: close the listener and tell
					// the main thread over the close channel.
					sc.Unlock()
					cc <- nil
					return // Stop the listener writer go routine.
				}
				sc.Unlock()
			}
			select {
			case _ = <-wc:
				sc.subWriteOperation()
			case <-cc:
				stop = true
			}
		}
	}(wc, cc)
}

// Close sends the termination signal to the writer listener.
func Close() {
	cc <- nil // Send close signal to writer listener.
	<-cc      // Wait for clear signal from writer listener go routine.
}

// addWriter increments the registered writers on the syncer.
func (sc *syncer) addWriter() {
	sc.Lock()
	sc.active++
	sc.Unlock()
}

// subWriter decrements the registered writers on the syncer.
func (sc *syncer) subWriter() {
	sc.Lock()
	sc.active--
	sc.Unlock()
}

// addWriteOperation increments the number of write operations on the syncer.
func (sc *syncer) addWriteOperation() {
	sc.Lock()
	sc.writing++
	sc.Unlock()
}

// subWriteOperation decrements the number of write operations on the syncer.
func (sc *syncer) subWriteOperation() {
	sc.Lock()
	sc.writing--
	sc.Unlock()
}
