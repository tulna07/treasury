"""
Generate 60 realistic Bond deals + 15 bond catalog entries + inventory + approval actions.
Output: migrations/seed/003_bond_deals_seed.sql
Also executes directly against the database.
"""

import os
import random
import uuid
from datetime import date, timedelta
from decimal import Decimal, ROUND_HALF_UP

try:
    from faker import Faker
except ImportError:
    raise SystemExit("Faker not installed. Run: pip install Faker")

try:
    import psycopg2
except ImportError:
    raise SystemExit("psycopg2 not installed. Run: pip install psycopg2-binary")

# ─── Configuration ───────────────────────────────────────────────────────────

NUM_DEALS = 60
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
API_DIR = os.path.dirname(SCRIPT_DIR)
OUTPUT_FILE = os.path.join(API_DIR, "migrations", "seed", "003_bond_deals_seed.sql")
DB_URL = "postgresql://mrm@localhost:5432/treasury"

# Existing seed IDs (from 001_seed.sql)
BRANCH_ID = "a0000000-0000-0000-0000-000000000001"
DEALER_USER_ID = "d0000000-0000-0000-0000-000000000001"   # dealer01
DESKHEAD_USER_ID = "d0000000-0000-0000-0000-000000000002"  # deskhead01
DIRECTOR_USER_ID = "d0000000-0000-0000-0000-000000000003"  # director01
ACCOUNTANT_USER_ID = "d0000000-0000-0000-0000-000000000004"  # accountant01
CHIEF_ACC_USER_ID = "d0000000-0000-0000-0000-000000000009"  # chiefacc01
ADMIN_USER_ID = "d0000000-0000-0000-0000-000000000010"  # admin01

COUNTERPARTY_IDS = [
    f"e0000000-0000-0000-0000-00000000000{i}" for i in range(1, 10)
] + ["e0000000-0000-0000-0000-000000000010"]  # SHB

# Counterparty names for FI/CCTG issuers
COUNTERPARTY_BANKS = {
    "e0000000-0000-0000-0000-000000000001": "Ngân hàng TMCP Hàng Hải Việt Nam (MSB)",
    "e0000000-0000-0000-0000-000000000002": "Ngân hàng TMCP Á Châu (ACB)",
    "e0000000-0000-0000-0000-000000000003": "Ngân hàng TMCP Ngoại thương Việt Nam (VCB)",
    "e0000000-0000-0000-0000-000000000004": "Ngân hàng TMCP Kỹ thương Việt Nam (TCB)",
    "e0000000-0000-0000-0000-000000000005": "Ngân hàng TMCP Việt Nam Thịnh Vượng (VPB)",
    "e0000000-0000-0000-0000-000000000006": "Ngân hàng TMCP Quân đội (MBB)",
    "e0000000-0000-0000-0000-000000000007": "Ngân hàng TMCP Đầu tư và Phát triển Việt Nam (BIDV)",
    "e0000000-0000-0000-0000-000000000008": "Ngân hàng TMCP Công Thương Việt Nam (VietinBank)",
    "e0000000-0000-0000-0000-000000000009": "Ngân hàng TMCP Sài Gòn Thương Tín (STB)",
    "e0000000-0000-0000-0000-000000000010": "Ngân hàng TMCP Sài Gòn - Hà Nội (SHB)",
}

FI_ISSUERS = [
    "Ngân hàng TMCP Hàng Hải Việt Nam (MSB)",
    "Ngân hàng TMCP Á Châu (ACB)",
    "Ngân hàng TMCP Ngoại thương Việt Nam (VCB)",
    "Ngân hàng TMCP Đầu tư và Phát triển Việt Nam (BIDV)",
    "Ngân hàng TMCP Công Thương Việt Nam (VietinBank)",
    "Ngân hàng TMCP Kỹ thương Việt Nam (Techcombank)",
    "Ngân hàng TMCP Việt Nam Thịnh Vượng (VPBank)",
    "Ngân hàng TMCP Quân đội (MB Bank)",
    "Ngân hàng TMCP Sài Gòn Thương Tín (Sacombank)",
    "Ngân hàng TMCP Sài Gòn - Hà Nội (SHB)",
]

