package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func cmdCompletion(args []string) error {
	if len(args) == 1 {
		script, err := completionScript(args[0])
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	}
	if len(args) >= 2 && args[0] == "candidates" {
		return printCompletionCandidates(args[1:])
	}
	return fmt.Errorf("usage: dorocap completion <zsh|bash|fish|powershell>")
}

func completionScript(shell string) (string, error) {
	switch shell {
	case "zsh":
		return zshCompletion, nil
	case "bash":
		return bashCompletion, nil
	case "fish":
		return fishCompletion, nil
	case "powershell":
		return powershellCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell %q: use zsh, bash, fish, or powershell", shell)
	}
}

func printCompletionCandidates(args []string) error {
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("usage: dorocap completion candidates <findings|evidence> [finding-id]")
	}
	root, err := findRoot()
	if err != nil {
		return err
	}
	var candidates []string
	switch args[0] {
	case "findings":
		candidates, err = findingCompletionCandidates(root)
	case "evidence":
		findingID := ""
		if len(args) == 2 {
			findingID = args[1]
		}
		candidates, err = evidenceCompletionCandidates(root, findingID)
	default:
		return fmt.Errorf("unknown completion candidate type %q", args[0])
	}
	if err != nil {
		return err
	}
	for _, candidate := range candidates {
		if !strings.ContainsAny(candidate, "\r\n") {
			fmt.Println(candidate)
		}
	}
	return nil
}

func findingCompletionCandidates(root string) ([]string, error) {
	dir := filepath.Join(root, "findings")
	names, err := findingFiles(dir)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(names))
	for _, name := range names {
		finding, err := parseFinding(filepath.Join(dir, name))
		if err == nil {
			ids = append(ids, finding.ID)
		}
	}
	return ids, nil
}

func evidenceCompletionCandidates(root, findingID string) ([]string, error) {
	excluded := map[string]bool{}
	if findingID != "" {
		dir := filepath.Join(root, "findings")
		if path, err := findingPath(dir, findingID); err == nil {
			if finding, err := parseFinding(path); err == nil {
				allLinks := append(append([]string{}, finding.Evidence...), finding.RetestEvidence...)
				for _, linked := range allLinks {
					excluded[linked] = true
				}
			}
		}
	}
	evidence, _ := inspectEvidence(root)
	paths := make([]string, 0, len(evidence))
	for _, item := range evidence {
		if !excluded[item.Path] {
			paths = append(paths, item.Path)
		}
	}
	return paths, nil
}

const zshCompletion = `#compdef dorocap

_dorocap_dynamic() {
  local kind=$1 finding_id=$2
  local -a values
  if [[ -n $finding_id ]]; then
    values=("${(@f)$("${words[1]}" completion candidates "$kind" "$finding_id" 2>/dev/null)}")
  else
    values=("${(@f)$("${words[1]}" completion candidates "$kind" 2>/dev/null)}")
  fi
  (( ${#values} )) && compadd -- "${values[@]}"
}

_dorocap() {
  local -a commands finding_commands types severities statuses retest_statuses shells
  commands=(
    'init:create an engagement and make it the global default'
    'ss:capture a screenshot or import a file'
    'note:add a timestamped note'
    'finding:add, link, or list findings'
    'export:generate the draft report'
	'finalize:verify and promote the reviewed draft report'
    'verify:verify evidence integrity'
    'path:show or set the global engagement path'
    'info:show the active engagement'
    'completion:generate shell completion'
    'version:show the installed version'
    'help:show help'
    '-h:show help'
    '--help:show help'
    '-v:show the installed version'
    '--version:show the installed version'
  )
	finding_commands=('add:create a finding' 'set:update finding metadata' 'link:link evidence to a finding' 'list:list findings')
  types=(recon staging exploitation postex files)
  severities=(crit high med low info)
	statuses=(open resolved partially-resolved accepted-risk not-applicable)
	retest_statuses=(not-tested resolved still-vulnerable partially-resolved)
  shells=(zsh bash fish powershell)

  if (( CURRENT == 2 )); then
    _describe 'command' commands
    return
  fi
  case ${words[2]} in
    ss)
      if (( CURRENT == 3 )); then
        compadd -- file recon staging exploitation postex
      elif [[ ${words[3]} == file && CURRENT -eq 4 ]]; then
        _files
      fi
      ;;
    note)
      (( CURRENT == 3 )) && compadd -- "${types[@]}"
      ;;
    finding)
      if (( CURRENT == 3 )); then
        _describe 'finding command' finding_commands
      elif [[ ${words[3]} == link && CURRENT -eq 4 ]]; then
        _dorocap_dynamic findings
      elif [[ ${words[3]} == link && CURRENT -eq 5 ]]; then
        _dorocap_dynamic evidence "${words[4]}"
	  elif [[ ${words[3]} == link && CURRENT -eq 6 ]]; then
		compadd -- --retest
	  elif [[ ${words[3]} == set && CURRENT -eq 4 ]]; then
		_dorocap_dynamic findings
	  elif [[ ${words[3]} == set && CURRENT -eq 5 ]]; then
		compadd -- --interactive -i severity status asset cvss cwe owasp retest-status retested
	  elif [[ ${words[3]} == set && CURRENT -eq 6 && ${words[5]} == severity ]]; then
		compadd -- "${severities[@]}"
	  elif [[ ${words[3]} == set && CURRENT -eq 6 && ${words[5]} == status ]]; then
		compadd -- "${statuses[@]}"
	  elif [[ ${words[3]} == set && CURRENT -eq 6 && ${words[5]} == retest-status ]]; then
		compadd -- "${retest_statuses[@]}"
      elif [[ ${words[3]} == add && ${words[CURRENT-1]} == --severity ]]; then
        compadd -- "${severities[@]}"
	  elif [[ ${words[3]} == add && ${words[CURRENT-1]} == --status ]]; then
		compadd -- "${statuses[@]}"
      elif [[ ${words[3]} == add && ${words[CURRENT]} == --severity=* ]]; then
        compadd -- --severity=crit --severity=high --severity=med --severity=low --severity=info
      elif [[ ${words[3]} == add && ${words[CURRENT]} == -* ]]; then
		compadd -- --interactive -i --severity --severity= --status --status= --asset --asset= --cvss --cvss= --cwe --cwe= --owasp --owasp=
      fi
      ;;
    path)
      (( CURRENT == 3 )) && _directories
      ;;
    completion)
      (( CURRENT == 3 )) && compadd -- "${shells[@]}"
      ;;
  esac
}

compdef _dorocap dorocap
`

