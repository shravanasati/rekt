package main

import (
	"fmt"

	"github.com/shravanasati/rekt/internal"
)

// todo rekt 3000 -> show process
// todo rekt 3000 -k -> kill process
// todo rekt 3000 -t -> terminate process
// todo rekt list -> list all processes occupying a port
// todo handle edge cases like SO_REUSEPORT

func main() {
	fmt.Println("hello from rekt")
	pf := internal.NewProcessFinder()
	fmt.Println(pf.FindPIDByPort(0))
}