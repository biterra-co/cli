// Package ui provides terminal styling for the CLI: colors, symbols, and
// consistent formatting. Respects NO_COLOR and disables colors when stdout
// or stderr is not a TTY (via fatih/color).
package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

const ruleChar = "─"

// Logo is the ASCII art Biterra logo.
var Logo = []string{
	"",
	"",
	" @@@@          @@@@@",
	" @@@@          @@@@@",
	" @@@@                 @@@@",
	" @@@@ @@@@@@   @@@@@@@@@@@@@@@@   @@@@@@@@   @@@@@ @@@@@@@@@@ @@@@@  @@@@@@@@",
	" @@@@@@@@@@@@@ @@@@@@@@@@@@@@@@ @@@@@@@@@@@@ @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@",
	" @@@@@@   @@@@@@@@@@  @@@@     @@@@@@   @@@@ @@@@@@@@@@@@@@@@@@@@@@@@@@    @@@@@",
	" @@@@      @@@@@@@@@  @@@@    @@@@@@@@@@@@@@@@@@@@@    @@@@@@        @@@@@@@@@@@",
	" @@@@      @@@@@@@@@  @@@@    @@@@@@@@@@@@@@@@@@@@     @@@@@        @@@@@@@@@@@@",
	" @@@@      @@@@@@@@@  @@@@    @@@@@@         @@@@@     @@@@@      @@@@     @@@@@",
	" @@@@     @@@@@@@@@@  @@@@     @@@@@@    @@@@@@@@@     @@@@@     @@@@@     @@@@@",
	" @@@@@@@@@@@@@@@@@@@  @@@@@@@@@ @@@@@@@@@@@@@@@@@@     @@@@@     @@@@@@@@@@@@@@@",
	" @@@@@@@@@@@   @@@@@    @@@@@@@   @@@@@@@@@  @@@@@     @@@@@        @@@@@@@@@@@@",
	"",
	"",
	"",
	"",
}

var (
	success  = color.New(color.FgGreen)
	errorC   = color.New(color.FgRed)
	warning  = color.New(color.FgYellow)
	info     = color.New(color.FgCyan)
	muted    = color.New(color.FgHiBlack)
	bold     = color.New(color.Bold)
	urlStyle = color.New(color.FgCyan, color.Underline)
)

// syncOut sets the color package's global Output to the current os.Stdout so
// output is captured when tests redirect stdout. (fatih/color's Printf uses
// color.Output, not SetWriter.)
func syncOut() {
	color.Output = os.Stdout
}

// Success prints a green success line (e.g. "✓ Done").
func Success(format string, args ...interface{}) {
	syncOut()
	success.Printf("✓ "+format+"\n", args...)
}

// SuccessNoSymbol prints a green line without a leading symbol.
func SuccessNoSymbol(format string, args ...interface{}) {
	syncOut()
	success.Printf(format+"\n", args...)
}

