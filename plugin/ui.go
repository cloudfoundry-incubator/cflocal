package plugin

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

type UI struct {
	Out io.Writer
	Err io.Writer
	In  io.Reader
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

func (u *UI) Error(err error) {
	fmt.Fprintf(u.Err, "Error: %s\n", err)
	fmt.Fprintln(u.Out, color.RedString("FAILED"))
}