CCTG_ISSUERS = FI_ISSUERS + [
    "Ngân hàng TMCP Kiên Long (KienlongBank)",
]

# ─── Bond Catalog — 15 Vietnamese Government Bonds (TPCP) ───────────────────

BOND_CATALOG = [
    # (id, bond_code, coupon_rate, payment_frequency, issue_date, maturity_date, tenor_label)
    ("bc000000-0000-0000-0000-000000000001", "TD2125068", 5.25, "SEMI_ANNUAL", "2021-06-15", "2031-06-15", "10yr"),
    ("bc000000-0000-0000-0000-000000000002", "TD2230045", 4.80, "ANNUAL", "2022-03-20", "2027-03-20", "5yr"),
    ("bc000000-0000-0000-0000-000000000003", "TD2335012", 3.50, "ANNUAL", "2023-01-10", "2026-01-10", "3yr"),
    ("bc000000-0000-0000-0000-000000000004", "TD2440078", 5.80, "SEMI_ANNUAL", "2024-04-01", "2039-04-01", "15yr"),
    ("bc000000-0000-0000-0000-000000000005", "TD2145023", 6.20, "SEMI_ANNUAL", "2021-09-01", "2041-09-01", "20yr"),
    ("bc000000-0000-0000-0000-000000000006", "TD2525034", 4.20, "ANNUAL", "2025-02-15", "2030-02-15", "5yr"),
    ("bc000000-0000-0000-0000-000000000007", "TD2328056", 3.80, "ANNUAL", "2023-07-01", "2026-07-01", "3yr"),
    ("bc000000-0000-0000-0000-000000000008", "TD2432089", 5.50, "SEMI_ANNUAL", "2024-08-15", "2034-08-15", "10yr"),
    ("bc000000-0000-0000-0000-000000000009", "TD2526011", 4.50, "ANNUAL", "2025-01-05", "2026-01-05", "1yr"),
    ("bc000000-0000-0000-0000-00000000000a", "TD2538067", 6.00, "SEMI_ANNUAL", "2025-03-01", "2040-03-01", "15yr"),
    ("bc000000-0000-0000-0000-00000000000b", "TD2245090", 4.00, "ANNUAL", "2022-11-10", "2027-11-10", "5yr"),
    ("bc000000-0000-0000-0000-00000000000c", "TD2350034", 5.75, "SEMI_ANNUAL", "2023-05-20", "2033-05-20", "10yr"),
    ("bc000000-0000-0000-0000-00000000000d", "TD2427045", 2.80, "ZERO_COUPON", "2024-06-01", "2025-06-01", "1yr"),
    ("bc000000-0000-0000-0000-00000000000e", "TD2543012", 6.50, "SEMI_ANNUAL", "2025-04-01", "2045-04-01", "20yr"),
    ("bc000000-0000-0000-0000-00000000000f", "TD2629078", 3.20, "ANNUAL", "2026-01-15", "2029-01-15", "3yr"),
]

# ─── Deal Distribution ──────────────────────────────────────────────────────

# 30 GOVERNMENT, 15 FINANCIAL_INSTITUTION, 15 CERTIFICATE_OF_DEPOSIT
CATEGORY_LIST = (
    ["GOVERNMENT"] * 30
    + ["FINANCIAL_INSTITUTION"] * 15
    + ["CERTIFICATE_OF_DEPOSIT"] * 15
)

# Status distribution: 10+8+8+5+15+5+4+3+2 = 60
STATUS_LIST = (
    ["OPEN"] * 10
    + ["PENDING_L2_APPROVAL"] * 8
    + ["PENDING_BOOKING"] * 8
    + ["PENDING_CHIEF_ACCOUNTANT"] * 5
    + ["COMPLETED"] * 15
    + ["REJECTED"] * 5
    + ["CANCELLED"] * 4
    + ["VOIDED_BY_ACCOUNTING"] * 3
    + ["PENDING_CANCEL_L1"] * 2
)

# Transaction types: 40% Outright, 30% Repo, 20% Reverse Repo, 10% Other = 24+18+12+6
TX_TYPE_LIST = (
    ["OUTRIGHT"] * 24
    + ["REPO"] * 18
    + ["REVERSE_REPO"] * 12
    + ["OTHER"] * 6
)