const bashCompletion = `_dorocap_complete() {
  local cur prev command sub candidate
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  command="${COMP_WORDS[1]}"
  sub="${COMP_WORDS[2]}"

  if [[ $COMP_CWORD -eq 1 ]]; then
	COMPREPLY=( $(compgen -W 'init ss note finding export finalize verify path info completion version help -h --help -v --version' -- "$cur") )
    return
  fi
  case "$command" in
    ss)
      if [[ $COMP_CWORD -eq 2 ]]; then
        COMPREPLY=( $(compgen -W 'file recon staging exploitation postex' -- "$cur") )
      elif [[ $sub == file && $COMP_CWORD -eq 3 ]]; then
        COMPREPLY=( $(compgen -f -- "$cur") )
        compopt -o filenames 2>/dev/null
      fi
      ;;
    note)
      [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W 'recon staging exploitation postex files' -- "$cur") )
      ;;
    finding)
      if [[ $COMP_CWORD -eq 2 ]]; then
		COMPREPLY=( $(compgen -W 'add set link list' -- "$cur") )
      elif [[ $sub == link && $COMP_CWORD -eq 3 ]]; then
        while IFS= read -r candidate; do
          [[ $candidate == "$cur"* ]] && COMPREPLY+=("$candidate")
        done < <("${COMP_WORDS[0]}" completion candidates findings 2>/dev/null)
      elif [[ $sub == link && $COMP_CWORD -eq 4 ]]; then
        while IFS= read -r candidate; do
          [[ $candidate == "$cur"* ]] && COMPREPLY+=("$candidate")
        done < <("${COMP_WORDS[0]}" completion candidates evidence "${COMP_WORDS[3]}" 2>/dev/null)
        compopt -o filenames 2>/dev/null
	  elif [[ $sub == link && $COMP_CWORD -eq 5 ]]; then
		COMPREPLY=( $(compgen -W '--retest' -- "$cur") )
	  elif [[ $sub == set && $COMP_CWORD -eq 3 ]]; then
		while IFS= read -r candidate; do
		  [[ $candidate == "$cur"* ]] && COMPREPLY+=("$candidate")
		done < <("${COMP_WORDS[0]}" completion candidates findings 2>/dev/null)
	  elif [[ $sub == set && $COMP_CWORD -eq 4 ]]; then
		COMPREPLY=( $(compgen -W '--interactive -i severity status asset cvss cwe owasp retest-status retested' -- "$cur") )
	  elif [[ $sub == set && $COMP_CWORD -eq 5 && ${COMP_WORDS[4]} == severity ]]; then
		COMPREPLY=( $(compgen -W 'crit high med low info' -- "$cur") )
	  elif [[ $sub == set && $COMP_CWORD -eq 5 && ${COMP_WORDS[4]} == status ]]; then
		COMPREPLY=( $(compgen -W 'open resolved partially-resolved accepted-risk not-applicable' -- "$cur") )
	  elif [[ $sub == set && $COMP_CWORD -eq 5 && ${COMP_WORDS[4]} == retest-status ]]; then
		COMPREPLY=( $(compgen -W 'not-tested resolved still-vulnerable partially-resolved' -- "$cur") )
      elif [[ $sub == add && $prev == --severity ]]; then
        COMPREPLY=( $(compgen -W 'crit high med low info' -- "$cur") )
	  elif [[ $sub == add && $prev == --status ]]; then
		COMPREPLY=( $(compgen -W 'open resolved partially-resolved accepted-risk not-applicable' -- "$cur") )
      elif [[ $sub == add && $cur == --severity=* ]]; then
        COMPREPLY=( $(compgen -W '--severity=crit --severity=high --severity=med --severity=low --severity=info' -- "$cur") )
      elif [[ $sub == add && $cur == -* ]]; then
		COMPREPLY=( $(compgen -W '--interactive -i --severity --severity= --status --status= --asset --asset= --cvss --cvss= --cwe --cwe= --owasp --owasp=' -- "$cur") )
      fi
      ;;
    path)
      [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -d -- "$cur") )
      ;;
    completion)
      [[ $COMP_CWORD -eq 2 ]] && COMPREPLY=( $(compgen -W 'zsh bash fish powershell' -- "$cur") )
      ;;
  esac
}

complete -F _dorocap_complete dorocap
`

