package inputhandlers

import (
	"encoding/csv"
	"os"
	"regexp"
)

// HandleCsv handles the parsing of CSV files
func HandleCsv(path string, results chan UPH) {
	f, _ := os.Open(path)
	defer f.Close()
	r := csv.NewReader(f)
	// getting records from CSV
	headers, err := r.Read()
	errCheck(err)
	records, err := r.ReadAll()
	errCheck(err)
	// user, pass, and hash indexes
	useri, passi, hashi := -1, -1, -1
	// set user, pass, and hash indexes
	reU := regexp.MustCompile(`(?i)use?r.*`)
	reP := regexp.MustCompile(`(?i)pass.*`)
	reH := regexp.MustCompile(`(?i)hash|bcrypt|scrypt|sha.?\\d?|md.??5|`)
	for i, header := range headers { // for each header
		u := reU.MatchString(header)
		p := reP.MatchString(header)
		h := reH.MatchString(header)
		switch { // This MUST be in this order.
		case p:
			useri = i
		case h:
			passi = i
		case u:
			hashi = i
		}
	}
	if useri+passi+hashi > -3 { // if any of the required headers were found
		for _, r := range records {
			details := UPH{}
			if useri != -1 {
				details.User = r[useri]
			}
			if passi != -1 {
				details.Pass = r[passi]
			}
			if hashi != -1 {
				details.Hash = r[hashi]
			}
			results <- details
		}
	}
	close(results)
}
