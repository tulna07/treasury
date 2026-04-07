package constants

// Module names for the Treasury system.
const (
	ModuleFX          = "FX"
	ModuleBond        = "BOND"
	ModuleMMInterbank = "MM_INTERBANK"
	ModuleMMOMORepo   = "MM_OMO_REPO"
	ModuleLimit       = "LIMIT"
	ModuleSettlement  = "SETTLEMENT"
	ModuleRisk        = "RISK"
	ModuleAccounting  = "ACCOUNTING"
	ModuleAdmin       = "ADMIN"
)

// AllModules lists all valid module names.
var AllModules = []string{
	ModuleFX, ModuleBond, ModuleMMInterbank, ModuleMMOMORepo, ModuleLimit,
	ModuleSettlement, ModuleRisk, ModuleAccounting, ModuleAdmin,
}
