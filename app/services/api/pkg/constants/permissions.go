package constants

// ─── Permission Actions ───
const (
	ActionView            = "VIEW"
	ActionCreate          = "CREATE"
	ActionEdit            = "EDIT"
	ActionDelete          = "DELETE"
	ActionApproveL1       = "APPROVE_L1"       // Desk Head
	ActionApproveL2       = "APPROVE_L2"       // Center Director / Division Head
	ActionApproveRiskL1   = "APPROVE_RISK_L1"  // Risk Officer
	ActionApproveRiskL2   = "APPROVE_RISK_L2"  // Risk Head
	ActionBookL1          = "BOOK_L1"          // Accountant
	ActionBookL2          = "BOOK_L2"          // Chief Accountant
	ActionSettle          = "SETTLE"           // Settlement Officer
	ActionRecall          = "RECALL"
	ActionCancelRequest   = "CANCEL_REQUEST"
	ActionCancelApproveL1 = "CANCEL_APPROVE_L1" // Desk Head approves cancel
	ActionCancelApproveL2 = "CANCEL_APPROVE_L2" // Division Head approves cancel
	ActionClone           = "CLONE"
	ActionExport          = "EXPORT"
	ActionManage          = "MANAGE"
)

// ─── Resources ───
const (
	ResourceFxDeal               = "FX_DEAL"
	ResourceBondDeal             = "BOND_DEAL"
	ResourceMMInterbankDeal      = "MM_INTERBANK_DEAL"
	ResourceMMOMORepoDeal        = "MM_OMO_REPO_DEAL"
	ResourceCreditLimit          = "CREDIT_LIMIT"
	ResourceInternationalPayment = "INTERNATIONAL_PAYMENT"
	ResourceMasterData           = "MASTER_DATA"
	ResourceAuditLog             = "AUDIT_LOG"
	ResourceSystem               = "SYSTEM"
)

// Perm builds a permission string: "RESOURCE.ACTION"
func Perm(resource, action string) string {
	return resource + "." + action
}

// ─── FX Deal Permissions ───
var (
	PermFxView            = Perm(ResourceFxDeal, ActionView)
	PermFxCreate          = Perm(ResourceFxDeal, ActionCreate)
	PermFxEdit            = Perm(ResourceFxDeal, ActionEdit)
	PermFxDelete          = Perm(ResourceFxDeal, ActionDelete)
	PermFxApproveL1       = Perm(ResourceFxDeal, ActionApproveL1)
	PermFxApproveL2       = Perm(ResourceFxDeal, ActionApproveL2)
	PermFxBookL1          = Perm(ResourceFxDeal, ActionBookL1)
	PermFxBookL2          = Perm(ResourceFxDeal, ActionBookL2)
	PermFxSettle          = Perm(ResourceFxDeal, ActionSettle)
	PermFxRecall          = Perm(ResourceFxDeal, ActionRecall)
	PermFxCancelRequest   = Perm(ResourceFxDeal, ActionCancelRequest)
	PermFxCancelApproveL1 = Perm(ResourceFxDeal, ActionCancelApproveL1)
	PermFxCancelApproveL2 = Perm(ResourceFxDeal, ActionCancelApproveL2)
	PermFxClone           = Perm(ResourceFxDeal, ActionClone)
	PermFxExport          = Perm(ResourceFxDeal, ActionExport)
)

// ─── Bond Deal Permissions ───
var (
	PermBondView            = Perm(ResourceBondDeal, ActionView)
	PermBondCreate          = Perm(ResourceBondDeal, ActionCreate)
	PermBondEdit            = Perm(ResourceBondDeal, ActionEdit)
	PermBondDelete          = Perm(ResourceBondDeal, ActionDelete)
	PermBondApproveL1       = Perm(ResourceBondDeal, ActionApproveL1)
	PermBondApproveL2       = Perm(ResourceBondDeal, ActionApproveL2)
	PermBondBookL1          = Perm(ResourceBondDeal, ActionBookL1)
	PermBondBookL2          = Perm(ResourceBondDeal, ActionBookL2)
	PermBondRecall          = Perm(ResourceBondDeal, ActionRecall)
	PermBondCancelRequest   = Perm(ResourceBondDeal, ActionCancelRequest)
	PermBondCancelApproveL1 = Perm(ResourceBondDeal, ActionCancelApproveL1)
	PermBondCancelApproveL2 = Perm(ResourceBondDeal, ActionCancelApproveL2)
	PermBondClone           = Perm(ResourceBondDeal, ActionClone)
	PermBondExport          = Perm(ResourceBondDeal, ActionExport)
)

