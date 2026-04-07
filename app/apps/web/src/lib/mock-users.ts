/**
 * DEV_ACCOUNTS — Quick-login reference for development mode only.
 * Maps to real backend usernames. Password for all: P@ssw0rd123
 */

export interface DevAccount {
  username: string;
  password: string;
  label: string;
  name: string;
  department: string;
}

export const DEV_ACCOUNTS: DevAccount[] = [
  { username: "dealer01", password: "P@ssw0rd123", label: "Dealer", name: "Nguyễn Văn An", department: "K.NV" },
  { username: "deskhead01", password: "P@ssw0rd123", label: "Desk Head", name: "Trần Thị Bình", department: "K.NV" },
  { username: "director01", password: "P@ssw0rd123", label: "Center Director", name: "Lê Minh Cường", department: "K.NV" },
  { username: "divhead01", password: "P@ssw0rd123", label: "Division Head", name: "Vũ Hoàng Hải", department: "K.NV" },
  { username: "risk01", password: "P@ssw0rd123", label: "Risk Officer", name: "Ngô Thị Phương", department: "P.QLRR" },
  { username: "riskhead01", password: "P@ssw0rd123", label: "Risk Head", name: "Đỗ Văn Giang", department: "P.QLRR" },
  { username: "accountant01", password: "P@ssw0rd123", label: "Accountant", name: "Phạm Thị Dung", department: "P.KTTC" },
  { username: "chiefacc01", password: "P@ssw0rd123", label: "Chief Accountant", name: "Bùi Thị Khánh", department: "P.KTTC" },
  { username: "settlement01", password: "P@ssw0rd123", label: "Settlement Officer", name: "Hoàng Văn Em", department: "International Settlements" },
  { username: "admin01", password: "P@ssw0rd123", label: "System Admin", name: "Nguyễn Văn Minh", department: "K.CN" },
];