PORTFOLIO_TYPES = ["HTM", "AFS", "HFT"]
CONFIRMATION_METHODS = ["EMAIL", "REUTERS", "OTHER"]
CONTRACT_BY = ["INTERNAL", "COUNTERPARTY"]

# ─── Helpers ─────────────────────────────────────────────────────────────────

fake = Faker()
random.seed(42)
Faker.seed(42)

# Deterministic UUIDs using seeded random
_uuid_rng = random.Random(12345)


def det_uuid() -> str:
    """Generate a deterministic UUID from seeded RNG."""
    return str(uuid.UUID(int=_uuid_rng.getrandbits(128), version=4))


def sql_escape(val: str) -> str:
    return val.replace("'", "''")


def rand_date_between(start: date, end: date) -> date:
    delta = (end - start).days
    return start + timedelta(days=random.randint(0, max(0, delta)))


# ─── Generate Bond Catalog SQL ──────────────────────────────────────────────

def generate_bond_catalog() -> list[str]:
    lines = []
    for cat_id, code, rate, freq, issue, maturity, _label in BOND_CATALOG:
        lines.append(
            f"INSERT INTO bond_catalog (id, bond_code, issuer, coupon_rate, payment_frequency, "
            f"issue_date, maturity_date, face_value, bond_type, is_active, created_by, updated_by)\n"
            f"VALUES ('{cat_id}', '{code}', 'Kho bạc Nhà nước', {rate:.4f}, '{freq}', "
            f"'{issue}', '{maturity}', 100000, 'GOVERNMENT', true, '{ADMIN_USER_ID}', '{ADMIN_USER_ID}')\n"
            f"ON CONFLICT (bond_code) DO NOTHING;"
        )
    return lines


# ─── Generate Bond Deals ────────────────────────────────────────────────────

