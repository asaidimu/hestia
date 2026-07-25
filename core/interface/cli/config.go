package cli

import "io"

type Config struct {
	Version string
	Stdin   io.Reader
	Stdout  io.Writer
}
