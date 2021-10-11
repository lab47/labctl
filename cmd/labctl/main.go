package main

import (
	"fmt"
	"os"

	"github.com/lab47/labctl/cli"
)

func main() {
	c, err := cli.NewCLI(os.Args[1:])
	if err != nil {
		fmt.Printf("Error setting up CLI: %s\n", err)
		os.Exit(1)
	}

	code, err := c.Run()
	if err != nil {
		fmt.Printf("Error running CLI: %s\n", err)
		os.Exit(1)
	}

	os.Exit(code)
}