def generate():
    # Shuffle all lists together
    combined = list(zip(CATEGORY_LIST, STATUS_LIST, TX_TYPE_LIST))
    random.shuffle(combined)
    categories, statuses, tx_types = zip(*combined)

    deal_inserts = []
    inventory_map = {}  # (bond_code, category, portfolio) -> quantity for COMPLETED BUY
    approval_inserts = []
    deal_data = []  # track for inventory + approval generation

    # Track deal numbers per date prefix
    govi_seq = 0
    fi_seq = 0

    for i in range(NUM_DEALS):
        deal_id = det_uuid()
        category = categories[i]
        status = statuses[i]
        tx_type = tx_types[i]

        # Direction: 60% BUY, 40% SELL (but SELL only for COMPLETED to have inventory)
        if status == "COMPLETED" and random.random() < 0.4:
            direction = "SELL"
        else:
            direction = "BUY" if random.random() < 0.75 else "SELL"

        # Trade date spread: 2026-03-25 to 2026-04-04
        td = date(2026, 3, 25) + timedelta(days=random.randint(0, 10))
        # Skip weekends
        while td.weekday() >= 5:
            td += timedelta(days=1)

        # Deal number
        day_str = td.strftime("%Y%m%d")
        if category == "GOVERNMENT":
            govi_seq += 1
            dn = f"G-{day_str}-{govi_seq:04d}"
        else:
            fi_seq += 1
            dn = f"F-{day_str}-{fi_seq:04d}"

        cp_id = random.choice(COUNTERPARTY_IDS)

        # Bond info
        bond_catalog_id = None
        bond_code_manual = None
        issuer = ""
        coupon_rate = Decimal(str(round(random.uniform(2.5, 6.5), 4)))

        if category == "GOVERNMENT":
            cat_entry = random.choice(BOND_CATALOG)
            bond_catalog_id = cat_entry[0]
            issuer = "Kho bạc Nhà nước"
            coupon_rate = Decimal(str(cat_entry[2]))
            issue_date_str = cat_entry[4]
            maturity_date_str = cat_entry[5]
            bond_code_for_inv = cat_entry[1]
        elif category == "FINANCIAL_INSTITUTION":
            issuer = random.choice(FI_ISSUERS)
            short_code = issuer.split("(")[-1].rstrip(")")
            year = random.choice(["23", "24", "25"])
            bond_code_manual = f"FI-{short_code}-{year}{random.randint(1,99):02d}"
            issue_date_val = date(2020 + random.randint(3, 5), random.randint(1, 12), random.randint(1, 28))
            tenor_years = random.choice([1, 2, 3, 5])
            maturity_val = issue_date_val + timedelta(days=tenor_years * 365)
            issue_date_str = issue_date_val.isoformat()
            maturity_date_str = maturity_val.isoformat()
            bond_code_for_inv = bond_code_manual
        else:  # CERTIFICATE_OF_DEPOSIT
            issuer = random.choice(CCTG_ISSUERS)
            short_code = issuer.split("(")[-1].rstrip(")")
            bond_code_manual = f"CD-{short_code}-{random.choice(['25', '26'])}{random.randint(1,99):02d}"
            issue_date_val = date(2025, random.randint(1, 12), random.randint(1, 28))
            tenor_months = random.choice([3, 6, 9, 12])
            maturity_val = issue_date_val + timedelta(days=tenor_months * 30)
            issue_date_str = issue_date_val.isoformat()
            maturity_date_str = maturity_val.isoformat()
            bond_code_for_inv = bond_code_manual

        # Dates
        value_date = td + timedelta(days=random.choice([1, 2, 3]))
        order_date = td - timedelta(days=random.randint(0, 2))
        payment_date = value_date
        maturity_date_obj = date.fromisoformat(maturity_date_str)
        remaining_tenor = max(1, (maturity_date_obj - payment_date).days)

        # Pricing
        quantity = random.choice([100, 500, 1000, 2000, 5000, 10000, 50000, 100000, 500000, 1000000, 2000000, 5000000])
        face_value = 100000
        clean_price = random.randint(95000, 130000)
        # Settlement price slightly higher (accrued interest)
        settlement_price = clean_price + random.randint(0, 3000)
        total_value = quantity * settlement_price
        discount_rate = Decimal(str(round(random.uniform(0, 5.0), 4)))

        portfolio_type = random.choice(PORTFOLIO_TYPES) if direction == "BUY" else None
        confirmation = random.choice(CONFIRMATION_METHODS)
        contract_by = random.choice(CONTRACT_BY)

        tx_type_other = None
        if tx_type == "OTHER":
            tx_type_other = random.choice([
                "Mua bán có kỳ hạn",
                "Giao dịch điều kiện",
                "Giao dịch thỏa thuận",
            ])

        # Cancel fields
        cancel_reason = None
        cancel_requested_by = None
        cancel_requested_at = None
        if status == "REJECTED":
            cancel_reason = random.choice([
                "Giá không hợp lý so với thị trường",
                "Vượt hạn mức đầu tư trái phiếu",
                "Thiếu hồ sơ pháp lý đối tác",
                "Không đạt tiêu chuẩn rủi ro",
                "Hết hạn mức phê duyệt trong ngày",
            ])
        elif status in ("CANCELLED", "PENDING_CANCEL_L1"):
            cancel_reason = random.choice([
                "Đối tác yêu cầu hủy giao dịch",
                "Thay đổi điều kiện thị trường",
                "Deal trùng lặp — đã có giao dịch tương tự",
                "Sai thông tin trái phiếu",
            ])
            cancel_requested_by = DEALER_USER_ID
            cancel_requested_at = f"{td}T10:00:00+07:00"
        elif status == "VOIDED_BY_ACCOUNTING":
            cancel_reason = random.choice([
                "Sai số liệu hạch toán",
                "Yêu cầu điều chỉnh từ kiểm toán",
                "Phát hiện sai sót sau phê duyệt",
            ])

        note = None
        if random.random() < 0.3:
            note = random.choice([
                "Giao dịch theo yêu cầu khách hàng VIP",
                "Trái phiếu đợt phát hành mới",
                "Giá tham chiếu HNX ngày giao dịch",
                "Giao dịch thỏa thuận ngoài sàn",
                "Yêu cầu xử lý gấp trong ngày",
                "Tái cơ cấu danh mục đầu tư",
            ])

        created_at = f"{td}T09:{random.randint(0,59):02d}:{random.randint(0,59):02d}+07:00"

        # Build INSERT
        cols = [
            "id", "deal_number", "bond_category", "trade_date", "branch_id",
            "order_date", "value_date", "direction", "counterparty_id",
            "transaction_type", "issuer", "coupon_rate",
            "issue_date", "maturity_date", "quantity", "face_value",
            "discount_rate", "clean_price", "settlement_price", "total_value",
            "payment_date", "remaining_tenor_days",
            "confirmation_method", "contract_prepared_by",
            "status", "created_by", "updated_by", "created_at", "updated_at",
        ]
        vals = [
            f"'{deal_id}'",
            f"'{dn}'",
            f"'{category}'",
            f"'{td.isoformat()}'",
            f"'{BRANCH_ID}'",
            f"'{order_date.isoformat()}'",
            f"'{value_date.isoformat()}'",
            f"'{direction}'",
            f"'{cp_id}'",
            f"'{tx_type}'",
            f"'{sql_escape(issuer)}'",
            f"{coupon_rate}",
            f"'{issue_date_str}'",
            f"'{maturity_date_str}'",
            f"{quantity}",
            f"{face_value}",
            f"{discount_rate}",
            f"{clean_price}",
            f"{settlement_price}",
            f"{total_value}",
            f"'{payment_date.isoformat()}'",
            f"{remaining_tenor}",
            f"'{confirmation}'",
            f"'{contract_by}'",
            f"'{status}'",
            f"'{DEALER_USER_ID}'",
            f"'{DEALER_USER_ID}'",
            f"'{created_at}'",
            f"'{created_at}'",
        ]

        if bond_catalog_id:
            cols.append("bond_catalog_id")
            vals.append(f"'{bond_catalog_id}'")
        if bond_code_manual:
            cols.append("bond_code_manual")
            vals.append(f"'{bond_code_manual}'")
        if portfolio_type:
            cols.append("portfolio_type")
            vals.append(f"'{portfolio_type}'")
        if tx_type_other:
            cols.append("transaction_type_other")
            vals.append(f"'{sql_escape(tx_type_other)}'")
        if cancel_reason:
            cols.append("cancel_reason")
            vals.append(f"'{sql_escape(cancel_reason)}'")
        if cancel_requested_by:
            cols.append("cancel_requested_by")
            vals.append(f"'{cancel_requested_by}'")
            cols.append("cancel_requested_at")
            vals.append(f"'{cancel_requested_at}'")
        if note:
            cols.append("note")
            vals.append(f"'{sql_escape(note)}'")

        deal_inserts.append(
            f"-- Deal {i+1}: {dn} | {category} {direction} {tx_type} | {status}\n"
            f"INSERT INTO bond_deals ({', '.join(cols)})\n"
            f"VALUES ({', '.join(vals)})\n"
            f"ON CONFLICT (deal_number) DO NOTHING;"
        )

        # Track data for inventory and approvals
        deal_data.append({
            "deal_id": deal_id,
            "deal_number": dn,
            "category": category,
            "direction": direction,
            "status": status,
            "quantity": quantity,
            "settlement_price": settlement_price,
            "bond_code": bond_code_for_inv,
            "bond_catalog_id": bond_catalog_id,
            "portfolio_type": portfolio_type or "HTM",
            "trade_date": td,
            "value_date": value_date,
        })

        # Accumulate inventory for COMPLETED BUY deals
        if status == "COMPLETED" and direction == "BUY":
            inv_key = (bond_code_for_inv, category, portfolio_type or "HTM")
            inventory_map[inv_key] = inventory_map.get(inv_key, 0) + quantity

        # Generate approval actions for deals past OPEN
        if status != "OPEN":
            approvals = _generate_approvals(deal_id, status, td)
            approval_inserts.extend(approvals)

    # Generate inventory inserts
    inventory_inserts = _generate_inventory(inventory_map)

    return deal_inserts, inventory_inserts, approval_inserts


