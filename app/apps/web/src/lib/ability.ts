import {
  AbilityBuilder,
  createMongoAbility,
  type MongoAbility,
} from "@casl/ability";

// ─── Subjects ────────────────────────────────────────────────
type Subjects =
  | "FXTransaction"      // Giao dịch ngoại tệ
  | "GTCGTransaction"    // Giấy tờ có giá
  | "MMTransaction"      // Kinh doanh tiền tệ
  | "Limit"              // Hạn mức liên ngân hàng
  | "Settlement"         // Thanh toán quốc tế
  | "Partner"            // Đối tác
  | "Securities"         // Danh mục trái phiếu
  | "AuditLog"           // Nhật ký
  | "Settings"           // Cài đặt
  | "all";

// ─── Actions ─────────────────────────────────────────────────
type Actions =
  | "view"
  | "create"
  | "update"
  | "delete"
  | "approve"
  | "recall"
  | "cancel"
  | "export"
  | "manage";

// ─── Ability type ────────────────────────────────────────────
export type AppAbility = MongoAbility<[Actions, Subjects]>;

// ─── Permission code → CASL mapping ─────────────────────────
const PERMISSION_MAP: Record<string, [Actions, Subjects]> = {
  // FX — Giao dịch ngoại tệ (backend: FX_DEAL.ACTION)
  "FX_DEAL.VIEW": ["view", "FXTransaction"],
  "FX_DEAL.CREATE": ["create", "FXTransaction"],
  "FX_DEAL.EDIT": ["update", "FXTransaction"],
  "FX_DEAL.APPROVE_L1": ["approve", "FXTransaction"],
  "FX_DEAL.APPROVE_L2": ["approve", "FXTransaction"],
  "FX_DEAL.RECALL": ["recall", "FXTransaction"],
  "FX_DEAL.CANCEL_REQUEST": ["cancel", "FXTransaction"],
  "FX_DEAL.CANCEL_APPROVE_L1": ["cancel", "FXTransaction"],
  "FX_DEAL.CANCEL_APPROVE_L2": ["cancel", "FXTransaction"],
  "FX_DEAL.EXPORT": ["export", "FXTransaction"],
  "FX_DEAL.DELETE": ["delete", "FXTransaction"],
  "FX_DEAL.CLONE": ["create", "FXTransaction"], // clone = create new
  "FX_DEAL.BOOK_L1": ["approve", "FXTransaction"],
  "FX_DEAL.BOOK_L2": ["approve", "FXTransaction"],
  "FX_DEAL.SETTLE": ["approve", "FXTransaction"],

  // GTCG / Bond — Giấy tờ có giá (backend: BOND_DEAL.ACTION)
  "BOND_DEAL.VIEW": ["view", "GTCGTransaction"],
  "BOND_DEAL.CREATE": ["create", "GTCGTransaction"],
  "BOND_DEAL.EDIT": ["update", "GTCGTransaction"],
  "BOND_DEAL.APPROVE_L1": ["approve", "GTCGTransaction"],
  "BOND_DEAL.APPROVE_L2": ["approve", "GTCGTransaction"],
  "BOND_DEAL.BOOK_L1": ["approve", "GTCGTransaction"],
  "BOND_DEAL.BOOK_L2": ["approve", "GTCGTransaction"],

  // MM — Kinh doanh tiền tệ (backend: MM_INTERBANK_DEAL.ACTION)
  "MM_INTERBANK_DEAL.VIEW": ["view", "MMTransaction"],
  "MM_INTERBANK_DEAL.CREATE": ["create", "MMTransaction"],
  "MM_INTERBANK_DEAL.EDIT": ["update", "MMTransaction"],
  "MM_INTERBANK_DEAL.APPROVE_L1": ["approve", "MMTransaction"],
  "MM_INTERBANK_DEAL.APPROVE_RISK_L1": ["approve", "MMTransaction"],
  "MM_INTERBANK_DEAL.APPROVE_RISK_L2": ["approve", "MMTransaction"],
  "MM_INTERBANK_DEAL.BOOK_L1": ["approve", "MMTransaction"],
  "MM_INTERBANK_DEAL.BOOK_L2": ["approve", "MMTransaction"],
  "MM_OMO_REPO_DEAL.VIEW": ["view", "MMTransaction"],
  "MM_OMO_REPO_DEAL.BOOK_L1": ["approve", "MMTransaction"],
  "MM_OMO_REPO_DEAL.BOOK_L2": ["approve", "MMTransaction"],

  // Limits — Hạn mức liên ngân hàng (backend: CREDIT_LIMIT.ACTION)
  "CREDIT_LIMIT.VIEW": ["view", "Limit"],
  "CREDIT_LIMIT.CREATE": ["create", "Limit"],
  "CREDIT_LIMIT.APPROVE_L1": ["approve", "Limit"],
  "CREDIT_LIMIT.APPROVE_RISK_L1": ["approve", "Limit"],
  "CREDIT_LIMIT.APPROVE_RISK_L2": ["approve", "Limit"],

  // International Settlements (backend: INTERNATIONAL_PAYMENT.ACTION)
  "INTERNATIONAL_PAYMENT.VIEW": ["view", "Settlement"],
  "INTERNATIONAL_PAYMENT.CREATE": ["create", "Settlement"],
  "INTERNATIONAL_PAYMENT.SETTLE": ["approve", "Settlement"],

  // Partner / Master Data — Đối tác (backend: MASTER_DATA.ACTION)
  "MASTER_DATA.VIEW": ["view", "Partner"],
  "MASTER_DATA.MANAGE": ["manage", "Partner"],

  // Audit — Nhật ký (backend: AUDIT_LOG.VIEW)
  "AUDIT_LOG.VIEW": ["view", "AuditLog"],

  // System — Cài đặt (backend: SYSTEM.MANAGE)
  "SYSTEM.MANAGE": ["manage", "Settings"],
};

/**
 * Build CASL ability từ danh sách permission codes
 */
export function defineAbilityFor(permissions: string[]): AppAbility {
  const { can, build } = new AbilityBuilder<AppAbility>(createMongoAbility);

  for (const code of permissions) {
    const mapping = PERMISSION_MAP[code];
    if (mapping) {
      can(mapping[0], mapping[1]);
    }
  }

  return build();
}

/**
 * Check quyền qua CASL ability
 */
export function hasPermission(
  ability: AppAbility,
  action: Actions,
  subject: Subjects
): boolean {
  return ability.can(action, subject);
}

export { PERMISSION_MAP };
export type { Actions, Subjects };