const fishCompletion = `function __dorocap_needs_command
    test (count (commandline -opc)) -eq 1
end

function __dorocap_at
    set -l words (commandline -opc)
    if test (count $words) -ne (math $argv[1] - 1)
        return 1
    end
    if test "$words[2]" != "$argv[2]"
        return 1
    end
    if test (count $argv) -ge 3; and test "$words[3]" != "$argv[3]"
        return 1
    end
    return 0
end

function __dorocap_set_field
    set -l words (commandline -opc)
    test (count $words) -eq 5
    and test "$words[2]" = finding
    and test "$words[3]" = set
    and test "$words[5]" = "$argv[1]"
end

complete -c dorocap -f
complete -c dorocap -n __dorocap_needs_command -a init -d 'Create an engagement and make it the global default'
complete -c dorocap -n __dorocap_needs_command -a ss -d 'Capture a screenshot or import a file'
complete -c dorocap -n __dorocap_needs_command -a note -d 'Add a timestamped note'
complete -c dorocap -n __dorocap_needs_command -a finding -d 'Add, link, or list findings'
complete -c dorocap -n __dorocap_needs_command -a 'export finalize verify path info completion version help'
complete -c dorocap -n __dorocap_needs_command -a '-h --help -v --version'

complete -c dorocap -n '__dorocap_at 3 ss' -a 'file recon staging exploitation postex'
complete -c dorocap -n '__dorocap_at 4 ss file' -F
complete -c dorocap -n '__dorocap_at 3 note' -a 'recon staging exploitation postex files'
complete -c dorocap -n '__dorocap_at 3 finding' -a 'add set link list'
complete -c dorocap -n '__dorocap_at 4 finding link' -a '(dorocap completion candidates findings 2>/dev/null)'
complete -c dorocap -n '__dorocap_at 5 finding link' -a '(dorocap completion candidates evidence (commandline -opc)[4] 2>/dev/null)'
complete -c dorocap -n '__dorocap_at 6 finding link' -a --retest
complete -c dorocap -n '__dorocap_at 4 finding set' -a '(dorocap completion candidates findings 2>/dev/null)'
complete -c dorocap -n '__dorocap_at 5 finding set' -a '--interactive -i severity status asset cvss cwe owasp retest-status retested'
complete -c dorocap -n '__dorocap_set_field severity' -a 'crit high med low info'
complete -c dorocap -n '__dorocap_set_field status' -a 'open resolved partially-resolved accepted-risk not-applicable'
complete -c dorocap -n '__dorocap_set_field retest-status' -a 'not-tested resolved still-vulnerable partially-resolved'
complete -c dorocap -n '__dorocap_at 3 path' -a '(__fish_complete_directories (commandline -ct))'
complete -c dorocap -n '__dorocap_at 3 completion' -a 'zsh bash fish powershell'
complete -c dorocap -n '__fish_seen_subcommand_from add' -l severity -a 'crit high med low info'
complete -c dorocap -n '__fish_seen_subcommand_from add' -s i -l interactive -d 'Prompt for finding details one by one'
complete -c dorocap -n '__fish_seen_subcommand_from add' -l status -a 'open resolved partially-resolved accepted-risk not-applicable'
complete -c dorocap -n '__fish_seen_subcommand_from add' -l asset -r
complete -c dorocap -n '__fish_seen_subcommand_from add' -l cvss -r
complete -c dorocap -n '__fish_seen_subcommand_from add' -l cwe -r
complete -c dorocap -n '__fish_seen_subcommand_from add' -l owasp -r
`