def _generate_approvals(deal_id: str, status: str, trade_date: date) -> list[str]:
    """Generate approval_actions for a deal based on its current status."""
    actions = []
    base_time = f"{trade_date.isoformat()}T"

    # Status flow: OPEN → PENDING_L2_APPROVAL → PENDING_BOOKING → PENDING_CHIEF_ACCOUNTANT → COMPLETED
    if status == "PENDING_L2_APPROVAL":
        actions.append(_approval_sql(
            deal_id, "DESK_HEAD_APPROVE", "OPEN", "PENDING_L2_APPROVAL",
            DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Đã kiểm tra, phê duyệt"
        ))

    elif status == "REJECTED":
        # Could be rejected at L1 or L2
        if random.random() < 0.5:
            actions.append(_approval_sql(
                deal_id, "DESK_HEAD_APPROVE", "OPEN", "PENDING_L2_APPROVAL",
                DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
                "Phê duyệt cấp 1"
            ))
            actions.append(_approval_sql(
                deal_id, "DIRECTOR_REJECT", "PENDING_L2_APPROVAL", "REJECTED",
                DIRECTOR_USER_ID, f"{base_time}11:{random.randint(0,59):02d}:00+07:00",
                random.choice(["Giá không hợp lý", "Vượt hạn mức", "Cần xem lại điều kiện"])
            ))
        else:
            actions.append(_approval_sql(
                deal_id, "DESK_HEAD_RETURN", "OPEN", "REJECTED",
                DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
                random.choice(["Trả lại để bổ sung hồ sơ", "Sai thông tin trái phiếu"])
            ))

    elif status == "PENDING_BOOKING":
        actions.append(_approval_sql(
            deal_id, "DESK_HEAD_APPROVE", "OPEN", "PENDING_L2_APPROVAL",
            DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt cấp 1"
        ))
        actions.append(_approval_sql(
            deal_id, "DIRECTOR_APPROVE", "PENDING_L2_APPROVAL", "PENDING_BOOKING",
            DIRECTOR_USER_ID, f"{base_time}11:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt cấp 2, chuyển hạch toán"
        ))

    elif status == "PENDING_CHIEF_ACCOUNTANT":
        actions.append(_approval_sql(
            deal_id, "DESK_HEAD_APPROVE", "OPEN", "PENDING_L2_APPROVAL",
            DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt"
        ))
        actions.append(_approval_sql(
            deal_id, "DIRECTOR_APPROVE", "PENDING_L2_APPROVAL", "PENDING_BOOKING",
            DIRECTOR_USER_ID, f"{base_time}11:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt cấp 2"
        ))
        actions.append(_approval_sql(
            deal_id, "ACCOUNTANT_APPROVE", "PENDING_BOOKING", "PENDING_CHIEF_ACCOUNTANT",
            ACCOUNTANT_USER_ID, f"{base_time}14:{random.randint(0,59):02d}:00+07:00",
            "Đã hạch toán, chuyển kế toán trưởng"
        ))

    elif status == "COMPLETED":
        actions.append(_approval_sql(
            deal_id, "DESK_HEAD_APPROVE", "OPEN", "PENDING_L2_APPROVAL",
            DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt"
        ))
        actions.append(_approval_sql(
            deal_id, "DIRECTOR_APPROVE", "PENDING_L2_APPROVAL", "PENDING_BOOKING",
            DIRECTOR_USER_ID, f"{base_time}11:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt cấp 2"
        ))
        actions.append(_approval_sql(
            deal_id, "ACCOUNTANT_APPROVE", "PENDING_BOOKING", "PENDING_CHIEF_ACCOUNTANT",
            ACCOUNTANT_USER_ID, f"{base_time}14:{random.randint(0,59):02d}:00+07:00",
            "Đã hạch toán"
        ))
        actions.append(_approval_sql(
            deal_id, "CHIEF_ACCOUNTANT_APPROVE", "PENDING_CHIEF_ACCOUNTANT", "COMPLETED",
            CHIEF_ACC_USER_ID, f"{base_time}15:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt hoàn tất"
        ))

    elif status == "VOIDED_BY_ACCOUNTING":
        # Was completed, then voided
        actions.append(_approval_sql(
            deal_id, "DESK_HEAD_APPROVE", "OPEN", "PENDING_L2_APPROVAL",
            DESKHEAD_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt"
        ))
        actions.append(_approval_sql(
            deal_id, "DIRECTOR_APPROVE", "PENDING_L2_APPROVAL", "PENDING_BOOKING",
            DIRECTOR_USER_ID, f"{base_time}11:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt cấp 2"
        ))
        actions.append(_approval_sql(
            deal_id, "ACCOUNTANT_APPROVE", "PENDING_BOOKING", "PENDING_CHIEF_ACCOUNTANT",
            ACCOUNTANT_USER_ID, f"{base_time}14:{random.randint(0,59):02d}:00+07:00",
            "Hạch toán"
        ))
        actions.append(_approval_sql(
            deal_id, "CHIEF_ACCOUNTANT_APPROVE", "PENDING_CHIEF_ACCOUNTANT", "COMPLETED",
            CHIEF_ACC_USER_ID, f"{base_time}15:{random.randint(0,59):02d}:00+07:00",
            "Phê duyệt"
        ))
        # Then chief accountant voids
        next_day = (trade_date + timedelta(days=1)).isoformat()
        actions.append(_approval_sql(
            deal_id, "CHIEF_ACCOUNTANT_REJECT", "COMPLETED", "VOIDED_BY_ACCOUNTING",
            CHIEF_ACC_USER_ID, f"{next_day}T09:{random.randint(0,59):02d}:00+07:00",
            random.choice(["Phát hiện sai sót hạch toán", "Yêu cầu điều chỉnh từ kiểm toán"])
        ))

    elif status == "CANCELLED":
        actions.append(_approval_sql(
            deal_id, "CANCEL_REQUEST", "OPEN", "PENDING_CANCEL_L1",
            DEALER_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Yêu cầu hủy giao dịch"
        ))
        actions.append(_approval_sql(
            deal_id, "CANCEL_DESK_HEAD_APPROVE", "PENDING_CANCEL_L1", "CANCELLED",
            DESKHEAD_USER_ID, f"{base_time}11:{random.randint(0,59):02d}:00+07:00",
            "Đồng ý hủy"
        ))

    elif status == "PENDING_CANCEL_L1":
        actions.append(_approval_sql(
            deal_id, "CANCEL_REQUEST", "OPEN", "PENDING_CANCEL_L1",
            DEALER_USER_ID, f"{base_time}10:{random.randint(0,59):02d}:00+07:00",
            "Yêu cầu hủy — sai thông tin giao dịch"
        ))

    return actions


