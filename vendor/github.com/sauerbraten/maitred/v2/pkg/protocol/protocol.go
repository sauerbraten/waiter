package protocol

// standard master server protocol constants
// exported constants can be sent to the master server
const (
	RegServ = "regserv"
	SuccReg = "succreg"
	FailReg = "failreg"

	AddBan    = "addgban"
	ClearBans = "cleargbans"

	ReqAuth  = "reqauth"
	ChalAuth = "chalauth"
	ConfAuth = "confauth"
	SuccAuth = "succauth"
	FailAuth = "failauth"
)

// non-standard stats command
const (
	Stats     = "stats"
	SuccStats = "succstats"
	FailStats = "failstats"
)

// non-standard administration commands
const (
	ReqAdmin  = "reqadmin"
	ChalAdmin = "chaladmin"
	ConfAdmin = "confadmin"
	SuccAdmin = "succadmin"
	FailAdmin = "failadmin"

	AddAuth     = "addauth"
	SuccAddAuth = "succaddauth"
	FailAddAuth = "failaddauth"
	DelAuth     = "delauth"
	SuccDelAuth = "succdelauth"
	FailDelAuth = "faildelauth"
)

// other non-standard commands
const (
	Lookup     = "lookup" // could be used for name protection
	SuccLookup = "succlookup"
	FailLookup = "faillookup"
)
