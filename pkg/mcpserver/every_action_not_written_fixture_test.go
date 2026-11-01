package mcpserver

import "fmt"

// fixtureNotWritten is the error a store returns for a failure before it wrote
// anything, marked with the sentinel (ErrPendingNotWritten). It is in a file of
// its own because it is the one symbol of these fixtures that the change
// adds: for the switches-off baseline, generated on main, this file is replaced
// by one that returns err unmarked, which is what the store returned there.
func fixtureNotWritten(err error) error {
	return fmt.Errorf("%w: %w", ErrPendingNotWritten, err)
}
