// vtwidth is a utility to measure the width of a string as it will be rendered
// in the terminal
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"git.sr.ht/~rockorager/vaxis"
)

func main() {
	var verbose bool
	flag.BoolVar(&verbose, "v", false, "print verbose result")
	flag.BoolVar(&verbose, "verbose", false, "print verbose result")
	flag.Parse()

	var input string
	switch len(flag.Args()) {
	case 0:
		fmt.Print("Enter text: ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input = scanner.Text()
	case 1:
		input = flag.Arg(0)
	case 2:
		fmt.Println("multiple arguments not supported")
		os.Exit(1)
	}
	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// We can close vaxis immediately. We have the data we need already
	// loaded in the struct
	vx.Close()
	w := vx.RenderedWidth(input)
	fmt.Println(w)
	if verbose {
		out := "|" + strings.Repeat("-", w) + "|"
		fmt.Println(out)
		fmt.Println("|" + input + "|")
	}
}
