package util

import "sync"

// ----------------------------
// ----- Type definitions -----
// ----------------------------

// perror provides a structure for listening for errors reported from parallel worker threads and means for retrieving
// errors when a parallel job hsa been completed.
type perror struct {
	listen     chan error // Channel for receiving error messages from worker threads.
	stop       chan error // Messages sent on this channel causes the perror struct to stop listening for errors.
	errors     []error    // Buffer of error messages.
	sync.Mutex            // For synchronising writes and reads.
}

// ----------------------
// ----- Constants ------
// ----------------------

// defaultBufferSize defines the fallback buffer size of the error array.
const defaultBufferSize = 16

// -------------------
// ----- globals -----
// -------------------

// ---------------------
// ----- functions -----
// ---------------------

// NewPerror returns a pointer to a perror struct with n number of pre-allocated slots for errors in the buffer.
func NewPerror(n int) *perror {
	if n < 1 {
		n = defaultBufferSize
	}
	pe := perror{
		listen: make(chan error),
		stop:   make(chan error),
		errors: make([]error, 0, n),
	}
	go pe.run()
	return &pe
}

// run starts listening for errors on the listen channel. Sending a message on the close channel causes the error
// listener to stop.
func (pe *perror) run() {
	defer close(pe.listen)
	for {
		select {
		case err := <-pe.listen:
			pe.Lock()
			pe.errors = append(pe.errors, err)
			pe.Unlock()
		case <-pe.stop:
			return
		}
	}
}

// Flush empties the buffered error messages of the error listener. Flush must not be called after Stop.
func (pe *perror) Flush() {
	pe.Lock()
	defer pe.Unlock()
	pe.errors = make([]error, 0, cap(pe.errors))
}

// Len returns the number of buffered errors.
func (pe *perror) Len() int {
	pe.Lock()
	defer pe.Unlock()
	return len(pe.errors)
}

// Stop sends the stop signal to the error listener.
func (pe *perror) Stop() {
	defer close(pe.stop)
	pe.stop <- nil
}

// Append sends the error message err to the error listener. <nil> errors are ignored.
func (pe *perror) Append(err error) {
	if err != nil {
		pe.listen <- err
	}
}

// Errors returns a buffered channel with all the reported errors since the last call to Reset, effectively creating
// an iterator.
func (pe *perror) Errors() <-chan error {
	pe.Lock()
	defer pe.Unlock()
	c := make(chan error, len(pe.errors))
	for _, e1 := range pe.errors {
		c <- e1
	}
	return c
}
