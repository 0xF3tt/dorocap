package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		usage(true)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		fmt.Fprint(os.Stderr, banner())
		err = cmdInit(os.Args[2:])
	case "ss":
		err = cmdScreenshot(os.Args[2:])
	case "note":
		err = cmdNote(os.Args[2:])
	case "finding":
		err = cmdFinding(os.Args[2:])
	case "export":
		fmt.Fprint(os.Stderr, banner())
		err = cmdExport(os.Args[2:])
	case "finalize":
		fmt.Fprint(os.Stderr, banner())
		err = cmdFinalize(os.Args[2:])
	case "verify":
		err = cmdVerify(os.Args[2:])
	case "path":
		err = cmdPath(os.Args[2:])
	case "info":
		err = cmdInfo(os.Args[2:])
	case "completion":
		err = cmdCompletion(os.Args[2:])
	case "version", "-v", "--version":
		if len(os.Args) != 2 {
			err = fmt.Errorf("usage: dorocap version")
		} else {
			fmt.Println("dorocap", version)
		}
	case "help", "-h", "--help":
		if len(os.Args) != 2 {
			err = fmt.Errorf("usage: dorocap help")
		} else {
			usage(true)
		}
	default:
		err = fmt.Errorf("unknown command %q; run `dorocap help`", os.Args[1])
	}

	if err != nil {
		printError(os.Stderr, "dorocap: %v", err)
		os.Exit(1)
	}
}

func usage(showBanner bool) {
	if showBanner {
		fmt.Fprint(os.Stderr, banner())
	}
	fmt.Fprintln(os.Stderr, `Usage:
  dorocap init <engagement-name>				scaffold a new engagement folder
  dorocap ss <type> [note...]				capture a screenshot into evidence/<type>/
  dorocap ss file <src-path> [note...]		copy a file into evidence/files/
  dorocap note <type> <text...>				append a timestamped note
  dorocap finding add --interactive            prompt for finding details one by one
  dorocap finding add <title>                  [--severity value] [--asset value] [--status value]
  dorocap finding set <id> --interactive       prompt to update metadata; Enter keeps current values
  dorocap finding set <id> <field> <value>     update asset, risk, status, or retest metadata
  dorocap finding link <id> <evidence-path>    add supporting evidence [--retest]
  dorocap finding list
  dorocap export                                write findings + evidence to report/draft/report.md
  dorocap finalize                              verify and copy the reviewed draft to report/final/report.md
  dorocap verify                                verify evidence sidecars and SHA-256 hashes
  dorocap path [<dir>]                          show or set the global engagement path (used when outside any engagement)
  dorocap info                                show the active engagement and how it was resolved
  dorocap completion <shell>                  generate zsh, bash, fish, or powershell autocomplete`)
}
