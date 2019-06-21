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
	"regexp"
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
		_, err := db.Query(
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
		errCheck(err)
	} else {
		_, err := db.Query(`INSERT INTO $1(user, pass, hash, fpath) VALUES ('$2', '$3', '$4', '$5')`, tableName, input.User, input.Pass, input.Hash, file)
		errCheck(err)
	}
}

// dbSetup returns a
func dbSetup(dbuser, dbname string) (db *sql.DB) {
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

// printList prints a list to a file or to STDOUT if the file path is ""
func printList(file string, list []string) {
	if file == "" { // print to stdout
		fmt.Println("The following files were not read:")
		for _, item := range list {
			fmt.Println("\t" + item)
		}
	} else { // print to file
		f, err := os.Create(file)
		errCheck(err)
		defer f.Close()
		for _, item := range list {
			_, err = f.WriteString(item + "\n")
			errCheck(err)
		}
	}
}

func sharedPrefix(list []string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	}
	min, max := list[0], list[0]
	for _, f := range list[1:] {
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
}

// listFromFile ... Is this really that different than just slurping file contents into memory?
func listFromFile(file string) (list []string) {
	f, _ := os.Open(file)
	defer f.Close()
	r := bufio.NewReader(f)
	for {
		str, err := r.ReadString('\n')
		if err != io.EOF {
			errCheck(err)
		}
		str, _ = filepath.Abs(str)
		list = append(list, str)
		if err == io.EOF {
			break
		}
	}
	return
}

// handleFiles handles the processing of all file types
func handleFiles(input, unreadPath, userMatch string, isRootPath, rDups bool, db *sql.DB) {
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
		listFromFile(input)
	}
	// finding shared path
	sharedPath := sharedPrefix(files)
	// handle files
	var results chan ih.UPH
	for _, file := range files {
		switch filepath.Ext(file) {
		case "txt", "": // lazily assuming no file extension means plain text
			go ih.HandleTxt(file, userMatch, results)
		case "csv":
			go ih.HandleCsv(file, results)
		case "db", "sql", "mysql", "sqlite3":
			// Not quite able to handle database imports yet.
			fallthrough
		default: // don't know how to parse
			// includes "pdf", "svg", "doc", "docx", "rtf", "html", "log", "xml" and other unhandleable files
			// includes "zip", "7z", "gz", "xz", "tar", "rar" shouldn't accidentally fill up a harddrive with uncompressed zips
			ignoredFiles = append(ignoredFiles, file)
			continue
		}
		// handle received data until channel closes.
		for r, ok := <-results; ok; r, ok = <-results {
			// Concurrency should be an addition here... hopefully it does not become to bogged down on the DB side as queries start adding up.
			go addRecord(db, r, strings.Replace(file, sharedPath, "", 1), rDups)
		}
	}
	// list all files that were not read
	printList(unreadPath, ignoredFiles)
}

func flagCheck(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	// Flag handling
	help := flag.Bool("h", false, "show more help about a flag")

	rPath := flag.String("p", "", "root path for recursive file import")
	fList := flag.String("i", "", "file with list of input files to read from")
	unreadPath := flag.String("u", "", "file to output all unparsed input files (Default STDOUT)")

	userMatch := flag.String("m", "", "RE2 Regular Expression for text files (use -m and -h flags for requirements)")

	rDups := flag.Bool("r", false, "remove unnecessary duplicates from database")
	dbUser := flag.String("dbu", "", "local database user")
	dbName := flag.String("dbn", "default", "local database name")

	flag.Parse()
	mIsSet := flagCheck("m")

	// Checking if extra help is needed
	if *help {
		switch {
		case mIsSet:
			fmt.Println(
				"Regular expression is in the RE2 format and can be written in two forms:\n",
				"\tExtracting a precise format: 'user:pass:hash'\n",
				"\tExtracting individual groups: '(user)|(pass)|(hash)'\n",
				"Both methods require that named capture groups are used. These names must be the following:\n",
				"\tusername/email: (?P<username>)\n",
				"\tpassword: (?P<password>)\n",
				"\thash: (?P<hash>)",
			)
		default:
			flag.PrintDefaults()
		}
		return
	}
	canRun := true
	var input string // either rPath or fList
	if *rPath != "" {
		input = *rPath
	} else {
		if *fList != "" {
			input = *fList
		} else {
			fmt.Println("Please specify a path")
			flag.PrintDefaults()
			canRun = false
		}
	}
	if *dbUser == "" {
		fmt.Println("Please enter a database Username")
		flag.PrintDefaults()
		canRun = false
	}
	if _, err := regexp.Compile(*userMatch); mIsSet && err != nil {
		fmt.Println("Regular Expression cannot be compiled. Please ensure correct syntax is being used.")
		canRun = false
	}
	if !canRun {
		return
	}

	db := dbSetup(*dbUser, *dbName)
	// process files
	handleFiles(input, *unreadPath, *userMatch, true, *rDups, db)
}
