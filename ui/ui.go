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
	spinnerWidth        = 5
	dockerProgressWidth = 72

	spinnerPrefix = ": building > "
	loaderPrefix  = ": "
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

func (u *UI) Loading(message string, f func(chan<- string) error) error {
	loadLen := len(message+loaderPrefix) + dockerProgressWidth
	spinLen := len(message+spinnerPrefix) + spinnerWidth*len(spinner[0])

	doneChan := make(chan error)
	progressChan := make(chan string)
	go func() { doneChan <- f(progressChan) }()

	var timeChan, tickChan <-chan time.Time
	ticks := 0
	for {
		select {
		case progress := <-progressChan:
			switch progress {
			case "":
				if timeChan == nil {
					timeChan = time.After(time.Second)
				}
			default:
				if timeChan != nil {
					timeChan, tickChan = nil, nil
					fmt.Fprintf(u.Out, "\r%s\r", strings.Repeat(" ", spinLen))
				}
				fmt.Fprintf(u.Out, "\r%s%s%s", message, loaderPrefix, progress)

			}
		case <-timeChan:
			tickChan = time.Tick(time.Millisecond * 250)
			fmt.Fprintf(u.Out, "\r%s\r", strings.Repeat(" ", loadLen))
		case <-tickChan:
			fmt.Fprintf(u.Out, "\r%s%s%s%s%s", message, spinnerPrefix,
				strings.Repeat(spinner[len(spinner)-1], ticks/len(spinner)%spinnerWidth),
				spinner[ticks%len(spinner)],
				strings.Repeat("  ", spinnerWidth-ticks/len(spinner)%spinnerWidth),
			)
			ticks++
		case err := <-doneChan:
			fmt.Fprintf(u.Out, "\r%s\r", strings.Repeat(" ", max(loadLen, spinLen)))
			fmt.Fprintf(u.Out, "\r%s   %s\r",
				strings.Repeat(" ", len(message)),
				strings.Repeat("  ", spinnerWidth),
			)
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
