package cliui

import (
	"flag"
	"os"
	"sync"
	"time"

	"github.com/muesli/termenv"
	"golang.org/x/xerrors"

	"github.com/coder/pretty"
)

var Canceled = xerrors.New("canceled")

// DefaultStyles compose visual elements of the UI.
var DefaultStyles Styles

type Styles struct {
	Code,
	DateTimeStamp,
	Error,
	Field,
	Keyword,
	Placeholder,
	Prompt,
	FocusedPrompt,
	Fuchsia,
	Warn,
	Wrap pretty.Style
}

var (
	color     termenv.Profile
	colorOnce sync.Once
)

var (
	Green  = Color("2")
	Red    = Color("1")
	Yellow = Color("3")
	Blue   = Color("6")
)

// Color returns a color for the given string.
func Color(s string) termenv.Color {
	colorOnce.Do(func() {
		color = termenv.NewOutput(os.Stdout).ColorProfile()
		if flag.Lookup("test.v") != nil {
			// Use a consistent colorless profile in tests so that results
			// are deterministic.
			color = termenv.Ascii
		}
	})
	return color.Color(s)
}

func isTerm() bool {
	return color != termenv.Ascii
}

// Bold returns a formatter that renders text in bold
// if the terminal supports it.
func Bold(s string) string {
	if !isTerm() {
		return s
	}
	return pretty.Sprint(pretty.Bold(), s)
}

// BoldFmt returns a formatter that renders text in bold
// if the terminal supports it.
func BoldFmt() pretty.Formatter {
	if !isTerm() {
		return pretty.Style{}
	}
	return pretty.Bold()
}

// Timestamp formats a timestamp for display.
func Timestamp(t time.Time) string {
	return pretty.Sprint(DefaultStyles.DateTimeStamp, t.Format(time.Stamp))
}

// Keyword formats a keyword for display.
func Keyword(s string) string {
	return pretty.Sprint(DefaultStyles.Keyword, s)
}

// Placeholder formats a placeholder for display.
func Placeholder(s string) string {
	return pretty.Sprint(DefaultStyles.Placeholder, s)
}

// Wrap prevents the text from overflowing the terminal.
func Wrap(s string) string {
	return pretty.Sprint(DefaultStyles.Wrap, s)
}

// Code formats code for display.
func Code(s string) string {
	return pretty.Sprint(DefaultStyles.Code, s)
}

// Field formats a field for display.
func Field(s string) string {
	return pretty.Sprint(DefaultStyles.Field, s)
}

func ifTerm(fmt pretty.Formatter) pretty.Formatter {
	if !isTerm() {
		return pretty.Nop
	}
	return fmt
}

func init() {
	// We do not adapt the color based on whether the terminal is light or dark.
	// Doing so would require a round-trip between the program and the terminal
	// due to the OSC query and response.
	DefaultStyles = Styles{
		Code: pretty.Style{
			ifTerm(pretty.XPad(1, 1)),
			pretty.FgColor(color.Color("#ED567A")),
			pretty.BgColor(color.Color("#2c2c2c")),
		},
		DateTimeStamp: pretty.Style{
			pretty.FgColor(color.Color("#7571F9")),
		},
		Error: pretty.Style{
			pretty.FgColor(Red),
		},
		Field: pretty.Style{
			pretty.XPad(1, 1),
			pretty.FgColor(color.Color("#FFFFFF")),
			pretty.BgColor(color.Color("#2b2a2a")),
		},
		Keyword: pretty.Style{
			pretty.FgColor(Green),
		},
		Placeholder: pretty.Style{
			pretty.FgColor(color.Color("#4d46b3")),
		},
		Prompt: pretty.Style{
			pretty.FgColor(color.Color("#5C5C5C")),
			pretty.Wrap("> ", ""),
		},
		Warn: pretty.Style{
			pretty.FgColor(Yellow),
		},
		Wrap: pretty.Style{
			pretty.LineWrap(80),
		},
	}

	DefaultStyles.FocusedPrompt = append(
		DefaultStyles.Prompt,
		pretty.FgColor(Blue),
	)
}

// ValidateNotEmpty is a helper function to disallow empty inputs!
func ValidateNotEmpty(s string) error {
	if s == "" {
		return xerrors.New("Must be provided!")
	}
	return nil
}
