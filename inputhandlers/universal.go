package inputhandlers

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
