"""
Generate 50 realistic FX deals as SQL INSERT statements.
Output: migrations/seed/002_fx_deals_seed.sql
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

# ─── Configuration ───────────────────────────────────────────────────────────

NUM_DEALS = 50
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
API_DIR = os.path.dirname(SCRIPT_DIR)  # app/services/api/
OUTPUT_FILE = os.path.join(API_DIR, "migrations", "seed", "002_fx_deals_seed.sql")

# Existing seed IDs (from 001_seed.sql)
BRANCH_ID = "a0000000-0000-0000-0000-000000000001"  # Head Office
DEALER_USER_ID = "d0000000-0000-0000-0000-000000000001"  # dealer01

COUNTERPARTY_IDS = [
    f"e0000000-0000-0000-0000-00000000000{i}" if i < 10
    else "e0000000-0000-0000-0000-00000000000a"
    for i in range(1, 11)
]

# SSI IDs
SSI_INTERNAL_USD = "20000000-0000-0000-0000-000000000001"
SSI_INTERNAL_VND = "20000000-0000-0000-0000-000000000002"
SSI_COUNTERPARTY = {
    "e0000000-0000-0000-0000-000000000001": "20000000-0000-0000-0000-000000000003",  # MSB USD
    "e0000000-0000-0000-0000-000000000002": "20000000-0000-0000-0000-000000000004",  # ACB USD
    "e0000000-0000-0000-0000-000000000003": "20000000-0000-0000-0000-000000000005",  # VCB VND
}
# For counterparties without explicit SSI in seed, we'll create them in the SQL
EXTRA_SSI_BASE = "20000000-0000-0000-0000-0000000000"  # + 06..0f

# ─── Deal Distribution ──────────────────────────────────────────────────────

# 25 SPOT, 15 FORWARD, 10 SWAP
DEAL_TYPE_LIST = ["SPOT"] * 25 + ["FORWARD"] * 15 + ["SWAP"] * 10

# 10 OPEN, 8 PENDING_L2_APPROVAL, 8 PENDING_BOOKING, 12 COMPLETED, 6 REJECTED, 6 CANCELLED
STATUS_LIST = (
    ["OPEN"] * 10
    + ["PENDING_L2_APPROVAL"] * 8
    + ["PENDING_BOOKING"] * 8
    + ["COMPLETED"] * 12
    + ["REJECTED"] * 6
    + ["CANCELLED"] * 6
)

DIRECTIONS = {
    "SPOT": ["BUY", "SELL"],
    "FORWARD": ["BUY", "SELL"],
    "SWAP": ["BUY_SELL", "SELL_BUY"],
}

# Currency pair config with proper precision
PAIR_CONFIG = {
    "USD/VND": {
        "base": "USD",
        "quote": "VND",
        "converted_ccy": "VND",
        "rate_min": 25800.00,
        "rate_max": 26200.00,
        "rate_dp": 2,       # 2 decimal places for USD/VND
        "calc": "MULTIPLY",
    },
    "EUR/USD": {
        "base": "EUR",
        "quote": "USD",
        "converted_ccy": "USD",
        "rate_min": 1.0800,
        "rate_max": 1.1200,
        "rate_dp": 6,       # 6 decimal places for EUR/USD
        "calc": "MULTIPLY",
    },
}

# ─── Helpers ─────────────────────────────────────────────────────────────────

fake = Faker()
random.seed(42)
Faker.seed(42)


def new_uuid() -> str:
    return str(uuid.uuid4())


def rand_rate(pair: dict) -> Decimal:
    """Generate a random rate with proper decimal precision."""
    dp = pair["rate_dp"]
    raw = random.uniform(pair["rate_min"], pair["rate_max"])
    quant = Decimal(10) ** -dp
    return Decimal(str(raw)).quantize(quant, rounding=ROUND_HALF_UP)


def rand_amount() -> Decimal:
    """Generate realistic notional: 10k to 5M in steps of 10k."""
    return Decimal(random.randint(1, 500)) * Decimal("10000.00")


def deal_number(idx: int) -> str:
    """FX-20260401-0001 through FX-20260403-0050, spread across 3 days."""
    if idx < 20:
        day = "20260401"
        seq = idx + 1
    elif idx < 35:
        day = "20260402"
        seq = idx - 20 + 1 + 20  # continue numbering
    else:
        day = "20260403"
        seq = idx - 35 + 1 + 35
    return f"FX-{day}-{idx + 1:04d}"


def trade_date_for(idx: int) -> date:
    if idx < 20:
        return date(2026, 4, 1)
    elif idx < 35:
        return date(2026, 4, 2)
    else:
        return date(2026, 4, 3)


def get_internal_ssi(pair_code: str) -> str:
    """Internal SSI based on the base currency."""
    if pair_code == "USD/VND":
        return SSI_INTERNAL_USD
    return SSI_INTERNAL_USD  # We only have USD and VND internal SSIs


def get_counterparty_ssi(cp_id: str) -> str:
    """Get counterparty SSI. Use existing or extra ones."""
    if cp_id in SSI_COUNTERPARTY:
        return SSI_COUNTERPARTY[cp_id]
    # Map remaining counterparties to extra SSI IDs
    idx = COUNTERPARTY_IDS.index(cp_id)
    return f"{EXTRA_SSI_BASE}{idx + 6:02x}"


def sql_escape(val: str) -> str:
    return val.replace("'", "''")


# ─── Generate Extra SSIs ────────────────────────────────────────────────────

def generate_extra_ssis() -> list[str]:
    """Generate SSI inserts for counterparties that don't have one in 001_seed."""
    lines = []
    bank_names = {
        "e0000000-0000-0000-0000-000000000004": ("TCB", "Techcombank", "VTCBVNVX"),
        "e0000000-0000-0000-0000-000000000005": ("VPB", "VPBank", "VPBKVNVX"),
        "e0000000-0000-0000-0000-000000000006": ("MBB", "MB Bank", "MSCBVNVX"),
        "e0000000-0000-0000-0000-000000000007": ("BID", "BIDV", "BIDVVNVX"),
        "e0000000-0000-0000-0000-000000000008": ("CTG", "VietinBank", "ICBVVNVX"),
        "e0000000-0000-0000-0000-000000000009": ("STB", "Sacombank", "SGTTVNVX"),
        "e0000000-0000-0000-0000-00000000000a": ("SHB", "SHB", "SHBAVNVX"),
    }
    for cp_id, (code, name, swift) in bank_names.items():
        ssi_id = get_counterparty_ssi(cp_id)
        lines.append(
            f"INSERT INTO settlement_instructions "
            f"(id, counterparty_id, currency_code, owner_type, account_number, bank_name, swift_code, is_default) "
            f"VALUES ('{ssi_id}', '{cp_id}', 'USD', 'COUNTERPARTY', '2001-USD-{code}', '{name}', '{swift}', true);"
        )
    return lines


