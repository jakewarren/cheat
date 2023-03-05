//nolint:funlen
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/pflag"
)

var (
	// build information set by ldflags
	appName   = "cheat"
	version   = "(ﾉ☉ヮ⚆)ﾉ ⌒*:･ﾟ✧"
	commit    = "(ﾉ☉ヮ⚆)ﾉ ⌒*:･ﾟ✧"
	buildDate = "(ﾉ☉ヮ⚆)ﾉ ⌒*:･ﾟ✧"
)

const usageMessage = `Usage: cheat [flags] [<command>]`

const usageExample = `Examples:
	- To see example usage of a program:
	cheat <command>
	
	- To edit a cheatsheet
	cheat -e <command>
	
	- To list available cheatsheets
	cheat -l
	
	- To search specific cheatsheet for a search term
	cheat -s <search term regex> <command>
	
	- To search all available cheatsheets
	cheat -s <search term regex>`

func main() {
	pflag.Usage = func() {
		_, _ = fmt.Fprintln(os.Stderr, usageMessage)
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "Flags:")
		pflag.PrintDefaults()
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, usageExample)
		_, _ = fmt.Fprintln(os.Stderr, "")
		_, _ = fmt.Fprintln(os.Stderr, "URL: https://github.com/jakewarren/cheat")
	}

	editCheatSheet := pflag.BoolP("edit", "e", false, "Edit cheatsheet")
	listCheatSheet := pflag.BoolP("list", "l", false, "List cheatsheets")
	searchCheatSheet := pflag.StringP("search", "s", "", "Search cheatsheet")
	displayHelp := pflag.BoolP("help", "h", false, "display help")
	displayVersion := pflag.BoolP("version", "V", false, "display version information")
	pflag.Parse()

	// override the default usage display
	if *displayHelp {
		pflag.Usage()
		os.Exit(0)
	}

	if *displayVersion {
		fmt.Printf(`%s:
    version     : %s
    git hash    : %s
    build date  : %s 
    go version  : %s
    go compiler : %s
    platform    : %s/%s
`, appName, version, commit, buildDate, runtime.Version(), runtime.Compiler, runtime.GOOS, runtime.GOARCH)
		os.Exit(0)
	}

	config := &JSONData{}
	configErr := config.ReadConfig()
	if configErr != nil {
		log.Fatal().Err(configErr).Msg("error reading config")
	}

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	if *listCheatSheet {
		listCheatSheets(config.Cheatdirs)
		os.Exit(0)
	}

	if len(*searchCheatSheet) > 0 && len(pflag.Args()) == 0 {
		searchAllCheatSheets(config.Cheatdirs, *searchCheatSheet)
		os.Exit(0)
	}

	if len(pflag.Args()) == 0 {
		pflag.Usage()
		os.Exit(0)
	}

	// disable color if the user requests it
	if !config.Highlight {
		color.NoColor = true
	}

	cmdname := pflag.Arg(0)

	cheatfile := locateCheatSheet(config, cmdname)

	if len(*searchCheatSheet) > 0 {
		err := searchCheatFile(cheatfile, *searchCheatSheet)
		if err != nil {
			log.Error().Err(err).Msg("error searching cheatsheet")
		}
		os.Exit(0)
	}

	if *editCheatSheet {
		err := editCheat(cheatfile, config)
		if err != nil {
			log.Error().Err(err).Msg("error editing cheatsheet")
		}

		os.Exit(0)
	}

	if !doesFileExist(cheatfile) {
		cheatsheetNotFound(cmdname)
	}

	err := printCheatFile(cheatfile)
	if err != nil {
		log.Error().Err(err).Msg("error printing cheatsheet")
	}
}

func listCheatSheets(cheatdirs []string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)

	for _, dir := range cheatdirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Fatal().Err(err).Msg("error reading directory")
		}

		for _, f := range files {
			// exclude python files like __init__.py
			if strings.HasPrefix(f.Name(), "__") {
				continue
			}

			fmt.Fprintf(w, "%s\t%s\n", f.Name(), filepath.Join(dir, f.Name()))
		}
	}

	w.Flush()
}

func searchAllCheatSheets(cheatdirs []string, searchterm string) {
	for _, dir := range cheatdirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			log.Fatal().Err(err).Msg("error reading directory")
		}

		for _, f := range files {
			// exclude python files like __init__.py
			if strings.HasPrefix(f.Name(), "__") {
				continue
			}

			if searchErr := searchCheatFile(filepath.Join(dir, f.Name()), searchterm); searchErr != nil {
				log.Error().Err(searchErr).Str("file", filepath.Join(dir, f.Name())).Msg("error searching cheat file")
			}
		}
	}
}

func locateCheatSheet(config *JSONData, cmdname string) string {
	for i := range config.Cheatdirs {
		if cheatFile := filepath.Join(config.Cheatdirs[i], cmdname); doesFileExist(cheatFile) {
			return cheatFile
		}
	}
	return filepath.Join(config.Cheatdirs[0], cmdname)
}