// ─── MM Interbank Deal Permissions ───
var (
	PermMMInterbankView            = Perm(ResourceMMInterbankDeal, ActionView)
	PermMMInterbankCreate          = Perm(ResourceMMInterbankDeal, ActionCreate)
	PermMMInterbankEdit            = Perm(ResourceMMInterbankDeal, ActionEdit)
	PermMMInterbankDelete          = Perm(ResourceMMInterbankDeal, ActionDelete)
	PermMMInterbankApproveL1       = Perm(ResourceMMInterbankDeal, ActionApproveL1)
	PermMMInterbankApproveL2       = Perm(ResourceMMInterbankDeal, ActionApproveL2)
	PermMMInterbankApproveRiskL1   = Perm(ResourceMMInterbankDeal, ActionApproveRiskL1)
	PermMMInterbankApproveRiskL2   = Perm(ResourceMMInterbankDeal, ActionApproveRiskL2)
	PermMMInterbankBookL1          = Perm(ResourceMMInterbankDeal, ActionBookL1)
	PermMMInterbankBookL2          = Perm(ResourceMMInterbankDeal, ActionBookL2)
	PermMMInterbankSettle          = Perm(ResourceMMInterbankDeal, ActionSettle)
	PermMMInterbankRecall          = Perm(ResourceMMInterbankDeal, ActionRecall)
	PermMMInterbankCancelRequest   = Perm(ResourceMMInterbankDeal, ActionCancelRequest)
	PermMMInterbankCancelApproveL1 = Perm(ResourceMMInterbankDeal, ActionCancelApproveL1)
	PermMMInterbankCancelApproveL2 = Perm(ResourceMMInterbankDeal, ActionCancelApproveL2)
	PermMMInterbankClone           = Perm(ResourceMMInterbankDeal, ActionClone)
	PermMMInterbankExport          = Perm(ResourceMMInterbankDeal, ActionExport)
)

// ─── MM OMO/Repo Deal Permissions ───
var (
	PermMMOMORepoView            = Perm(ResourceMMOMORepoDeal, ActionView)
	PermMMOMORepoCreate          = Perm(ResourceMMOMORepoDeal, ActionCreate)
	PermMMOMORepoEdit            = Perm(ResourceMMOMORepoDeal, ActionEdit)
	PermMMOMORepoDelete          = Perm(ResourceMMOMORepoDeal, ActionDelete)
	PermMMOMORepoApproveL1       = Perm(ResourceMMOMORepoDeal, ActionApproveL1)
	PermMMOMORepoApproveL2       = Perm(ResourceMMOMORepoDeal, ActionApproveL2)
	PermMMOMORepoBookL1          = Perm(ResourceMMOMORepoDeal, ActionBookL1)
	PermMMOMORepoBookL2          = Perm(ResourceMMOMORepoDeal, ActionBookL2)
	PermMMOMORepoRecall          = Perm(ResourceMMOMORepoDeal, ActionRecall)
	PermMMOMORepoCancelRequest   = Perm(ResourceMMOMORepoDeal, ActionCancelRequest)
	PermMMOMORepoCancelApproveL1 = Perm(ResourceMMOMORepoDeal, ActionCancelApproveL1)
	PermMMOMORepoCancelApproveL2 = Perm(ResourceMMOMORepoDeal, ActionCancelApproveL2)
	PermMMOMORepoClone           = Perm(ResourceMMOMORepoDeal, ActionClone)
)

// ─── Credit Limit Permissions ───
var (
	PermCreditLimitView          = Perm(ResourceCreditLimit, ActionView)
	PermCreditLimitCreate        = Perm(ResourceCreditLimit, ActionCreate)
	PermCreditLimitApproveL1     = Perm(ResourceCreditLimit, ActionApproveL1)
	PermCreditLimitApproveRiskL1 = Perm(ResourceCreditLimit, ActionApproveRiskL1)
	PermCreditLimitApproveRiskL2 = Perm(ResourceCreditLimit, ActionApproveRiskL2)
)

// ─── International Payment Permissions ───
var (
	PermIntlPaymentView   = Perm(ResourceInternationalPayment, ActionView)
	PermIntlPaymentCreate = Perm(ResourceInternationalPayment, ActionCreate)
	PermIntlPaymentSettle = Perm(ResourceInternationalPayment, ActionSettle)
)

// ─── Master Data Permissions ───
var (
	PermMasterDataView   = Perm(ResourceMasterData, ActionView)
	PermMasterDataManage = Perm(ResourceMasterData, ActionManage)
)

// ─── Audit Log Permissions ───
var (
	PermAuditLogView = Perm(ResourceAuditLog, ActionView)
)

