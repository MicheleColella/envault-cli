package ui

import (
	"fmt"
	"io"
	"os"
)

var (
	Out io.Writer = os.Stdout
	Err io.Writer = os.Stderr
)

var colorEnabled = os.Getenv("NO_COLOR") == "" && os.Getenv("TERM") != "dumb"

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
)

func colorize(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + ansiReset
}

func OK(msg string) {
	fmt.Fprintln(Out, colorize(ansiGreen, "✓")+" "+msg)
}

func Fail(msg string) {
	fmt.Fprintln(Err, colorize(ansiRed, "✗")+" "+msg)
}

func Info(msg string) {
	fmt.Fprintln(Out, colorize(ansiDim, "→")+" "+msg)
}

func Warn(msg string) {
	fmt.Fprintln(Out, colorize(ansiYellow, "!")+" "+msg)
}

func Header(msg string) {
	fmt.Fprintln(Out, colorize(ansiBold, msg))
}

func NotImplemented(name string) {
	Fail(name + ": not implemented yet")
}
