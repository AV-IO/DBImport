package inputhandlers

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
)

type parseGroup struct {
	re      *regexp.Regexp
	count   int
	handler func(string, string) UPH
}

// separate separates a string into a list based on the supplied separator.
// an error is returned if the expected list length is greater or less than the resulting list.
// set expected to -1 to accept any length of list
func separate(line string, separator string, expected int) (list []string, err error) {
	//re := regexp.MustCompile("(.*)(?<!(?<!\\)\\)" + separator) // ignores escaped separators and checks for a previously escaped '\'. (does not check for more than 1)
	re := regexp.MustCompile("(.*)" + separator) // So Go doesn't support negative lookbehinds :/
	list = re.FindAllString(line, -1)
	if expected == -1 || len(list) == expected {
		return
	}
	if len(list) > expected {
		err = errors.New("List length more than expected")
	} else if len(list) < expected {
		err = errors.New("List length less than expected")
	}
	return
}

// HandleTxt handles formats of plain text files not under a standard (e.g. csv)
// supported formats are as follows:
// user:pass:hash, user:hash, user:pass, hash, pass
func HandleTxt(path string, results chan UPH) {
	// open file
	f, _ := os.Open(path)
	defer f.Close()
	// ingest lines
	r := bufio.NewReader(f)
	var lines [10]string
	for i := 0; i < len(lines); i++ { // 10 seems like a decent sample
		str, err := r.ReadString('\n')
		errCheck(err)
		lines[i] = str
	}
	f.Seek(0, 0) // reset read offset
	// check for a pattern
	// assuming all strings > 32 are hashes (hopefully not long pass)

	reMatch := []parseGroup{
		// user:pass:hash
		parseGroup{
			re: regexp.MustCompile("\\w+([:\\,])\\w+\\1\\w{32,}"),
			handler: func(line string, separator string) UPH {
				list, err := separate(line, separator, 3)
				errCheck(err)
				return UPH{User: list[0], Pass: list[1], Hash: list[2]}
			},
		},
		// user:hash (hopefully not pass:hash)
		parseGroup{
			re: regexp.MustCompile("\\w+([:\\,])\\w{32,}"),
			handler: func(line string, separator string) UPH {
				list, err := separate(line, separator, 2)
				errCheck(err)
				return UPH{User: list[0], Hash: list[1]}
			},
		},
		// user:pass
		parseGroup{
			re: regexp.MustCompile("\\w+([:\\,])\\w+"),
			handler: func(line string, separator string) UPH {
				list, err := separate(line, separator, 2)
				errCheck(err)
				return UPH{User: list[0], Pass: list[1]}
			},
		},
		// hash
		parseGroup{
			re: regexp.MustCompile("\\w{32,}"),
			handler: func(line string, separator string) UPH {
				return UPH{Hash: line}
			},
		},
		// pass
		parseGroup{
			re: regexp.MustCompile("\\w+"),
			handler: func(line string, separator string) UPH {
				return UPH{Pass: line}
			},
		},
	}
	// set handler for matched regex
	var parser func(string, string) UPH
	var separator string
	for _, pg := range reMatch { // for each match
		tmpSep := ""
		for _, l := range lines { // for each line
			found := pg.re.FindString(l)
			if found != "" {
				pg.count++
				tmpSep = found // there is a risk that if the separator changes through the file, this could change.
			}
		}
		// in hierarchy order, if all lines matched call handler
		if pg.count == len(lines)-1 {
			parser = pg.handler
			separator = tmpSep
			break
		}
	}
	// process file
	for {
		line, err := r.ReadString('\n')
		if err != io.EOF {
			errCheck(err)
		}
		// adding a record from the selected parser
		results <- parser(line, separator)
		if err == io.EOF { // EOF can be returned with a valid last line, so checking here
			break
		}
	}
	close(results)
}
