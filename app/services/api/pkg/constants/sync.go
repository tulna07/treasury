package constants

import "strings"

// PermissionDef describes a permission for DB sync.
type PermissionDef struct {
	Code        string
	Resource    string
	Action      string
	Description string
}

// AllPermissionDefs returns all permissions defined in Go constants.
// This is the single source of truth — DB is synced from this on startup.
func AllPermissionDefs() []PermissionDef {
	codes := []string{
		// FX
		PermFxView, PermFxCreate, PermFxEdit, PermFxDelete,
		PermFxApproveL1, PermFxApproveL2,
		PermFxBookL1, PermFxBookL2, PermFxSettle,
		PermFxRecall, PermFxCancelRequest, PermFxCancelApproveL1, PermFxCancelApproveL2,
		PermFxClone, PermFxExport,
		// Bond
		PermBondView, PermBondCreate, PermBondEdit, PermBondDelete,
		PermBondApproveL1, PermBondApproveL2,
		PermBondBookL1, PermBondBookL2,
		PermBondRecall, PermBondCancelRequest, PermBondCancelApproveL1, PermBondCancelApproveL2,
		PermBondClone, PermBondExport,
		// MM Interbank
		PermMMInterbankView, PermMMInterbankCreate, PermMMInterbankEdit,
		PermMMInterbankApproveL1,
		PermMMInterbankApproveRiskL1, PermMMInterbankApproveRiskL2,
		PermMMInterbankBookL1, PermMMInterbankBookL2,
		// MM OMO/Repo
		PermMMOMORepoView, PermMMOMORepoBookL1, PermMMOMORepoBookL2,
		// Credit Limit
		PermCreditLimitView, PermCreditLimitCreate, PermCreditLimitApproveL1,
		PermCreditLimitApproveRiskL1, PermCreditLimitApproveRiskL2,
		// International Payment
		PermIntlPaymentView, PermIntlPaymentCreate, PermIntlPaymentSettle,
		// Master Data
		PermMasterDataView, PermMasterDataManage,
		// Audit Log
		PermAuditLogView,
		// System
		PermSystemManage,
	}

	defs := make([]PermissionDef, 0, len(codes))
	for _, code := range codes {
		parts := strings.SplitN(code, ".", 2)
		resource, action := parts[0], ""
		if len(parts) == 2 {
			action = parts[1]
		}
		defs = append(defs, PermissionDef{
			Code:     code,
			Resource: resource,
			Action:   action,
		})
	}
	return defs
}
