package main

import "arangodb-bk-restore/cmd"

// Build information (set by GoReleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	builtBy = "unknown"
)

func main() {
	cmd.Execute()
}
