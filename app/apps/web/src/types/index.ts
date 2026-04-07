/**
 * Shared TypeScript types cho Treasury System
 */

// ─── Chung ──────────────────────────────────────────────────
export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}

export type TransactionStatus =
  | "draft"       // Nháp
  | "pending"     // Chờ duyệt
  | "approved"    // Đã duyệt
  | "rejected"    // Từ chối
  | "settled"     // Đã thanh toán
  | "cancelled";  // Đã hủy

// ─── FX — Kinh doanh Ngoại tệ ──────────────────────────────
export interface FXTransaction {
  id: string;
  dealType: "spot" | "forward" | "swap";
  buyCurrency: string;
  sellCurrency: string;
  buyAmount: number;
  sellAmount: number;
  exchangeRate: number;
  valueDate: string;
  counterparty: string;
  status: TransactionStatus;
  createdAt: string;
  updatedAt: string;
}

// ─── GTCG — Giấy tờ có giá ─────────────────────────────────
export interface SecuritiesHolding {
  id: string;
  securityType: "bond" | "tbill" | "certificate";
  issuer: string;
  faceValue: number;
  purchasePrice: number;
  couponRate: number;
  maturityDate: string;
  status: "active" | "matured" | "sold";
  createdAt: string;
  updatedAt: string;
}

// ─── MM — Thị trường Tiền tệ ───────────────────────────────
export interface MMTransaction {
  id: string;
  dealType: "deposit" | "loan" | "repo";
  counterparty: string;
  amount: number;
  currency: string;
  interestRate: number;
  tenor: string;
  startDate: string;
  maturityDate: string;
  status: TransactionStatus;
  createdAt: string;
  updatedAt: string;
}

// ─── Limits — Hạn mức ──────────────────────────────────────
export interface Limit {
  id: string;
  counterparty: string;
  limitType: "fx" | "mm" | "gtcg" | "settlement" | "total";
  approvedAmount: number;
  usedAmount: number;
  availableAmount: number;
  currency: string;
  effectiveDate: string;
  expiryDate: string;
  status: "active" | "expired" | "suspended";
  createdAt: string;
  updatedAt: string;
}

// ─── Settlement — Thanh toán Quốc tế ────────────────────────
export interface Settlement {
  id: string;
  transactionType: "lc" | "tt" | "dp" | "collection";
  counterparty: string;
  amount: number;
  currency: string;
  beneficiary: string;
  effectiveDate: string;
  expiryDate: string;
  status: TransactionStatus;
  createdAt: string;
  updatedAt: string;
}

// ─── User / Auth ────────────────────────────────────────────
export interface User {
  id: string;
  name: string;
  email: string;
  role: string;
  department: string;
}
