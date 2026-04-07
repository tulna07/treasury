package security

import "github.com/kienlongbank/treasury-api/pkg/constants"

// RBACChecker provides role-based access control checks.
type RBACChecker struct {
	permissionsByRole map[string]map[string]bool
}

// NewRBACChecker creates a new RBAC checker loaded from the constants package.
func NewRBACChecker() *RBACChecker {
	checker := &RBACChecker{
		permissionsByRole: make(map[string]map[string]bool),
	}
	for role, perms := range constants.RolePermissions {
		permSet := make(map[string]bool, len(perms))
		for _, p := range perms {
			permSet[p] = true
		}
		checker.permissionsByRole[role] = permSet
	}
	return checker
}

// Reload refreshes the in-memory permission map from the constants package.
func (r *RBACChecker) Reload() {
	newMap := make(map[string]map[string]bool)
	for role, perms := range constants.RolePermissions {
		permSet := make(map[string]bool, len(perms))
		for _, p := range perms {
			permSet[p] = true
		}
		newMap[role] = permSet
	}
	r.permissionsByRole = newMap
}

// HasPermission checks if a single role has the given permission.
func (r *RBACChecker) HasPermission(role, permission string) bool {
	perms, ok := r.permissionsByRole[role]
	if !ok {
		return false
	}
	return perms[permission]
}

// HasAnyPermission checks if any of the given roles has the given permission.
func (r *RBACChecker) HasAnyPermission(roles []string, permission string) bool {
	for _, role := range roles {
		if r.HasPermission(role, permission) {
			return true
		}
	}
	return false
}

// GetPermissionsForRoles returns all unique permissions for the given roles.
func (r *RBACChecker) GetPermissionsForRoles(roles []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, role := range roles {
		if perms, ok := r.permissionsByRole[role]; ok {
			for p := range perms {
				if !seen[p] {
					seen[p] = true
					result = append(result, p)
				}
			}
		}
	}
	return result
}

// HasAnyOfPermissions checks if any of the given roles has any of the given permissions.
func (r *RBACChecker) HasAnyOfPermissions(roles []string, permissions ...string) bool {
	for _, perm := range permissions {
		if r.HasAnyPermission(roles, perm) {
			return true
		}
	}
	return false
}

// transitionKey is used for the transition permission map lookup.
type transitionKey struct {
	Module     string
	FromStatus string
	ToStatus   string
}