// Error prints a red error line to stdout. For Cobra-returned errors use ErrorToStderr.
func Error(format string, args ...interface{}) {
	syncOut()
	errorC.Print("✗ ")
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// ErrorToStderr prints the message in red to stderr (for final CLI error output).
func ErrorToStderr(msg string) {
	_, _ = errorC.Fprint(os.Stderr, "✗ ")
	_, _ = fmt.Fprintln(os.Stderr, msg)
}

// Warning prints a yellow warning line.
func Warning(format string, args ...interface{}) {
	syncOut()
	warning.Print("⚠ ")
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Info prints a cyan info line.
func Info(format string, args ...interface{}) {
	syncOut()
	info.Printf(format+"\n", args...)
}

// Muted prints dimmed secondary text.
func Muted(format string, args ...interface{}) {
	syncOut()
	muted.Printf(format+"\n", args...)
}

// Bold prints bold text.
func Bold(format string, args ...interface{}) {
	syncOut()
	bold.Printf(format+"\n", args...)
}

// URL prints a styled URL (cyan, underline when supported).
func URL(s string) {
	syncOut()
	urlStyle.Println(s)
}

// StepStart prints the start of a step (e.g. "Looking up world... ") with no newline.
func StepStart(msg string) {
	syncOut()
	muted.Print(msg)
}

// StepOK completes a step with a green "done" or custom text.
func StepOK(msg string) {
	syncOut()
	if msg == "" {
		msg = "done"
	}
	success.Println(msg)
}

// StepFail completes a step with red "failed." and returns so caller can print more.
func StepFail() {
	syncOut()
	errorC.Println("failed.")
}

// Prompt prints a prompt line with a clear ">" cue (muted, no newline).
func Prompt(format string, args ...interface{}) {
	syncOut()
	info.Print("  → ")
	muted.Printf(format, args...)
}

// Rule prints a subtle horizontal line (full width or fixed length).
func Rule() {
	syncOut()
	const width = 50
	muted.Println(strings.Repeat(ruleChar, width))
}

// parseHexColor parses #RRGGBB or RRGGBB into r, g, b. Returns ok false if invalid.
func parseHexColor(s string) (r, g, b uint8, ok bool) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "#") {
		s = s[1:]
	}
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	parseByte := func(hex string) (uint8, bool) {
		n, err := strconv.ParseUint(hex, 16, 8)
		return uint8(n), err == nil
	}
	var okR, okG, okB bool
	r, okR = parseByte(s[0:2])
	g, okG = parseByte(s[2:4])
	b, okB = parseByte(s[4:6])
	return r, g, b, okR && okG && okB
}

// logoColor returns the color to use for the logo from BITERRA_LOGO_COLOR.
// Supported: hex (#e7c818 or e7c818), hi-yellow (default), yellow, green, red, cyan, blue, magenta, white, or hi- variants.
func logoColor() *color.Color {
	name := strings.TrimSpace(os.Getenv("BITERRA_LOGO_COLOR"))
	if name == "" {
		name = "hi-yellow"
	}
	if strings.HasPrefix(name, "#") || (len(name) == 6 && isHex(name)) {
		// Hex handled in LogoPrint via 24-bit ANSI
		return color.New(color.FgHiYellow)
	}
	name = strings.ToLower(name)
	switch name {
	case "green":
		return color.New(color.FgGreen)
	case "red":
		return color.New(color.FgRed)
	case "yellow":
		return color.New(color.FgYellow)
	case "cyan":
		return color.New(color.FgCyan)
	case "blue":
		return color.New(color.FgBlue)
	case "magenta":
		return color.New(color.FgMagenta)
	case "white":
		return color.New(color.FgWhite)
	case "hi-green", "higreen":
		return color.New(color.FgHiGreen)
	case "hi-red", "hired":
		return color.New(color.FgHiRed)
	case "hi-yellow", "hiyellow":
		return color.New(color.FgHiYellow)
	case "hi-cyan", "hicyan":
		return color.New(color.FgHiCyan)
	case "hi-blue", "hiblue":
		return color.New(color.FgHiBlue)
	case "hi-magenta", "himagenta":
		return color.New(color.FgHiMagenta)
	case "hi-white", "hiwhite":
		return color.New(color.FgHiWhite)
	default:
		return color.New(color.FgHiYellow)
	}
}

