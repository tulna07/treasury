package constants

// Role codes matching DB values.
const (
	RoleDealer            = "DEALER"
	RoleDeskHead          = "DESK_HEAD"
	RoleCenterDirector    = "CENTER_DIRECTOR"
	RoleDivisionHead      = "DIVISION_HEAD"
	RoleRiskOfficer       = "RISK_OFFICER"
	RoleRiskHead          = "RISK_HEAD"
	RoleAccountant        = "ACCOUNTANT"
	RoleChiefAccountant   = "CHIEF_ACCOUNTANT"
	RoleSettlementOfficer = "SETTLEMENT_OFFICER"
	RoleAdmin             = "ADMIN"
)

// AllRoles lists all valid roles.
var AllRoles = []string{
	RoleDealer, RoleDeskHead, RoleCenterDirector, RoleDivisionHead,
	RoleRiskOfficer, RoleRiskHead, RoleAccountant, RoleChiefAccountant,
	RoleSettlementOfficer, RoleAdmin,
}