# ─── Main Generation ────────────────────────────────────────────────────────

def generate():
    # Shuffle deal types and statuses together to get a good mix
    combined = list(zip(DEAL_TYPE_LIST, STATUS_LIST))
    random.shuffle(combined)
    deal_types, statuses = zip(*combined)

    deal_inserts = []
    leg_inserts = []

    for i in range(NUM_DEALS):
        deal_id = new_uuid()
        deal_type = deal_types[i]
        status = statuses[i]
        direction = random.choice(DIRECTIONS[deal_type])
        pair_code = random.choice(list(PAIR_CONFIG.keys()))
        pair = PAIR_CONFIG[pair_code]
        cp_id = random.choice(COUNTERPARTY_IDS)
        td = trade_date_for(i)
        notional = rand_amount()
        dn = deal_number(i)

        # Cancel reason for REJECTED/CANCELLED
        cancel_reason = None
        cancel_requested_by = None
        cancel_requested_at = None
        if status == "REJECTED":
            cancel_reason = random.choice([
                "Rate không hợp lý",
                "Vượt hạn mức tín dụng",
                "Thiếu hồ sơ đối tác",
                "Không đủ điều kiện giao dịch",
            ])
        elif status == "CANCELLED":
            cancel_reason = random.choice([
                "Khách hàng yêu cầu hủy",
                "Thay đổi điều kiện thị trường",
                "Deal trùng lặp",
            ])
            cancel_requested_by = DEALER_USER_ID
            cancel_requested_at = f"{td}T10:00:00+07:00"

        # Build INSERT
        cols = [
            "id", "deal_number", "counterparty_id", "deal_type", "direction",
            "notional_amount", "currency_code", "pair_code", "trade_date",
            "branch_id", "status", "created_by", "updated_by", "created_at", "updated_at",
        ]
        vals = [
            f"'{deal_id}'",
            f"'{dn}'",
            f"'{cp_id}'",
            f"'{deal_type}'",
            f"'{direction}'",
            f"{notional:.2f}",
            f"'{pair['base']}'",
            f"'{pair_code}'",
            f"'{td.isoformat()}'",
            f"'{BRANCH_ID}'",
            f"'{status}'",
            f"'{DEALER_USER_ID}'",
            f"'{DEALER_USER_ID}'",
            f"'{td.isoformat()}T09:{random.randint(0,59):02d}:{random.randint(0,59):02d}+07:00'",
            f"'{td.isoformat()}T09:{random.randint(0,59):02d}:{random.randint(0,59):02d}+07:00'",
        ]

        if cancel_reason:
            cols.append("cancel_reason")
            vals.append(f"'{sql_escape(cancel_reason)}'")
        if cancel_requested_by:
            cols.append("cancel_requested_by")
            vals.append(f"'{cancel_requested_by}'")
            cols.append("cancel_requested_at")
            vals.append(f"'{cancel_requested_at}'")

        deal_inserts.append(
            f"-- Deal {i+1}: {dn} | {deal_type} {direction} {pair_code} | {status}\n"
            f"INSERT INTO fx_deals ({', '.join(cols)})\n"
            f"VALUES ({', '.join(vals)});"
        )

        # ─── Legs ────────────────────────────────────────────────────
        internal_ssi = get_internal_ssi(pair_code)
        cp_ssi = get_counterparty_ssi(cp_id)

        # Leg 1 (near leg)
        if deal_type == "SPOT":
            vd1 = td + timedelta(days=2)
        elif deal_type == "FORWARD":
            vd1 = td + timedelta(days=random.choice([30, 60, 90, 180]))
        else:  # SWAP near leg
            vd1 = td + timedelta(days=2)

        rate1 = rand_rate(pair)
        converted1 = (notional * rate1).quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)

        leg_inserts.append(
            f"INSERT INTO fx_deal_legs (id, deal_id, leg_number, value_date, settlement_date, "
            f"exchange_rate, converted_amount, converted_currency, internal_ssi_id, counterparty_ssi_id)\n"
            f"VALUES ('{new_uuid()}', '{deal_id}', 1, '{vd1.isoformat()}', '{vd1.isoformat()}', "
            f"{rate1}, {converted1:.2f}, '{pair['converted_ccy']}', '{internal_ssi}', '{cp_ssi}');"
        )

        # Leg 2 (far leg for SWAP only)
        if deal_type == "SWAP":
            vd2 = td + timedelta(days=random.choice([30, 60, 90, 180, 270, 365]))
            # Forward points: slight rate adjustment
            if pair_code == "USD/VND":
                fwd_points = Decimal(str(random.uniform(-50, 150))).quantize(Decimal("0.01"))
            else:
                fwd_points = Decimal(str(random.uniform(-0.005, 0.015))).quantize(
                    Decimal("0.000001")
                )
            rate2 = rate1 + fwd_points
            converted2 = (notional * rate2).quantize(Decimal("0.01"), rounding=ROUND_HALF_UP)

            leg_inserts.append(
                f"INSERT INTO fx_deal_legs (id, deal_id, leg_number, value_date, settlement_date, "
                f"exchange_rate, converted_amount, converted_currency, internal_ssi_id, counterparty_ssi_id)\n"
                f"VALUES ('{new_uuid()}', '{deal_id}', 2, '{vd2.isoformat()}', '{vd2.isoformat()}', "
                f"{rate2}, {converted2:.2f}, '{pair['converted_ccy']}', '{internal_ssi}', '{cp_ssi}');"
            )

    return deal_inserts, leg_inserts


