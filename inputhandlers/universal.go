package inputhandlers

import (
	"regexp"
)

// UPH is a struct defining a username, a password, and a hash.
// This is used for a uniform format between functions
type UPH struct {
	User string
	Pass string
	Hash string
}

// lazy error checking provided internally to package
func errCheck(err error) {
	if err != nil {
		panic(err)
	}
}

func namedMatchtoStringMap(re *regexp.Regexp, line string) (matchMap map[string]string) {
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

func namedMatchtoIndexMap(re *regexp.Regexp, list []string) (matchMap map[string]int) {
	matchMap = map[string]int{"username": -1, "password": -1, "hash": -1}
	for i, s := range list {
		match := re.FindStringSubmatch(s)
		for j, name := range re.SubexpNames() {
			if j >= len(match) {
				break
			}
			if j != 0 && name != "" && match[j] != "" && matchMap[name] == -1 {
				matchMap[name] = i
			}
		}
	}
	return matchMap
}

func standardMatchtoIndexMap(list []string) (matchMap map[string]int) {
	re := regexp.MustCompile(`(?i)(?P<username>use?r.*)|(?P<password>pass.*)|(?P<hash>hash|bcrypt|scrypt|sha.?\\d*|md.?5)`)
	return namedMatchtoIndexMap(re, list)
}
