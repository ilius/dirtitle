package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"syscall"
)

var titleSep = " ‹- "

/*
	" ‹- "	no flashes, but too small
	" ᐊ "	flashes (different font)
	" ◂ "	flashes (different font)
	" ◀ "	flashes (different font)
	" ≺ "	flashes (different font)
	" ⇽ "	flashes (different font), and too wide
	" ⇠ "	flashes (different font)
*/

var userObj *user.User // set on init

func init() {
	var err error
	userObj, err = user.Current()
	if err != nil {
		exitWithError(err)
	}
	{
		value := os.Getenv("DIRTITLE_SEP")
		if value != "" {
			titleSep = value
		}
	}
}

func exitWithError(err interface{}) {
	os.Stderr.WriteString(fmt.Sprintf("dirtitle: %v\n", err))
	os.Exit(1)
}

func getConfDir() string {
	memDir := filepath.Join("/run/shm", userObj.Username, ".dirtitle")
	stat, err := os.Stat(memDir)
	if err == nil && stat.IsDir() {
		return memDir
	}
	return filepath.Join(userObj.HomeDir, ".dirtitle")
}

// returns (title, stopHere, err)
func readTitleFile(dpath string) (string, bool, error) {
	data, err := os.ReadFile(filepath.Join(getConfDir(), dpath+".title"))
	if err == nil {
		title := string(bytes.TrimSpace(data))
		if title == "" {
			title = "Terminal"
		}
		return title, true, nil
	}
	if os.IsNotExist(err) {
		return "", false, nil
	}
	return "", false, err
}

// returns (title, stopHere, err)
func getShortTitle(dpath string) (string, bool, error) {
	if dpath == userObj.HomeDir {
		return "~", true, nil
	}
	title, stopHere, err := readTitleFile(dpath)
	if err != nil {
		return "", false, err
	}
	if stopHere {
		return title, true, nil
	}
	i := strings.LastIndex(dpath, "/")
	if i < 0 {
		return "", false, fmt.Errorf("bad directory path %#v", dpath)
	}
	dname := dpath[i+1:]
	if dname == "" {
		return " ", true, nil
	}
	if dname[0] == '.' {
		return " ", true, nil
	}
	stat, err := os.Stat(dpath)
	if err != nil {
		if os.IsNotExist(err) || os.IsPermission(err) {
			return dpath, true, nil
		}
		return "", false, err
	}
	statSys, ok := stat.Sys().(*syscall.Stat_t)
	if ok {
		dirUid := statSys.Uid
		user, err := user.Current()
		if err != nil {
			exitWithError(err)
		}
		if user.Uid != "0" && fmt.Sprintf("%d", dirUid) != user.Uid {
			return dpath, true, nil
		}
	}
	return dname, false, nil
}

func getLongTitle(dpath string) (string, error) {
	if dpath == userObj.HomeDir {
		return "~", nil
	}
	title, stopHere, err := readTitleFile(dpath)
	if err != nil {
		return "", err
	}
	if stopHere {
		return title, nil
	}
	parts := strings.Split(dpath, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("bad directory path %#v", dpath)
	}

	titleParts := []string{}
	stopIndex := len(parts) - 3
	if stopIndex < 1 {
		stopIndex = 1
	}
	for i := len(parts); i > stopIndex; i -= 1 {
		titlePart, isSet, err := getShortTitle(strings.Join(parts[:i], "/"))
		if err != nil {
			return "", err
		}
		titleParts = append(titleParts, titlePart)
		if isSet {
			break
		}
	}

	return strings.Join(titleParts, titleSep), nil
}

func getRunningCommand() string {
	cmd := os.Getenv("BASH_COMMAND")
	if cmd != "" {
		parts := strings.Split(cmd, " ")
		cmdEx := parts[0]
		switch cmdEx {
		case ".", "source", "test", "[", "cd", "export", "eval", "printf":
			break
		default:
			switch {
			case strings.Contains(cmd, "\033]0"):
				// The command is trying to set the title bar as well;
				// this is most likely the execution of $PROMPT_COMMAND.
				// In any case nested escapes confuse the terminal, so don't output them.
				break
			case strings.Contains(cmd, "direnv"):
				break
			case strings.HasPrefix(cmd, "["):
				break
			case strings.Contains(cmd, "dirtitle"):
				break
			case strings.Contains(cmd, "dir-title"):
				break
			default:
				histLastCmd := os.Getenv("HIST_LAST_COMMAND")
				if histLastCmd != "" {
					return histLastCmd
				}
				return cmd
			}
		}

	}
	return ""
}

func getTitleWithOpts(dpath string, long bool, showCommand bool) string {
	if showCommand {
		cmd := getRunningCommand()
		if cmd != "" {
			title, _, err := getShortTitle(dpath)
			if err != nil {
				exitWithError(err)
			}
			return title + ": " + cmd
		}
	}
	if long {
		title, err := getLongTitle(dpath)
		if err != nil {
			exitWithError(err)
		}
		return title
	}
	title, _, err := getShortTitle(dpath)
	if err != nil {
		exitWithError(err)
	}
	return title
}

func main() {
	longFlag := flag.Bool(
		"long",
		false,
		"dirtitle -long DIR_PATH",
	)
	showCommandFlag := flag.Bool(
		"show-command",
		false,
		"dirtitle -show-command",
	)

	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		exitWithError("Usage: dirtitle [-long] DIR_PATH")
	}

	dpath := args[0]
	dpath, err := filepath.Abs(dpath)
	if err != nil {
		exitWithError(err)
	}

	title := getTitleWithOpts(
		dpath,
		longFlag != nil && *longFlag,
		showCommandFlag != nil && *showCommandFlag,
	)

	fmt.Printf("%s\n", title)
	// maybe later: "\033]0;${TITLE}\007"
}
