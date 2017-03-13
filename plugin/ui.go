package plugin

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
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
