package inputhandlers

import (
	"encoding/csv"
	"os"
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
	// set user, pass, and hash indexes
	im := standardMatchtoIndexMap(headers)
	doContinue := false
	for _, v := range im {
		if v != -1 {
			doContinue = true
			break
		}
	}
	if doContinue { // if any of the required headers were found
		for _, r := range records {
			details := UPH{}
			if im["username"] != -1 {
				details.User = r[im["username"]]
			}
			if im["password"] != -1 {
				details.Pass = r[im["password"]]
			}
			if im["hash"] != -1 {
				details.Hash = r[im["hash"]]
			}
			results <- details
		}
	}
	close(results)
}