// transitionPermissions maps (module, from_status, to_status) → required permission.
var transitionPermissions = map[transitionKey]string{
	// FX approval flow
	{constants.ModuleFX, constants.StatusOpen, constants.StatusPendingL2Approval}:                        constants.PermFxApproveL1,
	{constants.ModuleFX, constants.StatusPendingL2Approval, constants.StatusPendingBooking}:              constants.PermFxApproveL2,
	{constants.ModuleFX, constants.StatusPendingL2Approval, constants.StatusRejected}:                    constants.PermFxApproveL2,
	{constants.ModuleFX, constants.StatusPendingBooking, constants.StatusPendingChiefAccountant}:         constants.PermFxBookL1,
	{constants.ModuleFX, constants.StatusPendingBooking, constants.StatusVoidedByAccounting}:             constants.PermFxBookL1,
	{constants.ModuleFX, constants.StatusPendingChiefAccountant, constants.StatusPendingSettlement}:      constants.PermFxBookL2,
	{constants.ModuleFX, constants.StatusPendingChiefAccountant, constants.StatusCompleted}:              constants.PermFxBookL2,
	{constants.ModuleFX, constants.StatusPendingChiefAccountant, constants.StatusVoidedByAccounting}:     constants.PermFxBookL2,
	{constants.ModuleFX, constants.StatusPendingSettlement, constants.StatusCompleted}:                   constants.PermFxSettle,
	{constants.ModuleFX, constants.StatusPendingSettlement, constants.StatusVoidedBySettlement}:          constants.PermFxSettle,

	// FX TP review flow (TP recall → PENDING_TP_REVIEW, TP re-approves → PENDING_L2)
	{constants.ModuleFX, constants.StatusPendingTPReview, constants.StatusPendingL2Approval}:             constants.PermFxApproveL1,
	{constants.ModuleFX, constants.StatusPendingTPReview, constants.StatusOpen}:                          constants.PermFxApproveL1,

	// FX cancel approval flow
	{constants.ModuleFX, constants.StatusCompleted, constants.StatusPendingCancelL1}:                    constants.PermFxCancelRequest,
	{constants.ModuleFX, constants.StatusPendingSettlement, constants.StatusPendingCancelL1}:             constants.PermFxCancelRequest,
	{constants.ModuleFX, constants.StatusPendingCancelL1, constants.StatusPendingCancelL2}:               constants.PermFxCancelApproveL1,
	{constants.ModuleFX, constants.StatusPendingCancelL1, constants.StatusCompleted}:                     constants.PermFxCancelApproveL1,      // reject cancel → revert
	{constants.ModuleFX, constants.StatusPendingCancelL1, constants.StatusPendingSettlement}:              constants.PermFxCancelApproveL1,      // reject cancel → revert
	{constants.ModuleFX, constants.StatusPendingCancelL2, constants.StatusCancelled}:                     constants.PermFxCancelApproveL2,
	{constants.ModuleFX, constants.StatusPendingCancelL2, constants.StatusCompleted}:                     constants.PermFxCancelApproveL2,      // reject cancel → revert
	{constants.ModuleFX, constants.StatusPendingCancelL2, constants.StatusPendingSettlement}:              constants.PermFxCancelApproveL2,      // reject cancel → revert

	// BOND approval flow — NO risk check, NO settlement, 2-level accounting
	{constants.ModuleBond, constants.StatusOpen, constants.StatusPendingL2Approval}:                      constants.PermBondApproveL1,
	{constants.ModuleBond, constants.StatusPendingL2Approval, constants.StatusPendingBooking}:             constants.PermBondApproveL2,
	{constants.ModuleBond, constants.StatusPendingL2Approval, constants.StatusRejected}:                   constants.PermBondApproveL2,
	{constants.ModuleBond, constants.StatusPendingBooking, constants.StatusPendingChiefAccountant}:        constants.PermBondBookL1,
	{constants.ModuleBond, constants.StatusPendingBooking, constants.StatusVoidedByAccounting}:            constants.PermBondBookL1,
	{constants.ModuleBond, constants.StatusPendingChiefAccountant, constants.StatusCompleted}:             constants.PermBondBookL2,
	{constants.ModuleBond, constants.StatusPendingChiefAccountant, constants.StatusVoidedByAccounting}:    constants.PermBondBookL2,

	// BOND cancel approval flow
	{constants.ModuleBond, constants.StatusCompleted, constants.StatusPendingCancelL1}:                   constants.PermBondCancelRequest,
	{constants.ModuleBond, constants.StatusPendingCancelL1, constants.StatusPendingCancelL2}:              constants.PermBondCancelApproveL1,
	{constants.ModuleBond, constants.StatusPendingCancelL1, constants.StatusCompleted}:                    constants.PermBondCancelApproveL1,  // reject cancel → revert
	{constants.ModuleBond, constants.StatusPendingCancelL2, constants.StatusCancelled}:                    constants.PermBondCancelApproveL2,
	{constants.ModuleBond, constants.StatusPendingCancelL2, constants.StatusCompleted}:                    constants.PermBondCancelApproveL2,  // reject cancel → revert

	// MM INTERBANK approval flow — CV → TP → GĐ → QLRR L1 → QLRR L2 → KTTC L1 → KTTC L2 → [TTQT] → COMPLETED
	{constants.ModuleMMInterbank, constants.StatusOpen, constants.StatusPendingTPReview}:                        constants.PermMMInterbankApproveL1,
	{constants.ModuleMMInterbank, constants.StatusPendingTPReview, constants.StatusPendingL2Approval}:           constants.PermMMInterbankApproveL1,
	{constants.ModuleMMInterbank, constants.StatusPendingTPReview, constants.StatusOpen}:                        constants.PermMMInterbankApproveL1, // TP reject → OPEN
	{constants.ModuleMMInterbank, constants.StatusPendingL2Approval, constants.StatusPendingRiskApproval}:       constants.PermMMInterbankApproveL2,
	{constants.ModuleMMInterbank, constants.StatusPendingL2Approval, constants.StatusRejected}:                  constants.PermMMInterbankApproveL2,
	{constants.ModuleMMInterbank, constants.StatusPendingRiskApproval, constants.StatusPendingBooking}:          constants.PermMMInterbankApproveRiskL1, // QLRR L1 → pass to QLRR L2 (via PENDING_BOOKING as intermediate)
	{constants.ModuleMMInterbank, constants.StatusPendingRiskApproval, constants.StatusVoidedByRisk}:            constants.PermMMInterbankApproveRiskL1,
	{constants.ModuleMMInterbank, constants.StatusPendingBooking, constants.StatusPendingChiefAccountant}:       constants.PermMMInterbankBookL1,
	{constants.ModuleMMInterbank, constants.StatusPendingBooking, constants.StatusVoidedByAccounting}:           constants.PermMMInterbankBookL1,
	{constants.ModuleMMInterbank, constants.StatusPendingChiefAccountant, constants.StatusPendingSettlement}:    constants.PermMMInterbankBookL2,
	{constants.ModuleMMInterbank, constants.StatusPendingChiefAccountant, constants.StatusCompleted}:            constants.PermMMInterbankBookL2,
	{constants.ModuleMMInterbank, constants.StatusPendingChiefAccountant, constants.StatusVoidedByAccounting}:   constants.PermMMInterbankBookL2,
	{constants.ModuleMMInterbank, constants.StatusPendingSettlement, constants.StatusCompleted}:                 constants.PermMMInterbankSettle,
	{constants.ModuleMMInterbank, constants.StatusPendingSettlement, constants.StatusVoidedBySettlement}:        constants.PermMMInterbankSettle,

	// MM INTERBANK cancel flow
	{constants.ModuleMMInterbank, constants.StatusCompleted, constants.StatusPendingCancelL1}:                   constants.PermMMInterbankCancelRequest,
	{constants.ModuleMMInterbank, constants.StatusPendingCancelL1, constants.StatusPendingCancelL2}:              constants.PermMMInterbankCancelApproveL1,
	{constants.ModuleMMInterbank, constants.StatusPendingCancelL1, constants.StatusCompleted}:                    constants.PermMMInterbankCancelApproveL1, // reject
	{constants.ModuleMMInterbank, constants.StatusPendingCancelL2, constants.StatusCancelled}:                    constants.PermMMInterbankCancelApproveL2,
	{constants.ModuleMMInterbank, constants.StatusPendingCancelL2, constants.StatusCompleted}:                    constants.PermMMInterbankCancelApproveL2, // reject

	// MM OMO/REPO approval flow — CV → TP → GĐ → KTTC L1 → KTTC L2 → COMPLETED (no QLRR, no TTQT)
	{constants.ModuleMMOMORepo, constants.StatusOpen, constants.StatusPendingL2Approval}:                        constants.PermMMOMORepoApproveL1,
	{constants.ModuleMMOMORepo, constants.StatusPendingL2Approval, constants.StatusPendingBooking}:              constants.PermMMOMORepoApproveL2,
	{constants.ModuleMMOMORepo, constants.StatusPendingL2Approval, constants.StatusRejected}:                    constants.PermMMOMORepoApproveL2,
	{constants.ModuleMMOMORepo, constants.StatusPendingBooking, constants.StatusPendingChiefAccountant}:         constants.PermMMOMORepoBookL1,
	{constants.ModuleMMOMORepo, constants.StatusPendingBooking, constants.StatusVoidedByAccounting}:             constants.PermMMOMORepoBookL1,
	{constants.ModuleMMOMORepo, constants.StatusPendingChiefAccountant, constants.StatusCompleted}:              constants.PermMMOMORepoBookL2,
	{constants.ModuleMMOMORepo, constants.StatusPendingChiefAccountant, constants.StatusVoidedByAccounting}:     constants.PermMMOMORepoBookL2,

	// MM OMO/REPO cancel flow
	{constants.ModuleMMOMORepo, constants.StatusCompleted, constants.StatusPendingCancelL1}:                     constants.PermMMOMORepoCancelRequest,
	{constants.ModuleMMOMORepo, constants.StatusPendingCancelL1, constants.StatusPendingCancelL2}:                constants.PermMMOMORepoCancelApproveL1,
	{constants.ModuleMMOMORepo, constants.StatusPendingCancelL1, constants.StatusCompleted}:                      constants.PermMMOMORepoCancelApproveL1, // reject
	{constants.ModuleMMOMORepo, constants.StatusPendingCancelL2, constants.StatusCancelled}:                      constants.PermMMOMORepoCancelApproveL2,
	{constants.ModuleMMOMORepo, constants.StatusPendingCancelL2, constants.StatusCompleted}:                      constants.PermMMOMORepoCancelApproveL2, // reject
}

// GetRequiredPermission returns the permission needed for a specific status transition.
// Returns empty string if the transition is not recognized.
func GetRequiredPermission(module, fromStatus, toStatus string) string {
	key := transitionKey{Module: module, FromStatus: fromStatus, ToStatus: toStatus}
	return transitionPermissions[key]
}
