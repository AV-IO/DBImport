package main

import (
	ih "./inputhandlers"
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/lib/pq"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const tableName = "default"

func errCheck(err error) {
	if err != nil {
		panic(err)
	}
}

// addRecord ads a ih.UPH & file to the database
// rDups dictates weather to remove duplicates on insert.
func addRecord(db *sql.DB, input ih.UPH, file string, rDups bool) {
	if rDups {
		// ToDo: double check that this isn't just PURE garbage.
		db.Query(
			`INSERT INTO $1(username, pass, hash, fpath)
	SELECT '$2', '$3', '$4', '$5'
WHERE NOT EXISTS (
	SELECT 1 FROM $1 WHERE
	(username='$2' AND pass='$3' AND hash='$4') OR
	(username='$2' AND pass='$3' AND '$4'=''  ) OR
	(username='$2' AND '$3'=''   AND hash='$4') OR
	(    '$2'=''   AND pass='$3' AND '$4'=''  ) OR
	(    '$2'=''   AND '$3'=''   AND hash='$4')
);`,
			tableName, input.User, input.Pass, input.Hash, file,
		)
	} else {
		db.Query(`INSERT INTO $1(user, pass, hash, fpath) VALUES ('$2', '$3', '$4', '$5')`, tableName, input.User, input.Pass, input.Hash, file)
	}
}

// dbSetup returns a
func dbSetup(dbuser string, dbname string) (db *sql.DB) {
	// Get DB connection
	db, err := sql.Open("postgres", "user="+dbuser+" dbname="+dbname+" sslmode=disable")
	errCheck(err)
	// creating table and populating columns
	_, err = db.Query(`CREATE TABLE IF NOT EXISTS $1 (
	id SERIAL PRIMARY KEY,
	username TEXT,
	pass TEXT,
	hash TEXT,
	fpath TEXT
);`, tableName)
	errCheck(err)
	// returning a set up DB
	return
}

// handleFiles handles the processing of all file types
func handleFiles(input string, isRootPath bool, unreadPath string, rDups bool, db *sql.DB) {
	var files, ignoredFiles []string
	if isRootPath {
		err := filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			path, _ = filepath.Abs(path)
			files = append(files, path)
			return nil
		})
		errCheck(err)
	} else { // if input is file with list of files to read from
		f, _ := os.Open(input)
		r := bufio.NewReader(f)
		for {
			str, err := r.ReadString('\n')
			if err != io.EOF {
				errCheck(err)
			}
			str, _ = filepath.Abs(str)
			files = append(files, str)
			if err == io.EOF {
				break
			}
		}
		f.Close()
	}
	// finding shared path
	sharedPath := func(fl []string) string {
		switch len(fl) {
		case 0:
			return ""
		case 1:
			return fl[0]
		}
		min, max := fl[0], fl[0]
		for _, f := range fl[1:] {
			switch {
			case f < min:
				min = f
			case f > max:
				max = f
			}
		}
		for i := 0; i < len(min) && i < len(max); i++ {
			if min[i] != max[i] {
				return min[:i]
			}
		}
		return min
	}(files)
	// handle files
	var results chan ih.UPH
	for _, file := range files {
		switch filepath.Ext(file) {
		case "txt", "": // lazily assuming no file extension means plain text
			go ih.HandleTxt(file, results)
		case "csv":
			go ih.HandleCsv(file, results)
		case "db", "sql", "mysql", "sqlite3":
			// go ih.HandleDB(file, results)
			fallthrough // Not quite able to handle database imports yet.
		case "pdf", "svg", "doc", "docx", "rtf", "html", "log", "xml": // unhandleable files
			fallthrough
		case "zip", "7z", "gz", "xz", "tar", "rar": // not going to accidentally fill up a harddrive with uncompressed zips. I'll let the user do that.
			fallthrough
		default: // don't know how to parse
			ignoredFiles = append(ignoredFiles, file)
			continue
		}
		// handle received data until channel closes.
		for {
			r, ok := <-results
			if ok { // if more records to come
				// Concurrency should be an addition here... hopefully it does not become to bogged down on the DB side as queries start adding up.
				go addRecord(db, r, strings.Replace(file, sharedPath, "", 1), rDups)
			} else { // channel closed
				break
			}
		}
	}
	// list all files that were not read
	if unreadPath == "" { // print to stdout
		fmt.Println("The following files were not read:")
		for _, file := range ignoredFiles {
			fmt.Println("\t" + file)
		}
	} else { // print to file
		f, err := os.Create(unreadPath)
		errCheck(err)
		defer f.Close()
		for _, file := range ignoredFiles {
			f.WriteString(file + "\n")
		}
	}
}

func main() {
	// Flag handling
	rPath := flag.String("p", "", "root path for file import")
	fList := flag.String("i", "", "file with list of input files to read from")
	rDups := flag.Bool("r", false, "remove unnecessary duplicates from database")
	unreadPath := flag.String("u", "", "file to output all unparsed input files (Default STDOUT)")
	dbUser := flag.String("dbu", "", "local database user")
	dbName := flag.String("dbn", "", "local database name")
	if *rPath == "" && *fList == "" {
		fmt.Println("Please specify a path")
		flag.PrintDefaults()
		return
	}
	if *dbName == "" {
		*dbName = "default"
	}
	if *dbUser == "" || *dbName == "" {
		fmt.Println("Please enter all requrired database information")
		flag.PrintDefaults()
		return
	}
	db := dbSetup(*dbUser, *dbName)
	// process files
	if *rPath == "" {
		handleFiles(*rPath, true, *unreadPath, *rDups, db)
	} else {
		handleFiles(*fList, false, *unreadPath, *rDups, db)
	}
}
