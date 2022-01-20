// stack.go provides a linked list type stack that holds arbitrary data.
// The bottom element is the first entry into the stack, while the top is
// the last entry to be added to the stack. The stack does not store <nil>
// values.

package util

import "sync"

// StackElement holds data in the Stack linked list.
type StackElement struct {
	E    interface{}   // Data held by stack entry.
	next *StackElement // Pointer to next entry following this StackElement.
}

// Stack is a linked list stack.
type Stack struct {
	size   int           // Number of entries in the stack.
	bottom *StackElement // The first element to be added to the stack.
	top    *StackElement // The last element to be added to the stack.
	mx     sync.Mutex    // For synchronising multiple worker threads to one stack.
}

// Push adds a new element to the top of the stack.
func (s *Stack) Push(e interface{}) {
	if e == nil {
		return
	}
	se := StackElement{
		E:    e,
		next: nil,
	}
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.size == 0 {
		s.bottom = &se
		s.top = &se
	} else {
		s.top.next = &se
		s.top = &se
	}
	s.size++
}

// Pop removes and returns the last inserted element on the stack.
// If no element has been added <nil> is returned.
func (s *Stack) Pop() interface{} {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.size == 0 {
		return nil
	}
	if s.size == 1 {
		e := s.bottom
		s.bottom = nil
		s.top = nil
		s.size--
		return e.E
	}

	prev := s.bottom
	e1 := prev.next
	for ; e1.next != nil; e1 = e1.next {
		prev = prev.next
	}
	s.top = prev
	s.top.next = nil
	s.size--
	return e1.E
}

// Peek works just like Pop, but it does not remove the element from the stack.
func (s *Stack) Peek() interface{} {
	s.mx.Lock()
	defer s.mx.Unlock()
	if s.size == 0 {
		return nil
	}
	e := s.top.E
	return e
}

// Size returns the number of elements in the stack.
func (s *Stack) Size() int {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.size
}

// Get returns the nth element from the stack, top down, not zero indexed.
// Get(1) returns the first element on stack, and is similar to Peek.
// Get(Stack.size) returns the last element on the stack, and is similar
// to returning the bottom element. If the index n is out of range or negative
// <nil> is returned. Get does not remove elements from the stack.
func (s *Stack) Get(n int) interface{} {
	s.mx.Lock()
	defer s.mx.Unlock()
	if n < 1 || n > s.size {
		return nil
	}

	e1 := s.bottom
	for i1 := 0; i1 <= s.size - n; i1++ {
		e1 = e1.next
	}
	return e1.E
}
