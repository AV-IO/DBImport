# DBImport

## What's it do
The goal of this project is to be able to mass import many file formats into a unified database containing usernames, passwords, and hashes.

## Details

## Supported input formats
* Text files (default accepted separators are `:` and `,`)
	* user:pass:hash
	* user:hash
	* user:pass
	* hash
	* pass
* CSV

## How to use
* Requires a PostgreSQL database
```
-p	root path for recursive file import
-i	file with list of input files to read from
-r	remove unnecessary duplicates from database
-u	file to output all unparsed input files (Default STDOUT)
-dbu	local database user
-dbn	local database name
```
### Suggested usage:
* use `-p` to import files recursively from a dump (or collection)
* use `-u` to capture unread files. If changes are made later to make the files readable you can use `-i` to limit the number of checked files
* use `-r` unless you really want duplicate records. It may be slightly slower, but the other option is to slowly dedup the entire DB later.

## Planned improvements
* User provided RE2 regex for text file parsing
* Support for database formats
	* generic SQL
	* sqlite3
	* mysql
	* more?

Pull Requests are welcome!