const powershellCompletion = `Register-ArgumentCompleter -Native -CommandName dorocap -ScriptBlock {
    param($wordToComplete, $commandAst, $cursorPosition)

    $words = @($commandAst.CommandElements | ForEach-Object { $_.Value })
	$commands = @('init', 'ss', 'note', 'finding', 'export', 'finalize', 'verify', 'path', 'info', 'completion', 'version', 'help', '-h', '--help', '-v', '--version')
    $categories = @('recon', 'staging', 'exploitation', 'postex')
    $severities = @('crit', 'high', 'med', 'low', 'info')
    $candidates = @()
	$index = $words.Count
	if ($wordToComplete -ne '' -and $words.Count -gt 0 -and $words[-1] -eq $wordToComplete) {
		$index--
	}

    if ($index -eq 1) {
        $candidates = $commands
    } else {
        $command = $words[1]
        switch ($command) {
            'ss' {
				if ($index -eq 2) {
                    $candidates = @('file') + $categories
				} elseif ($words[2] -eq 'file' -and $index -eq 3) {
                    $candidates = Get-ChildItem -Name -Path ($wordToComplete + '*') -ErrorAction SilentlyContinue
                }
            }
            'note' {
				if ($index -eq 2) { $candidates = $categories + @('files') }
            }
            'finding' {
				if ($index -eq 2) {
					$candidates = @('add', 'set', 'link', 'list')
                } elseif ($words[2] -eq 'link') {
					if ($index -eq 3) {
                        $candidates = & dorocap completion candidates findings 2>$null
					} elseif ($index -eq 4) {
                        $candidates = & dorocap completion candidates evidence $words[3] 2>$null
					} elseif ($index -eq 5) {
						$candidates = @('--retest')
                    }
				} elseif ($words[2] -eq 'add') {
					if ($index -gt 0 -and $words[$index - 1] -eq '--severity') {
                        $candidates = $severities
					} elseif ($index -gt 0 -and $words[$index - 1] -eq '--status') {
						$candidates = @('open', 'resolved', 'partially-resolved', 'accepted-risk', 'not-applicable')
                    } elseif ($wordToComplete.StartsWith('--severity=')) {
                        $candidates = $severities | ForEach-Object { '--severity=' + $_ }
                    } elseif ($wordToComplete.StartsWith('-')) {
						$candidates = @('--interactive', '-i', '--severity', '--severity=', '--status', '--status=', '--asset', '--asset=', '--cvss', '--cvss=', '--cwe', '--cwe=', '--owasp', '--owasp=')
                    }
				} elseif ($words[2] -eq 'set') {
					if ($index -eq 3) {
						$candidates = & dorocap completion candidates findings 2>$null
					} elseif ($index -eq 4) {
						$candidates = @('--interactive', '-i', 'severity', 'status', 'asset', 'cvss', 'cwe', 'owasp', 'retest-status', 'retested')
					} elseif ($index -eq 5 -and $words[4] -eq 'severity') {
						$candidates = $severities
					} elseif ($index -eq 5 -and $words[4] -eq 'status') {
						$candidates = @('open', 'resolved', 'partially-resolved', 'accepted-risk', 'not-applicable')
					} elseif ($index -eq 5 -and $words[4] -eq 'retest-status') {
						$candidates = @('not-tested', 'resolved', 'still-vulnerable', 'partially-resolved')
					}
                }
            }
            'path' {
				if ($index -eq 2) {
                    $candidates = Get-ChildItem -Directory -Name -Path ($wordToComplete + '*') -ErrorAction SilentlyContinue
                }
            }
            'completion' {
				if ($index -eq 2) { $candidates = @('zsh', 'bash', 'fish', 'powershell') }
            }
        }
    }

    $candidates | Where-Object {
        $_ -and $_.StartsWith($wordToComplete, [System.StringComparison]::OrdinalIgnoreCase)
    } | ForEach-Object {
        [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
    }
}
`
