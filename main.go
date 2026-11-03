package main

import "os"
import "fmt"
import "github.com/folio-org/mod-reporting/reporting"

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "<configFile.json>")
		os.Exit(1)
	}

	server, err := reporting.MakeConfiguredServer(os.Args[1], ".")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot create server: %s\n", os.Args[0], err)
		os.Exit(2)
	}

	err = server.Launch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: cannot launch server: %s\n", os.Args[0], err)
		os.Exit(3)
	}
}
