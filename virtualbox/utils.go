package virtualbox

import (
	"fmt"
	"log"
)

// errLogf is an abstraction function which allows you to both log and return an error
func errLogf(format string, args ...interface{}) error {
	// TODO: Consider call depth if we add line logging to the errors
	e := fmt.Errorf("[ERROR] "+format, args...)
	log.Println(e)
	return e
}