# ─── Write Output ────────────────────────────────────────────────────────────

def main():
    deal_inserts, leg_inserts = generate()
    extra_ssis = generate_extra_ssis()

    os.makedirs(os.path.dirname(OUTPUT_FILE), exist_ok=True)

    with open(OUTPUT_FILE, "w", encoding="utf-8") as f:
        f.write("-- ============================================================================\n")
        f.write("-- 002_fx_deals_seed.sql — 50 realistic FX deals for testing\n")
        f.write("-- Generated by: scripts/generate_seed_data.py\n")
        f.write("-- ============================================================================\n\n")
        f.write("BEGIN;\n\n")

        # Extra SSIs for counterparties not covered in 001_seed
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- Additional Settlement Instructions for counterparties\n")
        f.write("-- ---------------------------------------------------------------------------\n")
        for ssi in extra_ssis:
            f.write(f"{ssi}\n")
        f.write("\n")

        # FX Deals
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- FX Deals (50 total: 25 SPOT, 15 FORWARD, 10 SWAP)\n")
        f.write("-- Status mix: 10 OPEN, 8 PENDING_L2, 8 PENDING_BOOKING,\n")
        f.write("--             12 COMPLETED, 6 REJECTED, 6 CANCELLED\n")
        f.write("-- ---------------------------------------------------------------------------\n\n")
        for sql in deal_inserts:
            f.write(f"{sql}\n\n")

        # FX Deal Legs
        f.write("-- ---------------------------------------------------------------------------\n")
        f.write("-- FX Deal Legs (1 per SPOT/FORWARD, 2 per SWAP)\n")
        f.write("-- ---------------------------------------------------------------------------\n\n")
        for sql in leg_inserts:
            f.write(f"{sql}\n\n")

        f.write("COMMIT;\n")

    total_legs = len(leg_inserts)
    print(f"✅ Generated {len(deal_inserts)} deals + {total_legs} legs")
    print(f"📄 Output: {OUTPUT_FILE}")

    # Verify counts
    type_counts = {"SPOT": 0, "FORWARD": 0, "SWAP": 0}
    status_counts = {}
    for sql in deal_inserts:
        for t in type_counts:
            if f"'{t}'" in sql.split("\n")[0]:
                type_counts[t] += 1
                break
        for line in sql.split("\n"):
            if "VALUES" in line:
                for s in ["OPEN", "PENDING_L2_APPROVAL", "PENDING_BOOKING", "COMPLETED", "REJECTED", "CANCELLED"]:
                    if f"'{s}'" in line:
                        status_counts[s] = status_counts.get(s, 0) + 1
                        break

    print(f"\n📊 Deal types: {type_counts}")
    print(f"📊 Statuses: {status_counts}")
    print(f"📊 Legs: {total_legs} (expected: {50 + 10} = 60 for 10 SWAPs)")


if __name__ == "__main__":
    main()