func searchCheatFile(cheatfile, searchterm string) error {
	file, err := os.Open(cheatfile)
	if err != nil {
		return errors.Wrap(err, "error opening cheat file")
	}

	text, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.Wrap(err, "error reading cheat file")
	}

	blocks := string2Blocks(text)

	result := make([]string, 0)

	r, reErr := regexp.Compile(searchterm)
	if reErr != nil {
		return reErr
	}

	for _, b := range blocks {
		t := strings.Join(b, "\n")
		if r.MatchString(t) {
			result = append(result, t, "")
		}
	}

	if len(result) > 0 {
		color.New(color.FgHiGreen, color.Underline).Printf("\n%s\n", cheatfile)
		fmt.Println(pretty(strings.TrimSpace(strings.Join(result, "\n"))))
	}

	return nil
}

// string2Blocks converts a cheat file into blocks of "entries"
//     each "block" should contain the command and associated comment lines
func string2Blocks(in []byte) [][]string {
	var (
		blocks = make([][]string, 1)
		index  = 0
	)
	scanner := bufio.NewScanner(bytes.NewReader(in))
	blocks[index] = make([]string, 0)
	for scanner.Scan() {
		currentLine := scanner.Text()
		if currentLine == "" {
			index++
			blocks = append(blocks, []string{})
			continue
		}
		blocks[index] = append(blocks[index], currentLine)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return blocks
}

func editCheat(cheatfile string, config *JSONData) error {
	cheatFileDir := filepath.Dir(cheatfile)
	cheatFileName := filepath.Base(cheatfile)

	// check if the user is editing a cheatsheet from outside their primary directory
	if cheatFileDir != config.Cheatdirs[0] {
		// create a copy in the primary directory
		srcFile, err := os.Open(cheatfile)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not open %s", cheatfile))
		}
		defer srcFile.Close()

		cheatfile = filepath.Join(config.Cheatdirs[0], cheatFileName)
		dstFile, err := os.Create(cheatfile)
		if err != nil {
			return err
		}
		defer dstFile.Close()

		_, err = io.Copy(dstFile, srcFile)
		if err != nil {
			return err
		}

		err = dstFile.Sync()
		if err != nil {
			return err
		}
	}

	// open the editor
	editor, err := exec.LookPath(config.Editor)
	if err != nil {
		return errors.Wrap(err, "editor not found")
	}

	cmd := exec.Command(editor, cheatfile)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func printCheatFile(cheatfile string) error {
	file, err := os.Open(cheatfile)
	if err != nil {
		return errors.Wrap(err, "error opening cheat file")
	}

	text, err := ioutil.ReadAll(file)
	if err != nil {
		return errors.Wrap(err, "error reading cheat file")
	}

	fmt.Println(pretty(string(text)))

	return nil
}

func pretty(s string) string {
	sep := "\n"
	lastLine := ""
	prettyLines := make([]string, 0)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		l := scanner.Text()
		if len(lastLine) > 0 && lastLine[0] == '-' && l == "" {
			continue
		}
		lastLine = l
		if len(l) > 0 {
			// look for our snippet syntax start
			if len(l) > 3 && l[0:3] == "#--" {
				// print the snippet header
				l = "┏━━━〘" + l[4:] + " 〙━━●"
				l = color.YellowString(l)
				prettyLines = append(prettyLines, l)

				// read the rest of the snippet until we reach the end syntax
				for scanner.Scan() {
					l := scanner.Text()

					if len(l) >= 4 && l[0:4] == "#--#" {
						prettyLines = append(prettyLines, color.YellowString("┗●"))
						break
					}

					// format the body of the snippet
					l = colorizeLine(l)

					prettyLines = append(prettyLines, l)
				}

				continue
			}

			l = colorizeLine(l)
		}
		prettyLines = append(prettyLines, l)
	}
	return strings.Join(prettyLines, sep)
}

func colorizeLine(l string) string {
	if l == "" {
		return l
	}

	if rune(l[0]) == '┃' {
		l = l[1:]
	}

	switch l[0] {
	case '#':
		// skip empty comment lines
		if len(l) == 1 {
			l = "-"
			break
		}
		// colorize comments based on the number of comment markers, allowing control over the colorization
		// 	# First level colorized yellow
		//  ## Second level colorized purple
		//  ### Third level colorized cyan
		switch l[1] {
		// skip over shebangs (#!), don't colorize them
		case '!':
			break
		case '#':
			switch l[2] {
			case '#':
				l = "‣" + color.CyanString(highlightHyperlinks(l[3:]))
			default:
				l = "‣" + color.MagentaString(highlightHyperlinks(l[2:]))
			}
		default:
			l = "-" + color.YellowString(highlightHyperlinks(l[1:]))
		}
	}

	return l
}

func highlightHyperlinks(l string) string {
	// highlight hyperlinks
	re := regexp.MustCompile(`https?://.*`)
	if re.MatchString(l) {
		link := re.FindString(l)
		const escape = "\x1b"
		link = fmt.Sprintf("%s[%dm", escape, 4) + link + fmt.Sprintf("%s[%dm", escape, 24)
		l = re.ReplaceAllString(l, link)
	}
	return l
}

func doesFileExist(cheatfile string) bool {
	_, err := os.Stat(cheatfile)
	return err == nil
}

func cheatsheetNotFound(cmdname string) {
	fmt.Fprintf(os.Stderr, "No cheatsheet found for '%s'\n", cmdname)
	fmt.Fprintf(os.Stderr, "To create a new sheet, run: cheat -e %s\n", cmdname)
	os.Exit(1)
}