func isHex(s string) bool {
	for _, c := range s {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

// ansiHexForeground returns ANSI 24-bit foreground escape for r,g,b; empty if color disabled.
func ansiHexForeground(r, g, b uint8) string {
	if color.NoColor {
		return ""
	}
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

// ansiHexForegroundLogo always returns the 24-bit escape (for logo) so it stays yellow even when NoColor is set elsewhere.
func ansiHexForegroundLogo(r, g, b uint8) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", r, g, b)
}

const ansiReset = "\033[0m"

// ansiBrightYellow is SGR 93 — works in Terminal.app and virtually all color terminals (24-bit often does not).
const ansiBrightYellow = "\033[93m"

const esc = "\033"

// defaultLogoHex is used when BITERRA_LOGO_COLOR is set to a hex (e.g. #e7c818).
const defaultLogoHex = "e7c818"

// stripEsc removes any ESC (0x1b) from s so pasted logo lines can't break the terminal color.
func stripEsc(s string) string {
	return strings.ReplaceAll(s, esc, "")
}

// LogoPrint prints the ASCII art Biterra logo. Default: bright yellow (ANSI 93). Set BITERRA_LOGO_COLOR to a hex or name to override.
func LogoPrint() {
	syncOut()
	env := strings.TrimSpace(os.Getenv("BITERRA_LOGO_COLOR"))
	// Default: always use ANSI bright yellow so the logo is actually yellow in Terminal and everywhere else
	if env == "" {
		for _, line := range Logo {
			fmt.Fprint(os.Stdout, ansiBrightYellow, stripEsc(line), ansiReset, "\n")
		}
		return
	}
	if r, g, b, ok := parseHexColor(env); ok {
		seq := ansiHexForegroundLogo(r, g, b)
		for _, line := range Logo {
			fmt.Fprint(os.Stdout, seq, stripEsc(line), ansiReset, "\n")
		}
		return
	}
	c := logoColor()
	for _, line := range Logo {
		_, _ = c.Fprintln(os.Stdout, stripEsc(line))
	}
}

// Header prints the logo, then a bold title with an optional muted subtitle.
func Header(title, subtitle string) {
	LogoPrint()
	Blank()
	syncOut()
	bold.Println(title)
	if subtitle != "" {
		muted.Println(subtitle)
	}
}

// KeyValue prints "  label: value" with label muted and value in default/cyan.
func KeyValue(label, value string) {
	syncOut()
	muted.Printf("  %s: ", label)
	info.Println(value)
}

// Block prints a left-edge bar and indented message (for error/next-steps blocks).
func Block(lines []string) {
	syncOut()
	for _, s := range lines {
		muted.Print("  │ ")
		fmt.Fprintln(os.Stdout, s)
	}
}

// SuccessBlock prints a rule, success message, and muted lines, then a rule.
func SuccessBlock(successMsg string, mutedLines []string) {
	Blank()
	Rule()
	syncOut()
	success.Printf("  ✓ %s\n", successMsg)
	for _, s := range mutedLines {
		muted.Printf("    %s\n", s)
	}
	Rule()
}

// Line prints a plain line (no color).
func Line(format string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Blank prints a blank line.
func Blank() {
	fmt.Fprintln(os.Stdout)
}

// Bullet prints a muted bullet and message.
func Bullet(format string, args ...interface{}) {
	syncOut()
	muted.Print("  • ")
	fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// NextSteps prints a "Next steps:" header and bullet lines.
func NextSteps(header string, bullets []string) {
	syncOut()
	muted.Println("  " + header)
	for _, b := range bullets {
		muted.Print("  • ")
		fmt.Fprintln(os.Stdout, b)
	}
}

// ErrorBlock prints a rule, error message in a bar block, then "Next steps" with bullets, then rule.
func ErrorBlock(errMsg string, nextSteps []string) {
	Blank()
	Rule()
	syncOut()
	errorC.Print("  ✗ ")
	fmt.Fprintln(os.Stdout, errMsg)
	Blank()
	syncOut()
	muted.Println("  Next steps:")
	for _, s := range nextSteps {
		muted.Print("    • ")
		fmt.Fprintln(os.Stdout, s)
	}
	Rule()
}

// Section prints a bold section title (e.g. "Teams").
func Section(title string) {
	Blank()
	syncOut()
	bold.Println(title + ":")
}

// Option prints a numbered option (e.g. "  1. Name (uid)").
func Option(i int, name, detail string) {
	fmt.Fprintf(os.Stdout, "  %d. %s", i, name)
	if detail != "" {
		syncOut()
		muted.Printf(" (%s)", detail)
	}
	fmt.Fprintln(os.Stdout)
}

// CheckOK prints the short "OK" status line for biterra check (green check + message).
func CheckOK(format string, args ...interface{}) {
	syncOut()
	success.Printf("✓ "+format+"\n", args...)
}

// CheckStatus prints a small status block: rule, key-value rows, rule (for biterra check).
func CheckStatus(tokenStatus, roundStatus string) {
	Rule()
	syncOut()
	muted.Print("  Token  ")
	success.Println(tokenStatus)
	muted.Print("  Round  ")
	fmt.Fprintln(os.Stdout, roundStatus)
	Rule()
}