// ─── System Permissions ───
var (
	PermSystemManage = Perm(ResourceSystem, ActionManage)
)

// RolePermissions maps each role to its set of permissions.
// This is the SINGLE SOURCE OF TRUTH for role-permission mapping.
var RolePermissions = map[string][]string{
	RoleDealer: {
		// FX
		PermFxView, PermFxCreate, PermFxEdit, PermFxDelete,
		PermFxRecall, PermFxCancelRequest, PermFxClone, PermFxExport,
		// Bond
		PermBondView, PermBondCreate, PermBondEdit, PermBondDelete,
		PermBondRecall, PermBondCancelRequest, PermBondClone, PermBondExport,
		// MM Interbank
		PermMMInterbankView, PermMMInterbankCreate, PermMMInterbankEdit, PermMMInterbankDelete,
		PermMMInterbankRecall, PermMMInterbankCancelRequest, PermMMInterbankClone, PermMMInterbankExport,
		// MM OMO/Repo
		PermMMOMORepoView, PermMMOMORepoCreate, PermMMOMORepoEdit, PermMMOMORepoDelete,
		PermMMOMORepoRecall, PermMMOMORepoCancelRequest, PermMMOMORepoClone,
	},
	RoleDeskHead: {
		// FX
		PermFxView, PermFxApproveL1, PermFxRecall, PermFxCancelApproveL1, PermFxExport,
		// Bond
		PermBondView, PermBondApproveL1, PermBondRecall, PermBondCancelApproveL1, PermBondExport,
		// MM Interbank
		PermMMInterbankView, PermMMInterbankApproveL1, PermMMInterbankRecall, PermMMInterbankCancelApproveL1,
		// MM OMO/Repo
		PermMMOMORepoView, PermMMOMORepoApproveL1, PermMMOMORepoRecall, PermMMOMORepoCancelApproveL1,
		// Limit
		PermCreditLimitView,
	},
	RoleCenterDirector: {
		// FX
		PermFxView, PermFxApproveL2, PermFxCancelApproveL2, PermFxExport,
		// Bond
		PermBondView, PermBondApproveL2, PermBondCancelApproveL2, PermBondExport,
		// MM Interbank
		PermMMInterbankView, PermMMInterbankApproveL2, PermMMInterbankCancelApproveL2,
		// MM OMO/Repo
		PermMMOMORepoView, PermMMOMORepoApproveL2, PermMMOMORepoCancelApproveL2,
		// Limit
		PermCreditLimitView, PermCreditLimitApproveL1,
	},
	RoleDivisionHead: {
		// FX
		PermFxView, PermFxApproveL2, PermFxCancelApproveL2, PermFxExport,
		// Bond
		PermBondView, PermBondApproveL2, PermBondCancelApproveL2, PermBondExport,
		// MM Interbank
		PermMMInterbankView, PermMMInterbankApproveL2, PermMMInterbankCancelApproveL2,
		// MM OMO/Repo
		PermMMOMORepoView, PermMMOMORepoApproveL2, PermMMOMORepoCancelApproveL2,
		// Limit
		PermCreditLimitView, PermCreditLimitCreate, PermCreditLimitApproveL1,
	},
	RoleRiskOfficer: {
		// NO FX permissions! Risk only sees MM Interbank for limit/risk approval
		PermMMInterbankView, PermMMInterbankApproveRiskL1,
		PermCreditLimitView, PermCreditLimitApproveRiskL1,
	},
	RoleRiskHead: {
		PermMMInterbankView, PermMMInterbankApproveRiskL2,
		PermCreditLimitView, PermCreditLimitApproveRiskL2,
	},
	RoleAccountant: {
		PermFxView, PermFxBookL1,
		PermBondView, PermBondBookL1,
		PermMMInterbankView, PermMMInterbankBookL1,
		PermMMOMORepoView, PermMMOMORepoBookL1,
	},
	RoleChiefAccountant: {
		PermFxView, PermFxBookL2,
		PermBondView, PermBondBookL2,
		PermMMInterbankView, PermMMInterbankBookL2,
		PermMMOMORepoView, PermMMOMORepoBookL2,
	},
	RoleSettlementOfficer: {
		PermFxView, PermFxSettle,
		PermMMInterbankView, PermMMInterbankSettle,
		PermIntlPaymentView, PermIntlPaymentSettle,
	},
	RoleAdmin: {
		PermFxView, PermFxExport, PermBondView, PermBondExport,
		PermMMInterbankView, PermMMInterbankExport, PermMMOMORepoView,
		PermCreditLimitView, PermIntlPaymentView,
		PermMasterDataView, PermMasterDataManage,
		PermAuditLogView,
		PermSystemManage,
	},
}
