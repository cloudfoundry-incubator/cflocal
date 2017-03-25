package ui

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/fatih/color"
)

var spinner = []string{". ", "o ", "O ", "8 ", "oo", "OO", "88"}

const (
	spinnerWidth = 6
	loaderWidth  = 72

	spinnerPrefix = ": building > "
	loaderPrefix  = ": "

	spinnerDelay    = 2 * time.Second
	spinnerInterval = 250 * time.Millisecond
)

type UI struct {
	Out       io.Writer
	Err       io.Writer
	In        io.Reader
	ErrIsTerm bool
}

func (u *UI) Prompt(message string) string {
	in := bufio.NewReader(u.In)
	fmt.Fprint(u.Out, message+" ")
	text, err := in.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(text, "\n")
}

func (u *UI) Output(format string, a ...interface{}) {
	fmt.Fprintf(u.Out, format+"\n", a...)
}

func (u *UI) Warn(format string, a ...interface{}) {
	writer := u.Err
	if !u.ErrIsTerm {
		// use u.Out with pre-6.22.0 cf CLI
		writer = u.Out
	}
	fmt.Fprintf(writer, "Warning: "+format+"\n", a...)
}

func (u *UI) Error(err error) {
	writer := u.Err
	if !u.ErrIsTerm {
		// use u.Out with pre-6.22.0 cf CLI
		writer = u.Out
	}
	fmt.Fprintf(writer, "Error: %s\n", err)
	fmt.Fprintln(u.Out, color.RedString("FAILED"))
}

func (u *UI) Loading(message string, f func(progress chan<- string) error) error {
	loadLen := len(message+loaderPrefix) + loaderWidth
	spinLen := len(message+spinnerPrefix) + spinnerWidth*len(spinner[0])

	done := make(chan error)
	progress := make(chan string)
	go func() { done <- f(progress) }()

	var updateSpinner <-chan time.Time
	startSpinner := time.After(spinnerDelay)
	ticks := 0

	for {
		select {
		case <-startSpinner:
			startSpinner = nil
			updateSpinner = time.Tick(spinnerInterval)
		case <-updateSpinner:
			fmt.Fprintf(u.Out, "\r%s%s%s%s%s", message, spinnerPrefix,
				strings.Repeat(spinner[len(spinner)-1], ticks/len(spinner)%spinnerWidth),
				spinner[ticks%len(spinner)],
				strings.Repeat("  ", spinnerWidth-ticks/len(spinner)%spinnerWidth),
			)
			ticks++
		case status := <-progress:
			switch status {
			case "":
				if updateSpinner == nil && startSpinner == nil {
					fmt.Fprintf(u.Out, "\r%s\r", strings.Repeat(" ", loadLen))
					updateSpinner = time.Tick(spinnerInterval)
				}
			default:
				if updateSpinner != nil {
					fmt.Fprintf(u.Out, "\r%s\r", strings.Repeat(" ", spinLen))
					updateSpinner = nil
				} else if startSpinner != nil {
					startSpinner = nil
				}
				fmt.Fprintf(u.Out, "\r%s%s%s", message, loaderPrefix, status)
			}
		case err := <-done:
			fmt.Fprintf(u.Out, "\r%s\r", strings.Repeat(" ", max(loadLen, spinLen)))
			return err
		}
	}
}

func max(i, j int) int {
	if i > j {
		return i
	}
	return j
}
