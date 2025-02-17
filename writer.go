package easynet

import (
	"io"
	"os"
)

var DefaultWriter io.Writer = os.Stdout

var DefaultErrorWriter io.Writer = os.Stderr
