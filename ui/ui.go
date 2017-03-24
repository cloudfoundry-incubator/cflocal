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

const spinnerWidth = 5

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

func (u *UI) Loading(message string, f func() error) error {
	doneChan := make(chan error)
	go func() { doneChan <- f() }()

	timeChan := time.NewTimer(2 * time.Second).C
	var tickChan <-chan time.Time

	ticks := 0
	for {
		select {
		case <-timeChan:
			tickChan = time.NewTicker(time.Millisecond * 250).C
		case <-tickChan:
			fmt.Fprintf(u.Out, "\r%s > %s%s%s", message,
				strings.Repeat(spinner[len(spinner)-1], ticks/len(spinner)%spinnerWidth),
				spinner[ticks%len(spinner)],
				strings.Repeat("  ", spinnerWidth-ticks/len(spinner)%spinnerWidth),
			)
			ticks++
		case err := <-doneChan:
			if ticks > 0 {
				fmt.Fprintf(u.Out, "\r%s   %s\r",
					strings.Repeat(" ", len(message)),
					strings.Repeat("  ", spinnerWidth),
				)
			}
			return err
		}
	}
}
