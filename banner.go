package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

const (
	clrBlue     = "\x1b[38;2;122;162;247m" // blue
	clrViolet   = "\x1b[38;2;168;140;240m" // violet
	clrCyan     = "\x1b[38;2;125;207;255m" // cyan
	clrCyanDeep = "\x1b[38;2;42;195;222m"  // cyanDeep
	clrText     = "\x1b[38;2;192;202;245m" // text
	clrMuted    = "\x1b[38;2;169;177;214m" // textMuted
	clrMagenta  = "\x1b[38;2;247;118;142m" // magenta
	clrOrange   = "\x1b[38;2;255;158;100m" // orange
	clrGreen    = "\x1b[38;2;158;206;106m" // green
	clrYellow   = "\x1b[38;2;224;175;104m" // yellow
	clrRed      = "\x1b[38;2;219;75;75m"   // red
	clrBold     = "\x1b[1m"
	clrReset    = "\x1b[0m"
)

type terminalColorMode uint8

const (
	colorNone terminalColorMode = iota
	colorANSI16
	colorTrue
)

func terminalColorModeFor(isTTY bool, goos string, env map[string]string) terminalColorMode {
	if !isTTY {
		return colorNone
	}
	if _, disabled := env["NO_COLOR"]; disabled || strings.EqualFold(env["TERM"], "dumb") {
		return colorNone
	}
	if goos == "windows" {
		modern := env["WT_SESSION"] != "" || env["ANSICON"] != "" || strings.EqualFold(env["ConEmuANSI"], "ON") || env["TERM"] != ""
		if !modern {
			return colorNone
		}
	}
	colorTerm := strings.ToLower(env["COLORTERM"])
	term := strings.ToLower(env["TERM"])
	if colorTerm == "truecolor" || colorTerm == "24bit" || strings.Contains(term, "truecolor") || strings.Contains(term, "direct") || env["WT_SESSION"] != "" {
		return colorTrue
	}
	return colorANSI16
}

func terminalEnvironment() map[string]string {
	env := make(map[string]string)
	for _, key := range []string{"NO_COLOR", "TERM", "COLORTERM", "WT_SESSION", "ANSICON", "ConEmuANSI"} {
		if value, ok := os.LookupEnv(key); ok {
			env[key] = value
		}
	}
	return env
}

func streamIsTTY(stream *os.File) bool {
	info, err := stream.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func streamColorMode(stream *os.File) terminalColorMode {
	return terminalColorModeFor(streamIsTTY(stream), runtime.GOOS, terminalEnvironment())
}

func colorEnabled(stream *os.File) bool {
	return streamColorMode(stream) != colorNone
}

func unicodeEnabled(stream *os.File) bool {
	return terminalUnicodeFor(streamIsTTY(stream), runtime.GOOS, environmentFor("DOROCAP_ASCII", "TERM", "LC_ALL", "LC_CTYPE", "LANG", "WT_SESSION", "ANSICON", "ConEmuANSI"))
}

func environmentFor(keys ...string) map[string]string {
	env := make(map[string]string)
	for _, key := range keys {
		if value, ok := os.LookupEnv(key); ok {
			env[key] = value
		}
	}
	return env
}

func terminalUnicodeFor(isTTY bool, goos string, env map[string]string) bool {
	if _, ascii := env["DOROCAP_ASCII"]; ascii || !isTTY || strings.EqualFold(env["TERM"], "dumb") {
		return false
	}
	locale := env["LC_ALL"]
	if locale == "" {
		locale = env["LC_CTYPE"]
	}
	if locale == "" {
		locale = env["LANG"]
	}
	locale = strings.ToLower(locale)
	if locale != "" && !strings.Contains(locale, "utf-8") && !strings.Contains(locale, "utf8") {
		return false
	}
	if goos == "windows" {
		return env["WT_SESSION"] != "" || env["ANSICON"] != "" || strings.EqualFold(env["ConEmuANSI"], "ON") || env["TERM"] != ""
	}
	return true
}

func nerdEnabled(stream *os.File) bool {
	if !colorEnabled(stream) {
		return false
	}
	if _, disabled := os.LookupEnv("DOROCAP_NO_NERD"); disabled {
		return false
	}
	if _, disabled := os.LookupEnv("NEUST_NO_NERD"); disabled {
		return false
	}
	return os.Getenv("DOROCAP_NERD") != "" || os.Getenv("NEUST_NERD") != ""
}

func banner() string {
	if !colorEnabled(os.Stderr) || !unicodeEnabled(os.Stderr) {
		return "+----------------------------------------\n" +
			"|   *  D O R O C A P\n" +
			"|   Collect. Preserve. Prove.\n" +
			"+--------------------------  + @0xF3tt\n\n"
	}
	star := "*"
	bat := "+"
	if nerdEnabled(os.Stderr) {
		star = "\uf4f5"
		bat = "\U000f0b5f"
	}
	return paint(os.Stderr, "╭────────────────────────────────────────", clrBlue) + "\n" +
		paint(os.Stderr, "│   ", clrBlue) + paint(os.Stderr, star+"  ", clrCyan) + paint(os.Stderr, "D O R O C A P", clrBold+clrViolet) + "\n" +
		paint(os.Stderr, "│   ", clrBlue) + paint(os.Stderr, "Collect. Preserve. Prove.", clrMuted) + "\n" +
		paint(os.Stderr, "╰──────────────────────────  ", clrBlue) + paint(os.Stderr, bat+" @0xF3tt", clrMagenta) + "\n\n"
}

func paint(stream *os.File, text, color string) string {
	mode := streamColorMode(stream)
	if mode == colorNone {
		return text
	}
	if mode == colorTrue {
		return color + text + clrReset
	}
	bold := strings.Contains(color, clrBold)
	base := strings.ReplaceAll(color, clrBold, "")
	ansi := map[string]string{
		clrBlue: "94", clrViolet: "95", clrCyan: "96", clrCyanDeep: "36",
		clrText: "97", clrMuted: "37", clrMagenta: "95", clrOrange: "93",
		clrGreen: "92", clrYellow: "93", clrRed: "91",
	}[base]
	if ansi == "" {
		return text
	}
	if bold {
		ansi = "1;" + ansi
	}
	return "\x1b[" + ansi + "m" + text + clrReset
}

func writeRole(stream *os.File, role, color, format string, args ...any) {
	icon := ""
	if nerdEnabled(stream) {
		icons := map[string]string{
			"info": "\U000f02fc ", "ok": "\U000f05e0 ", "warning": "\U000f0026 ", "error": "\U000f0159 ",
		}
		icon = icons[role]
	}
	message := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprintf(stream, "%s %s\n", paint(stream, icon+role+":", color), message)
}

func printInfo(format string, args ...any) {
	writeRole(os.Stdout, "info", clrCyanDeep, format, args...)
}

func printOK(format string, args ...any) {
	writeRole(os.Stdout, "ok", clrGreen, format, args...)
}

func printWarning(format string, args ...any) {
	writeRole(os.Stdout, "warning", clrYellow, format, args...)
}

func printError(w io.Writer, format string, args ...any) {
	if stream, ok := w.(*os.File); ok {
		writeRole(stream, "error", clrRed, format, args...)
		return
	}
	_, _ = fmt.Fprintf(w, "error: "+format+"\n", args...)
}

func severityColor(severity string) string {
	switch severity {
	case "crit":
		return clrRed
	case "high":
		return clrOrange
	case "med":
		return clrYellow
	case "low":
		return clrGreen
	default:
		return clrCyanDeep
	}
}