def _approval_sql(deal_id, action_type, before, after, user_id, performed_at, reason):
    action_id = det_uuid()
    return (
        f"INSERT INTO approval_actions (id, deal_module, deal_id, action_type, "
        f"status_before, status_after, performed_by, performed_at, reason)\n"
        f"VALUES ('{action_id}', 'BOND', '{deal_id}', '{action_type}', "
        f"'{before}', '{after}', '{user_id}', '{performed_at}', '{sql_escape(reason)}')\n"
        f"ON CONFLICT DO NOTHING;"
    )


def _generate_inventory(inventory_map: dict) -> list[str]:
    """Generate bond_inventory records for COMPLETED BUY deals."""
    inserts = []
    for (bond_code, category, portfolio), qty in inventory_map.items():
        inv_id = det_uuid()
        # Find catalog_id if GOVERNMENT
        catalog_id = None
        if category == "GOVERNMENT":
            for cat in BOND_CATALOG:
                if cat[1] == bond_code:
                    catalog_id = cat[0]
                    break

        catalog_col = ", bond_catalog_id" if catalog_id else ""
        catalog_val = f", '{catalog_id}'" if catalog_id else ""

        # Add extra quantity so SELL deals can pass inventory check
        available = qty + random.randint(1000, 50000)

        inserts.append(
            f"INSERT INTO bond_inventory (id, bond_code, bond_category, portfolio_type, "
            f"available_quantity, acquisition_date, acquisition_price, updated_by{catalog_col})\n"
            f"VALUES ('{inv_id}', '{bond_code}', '{category}', '{portfolio}', "
            f"{available}, '2026-03-20', 100000, '{DEALER_USER_ID}'{catalog_val})\n"
            f"ON CONFLICT (bond_code, bond_category, portfolio_type) DO NOTHING;"
        )
    return inserts


