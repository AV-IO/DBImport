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
func separate(line, separator string, expected int) (list []string, err error) {
	re := regexp.MustCompile("(.*)" + separator) // So Go doesn't support negative lookbehinds :/
	var ilist []int
	// checking if separator is escaped
	for _, index := range re.FindAllStringIndex(line, -1) {
		count := 0
		for ; line[index[0]-1-count] == '\\'; count++ {
		}
		if count%2 == 0 {
			ilist = append(ilist, index[0])
		}
	}
	// setting list based on indexes supplied from ilist
	if len(ilist) > 0 {
		list = append(list, line[:ilist[0]])
		for i := 0; i < len(ilist)-1; i++ {
			list = append(list, line[ilist[i]:ilist[i+1]])
		}
	}
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

func namedMatchtoMap(re *regexp.Regexp, line string) (matchMap map[string]string) {
	matchMap = make(map[string]string)
	matches := re.FindAllStringSubmatch(line, -1)
	for i := range matches {
		for j, name := range re.SubexpNames() {
			if j != 0 && name != "" && matches[i][j] != "" && matchMap[name] == "" {
				matchMap[name] = matches[i][j]
			}
		}
	}
	return matchMap
}

func checkUserMatch(userMatch string, lines []string, userPG parseGroup) bool {
	for _, l := range lines {
		if len(namedMatchtoMap(userPG.re, l)) > 0 {
			userPG.count++
		}
	}
	return userPG.count == len(lines)-1
}

func checkStaticMatches(staticMatches []parseGroup, lines []string) (index int, separator string) {
	for i, pg := range staticMatches { // for each match
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
			return i, tmpSep
		}
	}
	return -1, ""
}

// HandleTxt handles formats of plain text files not under a standard (e.g. csv)
// supported formats are as follows:
// user:pass:hash, user:hash, user:pass, hash, pass
func HandleTxt(path, userMatch string, results chan UPH) {
	// open file
	f, _ := os.Open(path)
	defer f.Close()
	// ingest lines
	r := bufio.NewReader(f)
	lines := make([]string, 10)
	for i := 0; i < len(lines); i++ { // 10 seems like a decent sample
		str, err := r.ReadString('\n')
		errCheck(err)
		lines[i] = str
	}
	_, err := f.Seek(0, 0) // reset read offset
	errCheck(err)
	// check for a pattern
	// assuming all strings > 32 are hashes (hopefully not long pass)

	staticMatches := []parseGroup{
		// user:pass:hash
		parseGroup{
			re: regexp.MustCompile(`\w+[:,]\w+[:,]\w{32,}`),
			handler: func(line, separator string) UPH {
				list, err := separate(line, separator, 3)
				errCheck(err)
				return UPH{User: list[0], Pass: list[1], Hash: list[2]}
			},
		},
		// user:hash (hopefully not pass:hash)
		parseGroup{
			re: regexp.MustCompile(`\w+([:\,])\w{32,}`),
			handler: func(line, separator string) UPH {
				list, err := separate(line, separator, 2)
				errCheck(err)
				return UPH{User: list[0], Hash: list[1]}
			},
		},
		// user:pass
		parseGroup{
			re: regexp.MustCompile(`\w+([:\,])\w+`),
			handler: func(line, separator string) UPH {
				list, err := separate(line, separator, 2)
				errCheck(err)
				return UPH{User: list[0], Pass: list[1]}
			},
		},
		// hash
		parseGroup{
			re: regexp.MustCompile(`\w{32,}`),
			handler: func(line, separator string) UPH {
				return UPH{Hash: line}
			},
		},
		// pass
		parseGroup{
			re: regexp.MustCompile(`\w+`),
			handler: func(line, separator string) UPH {
				return UPH{Pass: line}
			},
		},
	}

	// set handler for matched regex
	var parser func(string, string) UPH
	var separator string
	// prioritize matching user supplied regex
	if userMatch != "" {
		userRE, err := regexp.Compile(userMatch)
		errCheck(err)
		userPG := parseGroup{
			re: userRE,
			handler: func(line, separator string) UPH {
				matchMap := namedMatchtoMap(userRE, line)
				return UPH{User: matchMap["username"], Pass: matchMap["password"], Hash: matchMap["hash"]}
			},
		}
		if checkUserMatch(userMatch, lines, userPG) {
			parser = userPG.handler
		}
	}
	if parser == nil { // if user match was not found
		var index int
		index, separator = checkStaticMatches(staticMatches, lines)
		if index != -1 {
			parser = staticMatches[index].handler
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
