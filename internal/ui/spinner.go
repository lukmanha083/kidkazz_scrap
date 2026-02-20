package ui

import (
	"fmt"
	"os"
	"sync"
	"time"
)

var frames = []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

// Spinner displays an animated progress indicator on stderr.
type Spinner struct {
	mu   sync.Mutex
	msg  string
	done chan struct{}
}

// NewSpinner creates a new Spinner (not yet running).
func NewSpinner() *Spinner {
	return &Spinner{}
}

// Start begins the spinner animation with the given message.
func (s *Spinner) Start(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.done = make(chan struct{})
	s.mu.Unlock()

	go s.run()
}

// Update changes the spinner message while it's running.
func (s *Spinner) Update(msg string) {
	s.mu.Lock()
	s.msg = msg
	s.mu.Unlock()
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	s.mu.Lock()
	if s.done != nil {
		close(s.done)
		s.done = nil
	}
	s.mu.Unlock()

	// Clear the spinner line
	fmt.Fprintf(os.Stderr, "\r\033[K")
}

func (s *Spinner) run() {
	tick := time.NewTicker(80 * time.Millisecond)
	defer tick.Stop()

	i := 0
	for {
		select {
		case <-s.done:
			return
		case <-tick.C:
			s.mu.Lock()
			msg := s.msg
			s.mu.Unlock()
			fmt.Fprintf(os.Stderr, "\r\033[K%c %s", frames[i%len(frames)], msg)
			i++
		}
	}
}