# ─── Write Output & Execute ─────────────────────────────────────────────────

def main():
    deal_inserts, inventory_inserts, approval_inserts = generate()
    catalog_inserts = generate_bond_catalog()

    os.makedirs(os.path.dirname(OUTPUT_FILE), exist_ok=True)

    # Write SQL file
    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        f.write("-- ============================================================================\n")
        f.write("-- 003_bond_deals_seed.sql — 15 bond catalog + 60 bond deals + inventory\n")
        f.write("-- Generated by: scripts/generate_bond_seed.py\n")
        f.write("-- ============================================================================\n\n")
        f.write("BEGIN;\n\n")

        # Bond Catalog
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- Bond Catalog — 15 Trái phiếu Chính phủ (TPCP) Việt Nam\n")
        f.write("-- ---------------------------------------------------------------------------\n\n")
        for sql in catalog_inserts:
            f.write(f"{sql}\n\n")

        # Bond Inventory (before deals so SELL FK checks pass)
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- Bond Inventory — Tồn kho trái phiếu cho deals COMPLETED BUY\n")
        f.write("-- ---------------------------------------------------------------------------\n\n")
        for sql in inventory_inserts:
            f.write(f"{sql}\n\n")

        # Bond Deals
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- Bond Deals (60 total: 30 GOVERNMENT, 15 FI, 15 CCTG)\n")
        f.write("-- Status: 10 OPEN, 8 PENDING_L2, 8 PENDING_BOOKING,\n")
        f.write("--         5 PENDING_CHIEF_ACC, 15 COMPLETED, 5 REJECTED,\n")
        f.write("--         4 CANCELLED, 3 VOIDED, 2 PENDING_CANCEL_L1\n")
        f.write("-- ---------------------------------------------------------------------------\n\n")
        for sql in deal_inserts:
            f.write(f"{sql}\n\n")

        # Approval Actions
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- Approval Actions — Lịch sử phê duyệt cho Bond deals\n")
        f.write("-- ---------------------------------------------------------------------------\n\n")
        for sql in approval_inserts:
            f.write(f"{sql}\n\n")

        f.write("COMMIT;\n")

    total = len(catalog_inserts) + len(deal_inserts) + len(inventory_inserts) + len(approval_inserts)
    print(f"✅ Generated SQL file: {OUTPUT_FILE}")
    print(f"   📦 {len(catalog_inserts)} bond catalog entries")
    print(f"   📦 {len(deal_inserts)} bond deals")
    print(f"   📦 {len(inventory_inserts)} inventory records")
    print(f"   📦 {len(approval_inserts)} approval actions")
    print(f"   📦 Total: {total} INSERT statements")

    # Count stats
    cat_counts = {}
    status_counts = {}
    dir_counts = {}
    tx_counts = {}
    for sql in deal_inserts:
        first_line = sql.split("\n")[0]
        for cat in ["GOVERNMENT", "FINANCIAL_INSTITUTION", "CERTIFICATE_OF_DEPOSIT"]:
            if cat in first_line:
                cat_counts[cat] = cat_counts.get(cat, 0) + 1
                break
        for d in ["BUY", "SELL"]:
            if d in first_line:
                dir_counts[d] = dir_counts.get(d, 0) + 1
                break
        for s in ["OPEN", "PENDING_L2_APPROVAL", "PENDING_BOOKING", "PENDING_CHIEF_ACCOUNTANT",
                   "COMPLETED", "REJECTED", "CANCELLED", "VOIDED_BY_ACCOUNTING", "PENDING_CANCEL_L1"]:
            if s in first_line:
                status_counts[s] = status_counts.get(s, 0) + 1
                break
        for t in ["REVERSE_REPO", "OUTRIGHT", "REPO", "OTHER"]:
            if t in first_line:
                tx_counts[t] = tx_counts.get(t, 0) + 1
                break

    print(f"\n📊 Categories: {cat_counts}")
    print(f"📊 Statuses: {status_counts}")
    print(f"📊 Directions: {dir_counts}")
    print(f"📊 Transaction types: {tx_counts}")

    # Execute against database
    print(f"\n🔄 Executing against database: {DB_URL}")
    try:
        with open(OUTPUT_FILE, "r", encoding="utf-8") as f:
            sql_content = f.read()

        conn = psycopg2.connect(DB_URL)
        conn.autocommit = False
        cur = conn.cursor()
        cur.execute(sql_content)
        conn.commit()
        cur.close()
        conn.close()
        print("✅ Database execution successful!")
    except Exception as e:
        print(f"❌ Database execution failed: {e}")
        raise


if __name__ == "__main__":
    main()
