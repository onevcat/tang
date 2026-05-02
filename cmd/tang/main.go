// Command tang is the Tangled command-line client.
package main

import (
	"os"

	"tangled.org/onev.cat/tang/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	root := cli.NewRootCommand(cli.BuildInfo{
		Version: version,
		Commit:  commit,
	})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
