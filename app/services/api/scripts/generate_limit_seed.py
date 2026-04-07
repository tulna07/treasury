"""
Generate 20 credit limit records (10 counterparties × 2 limit types).
Output: migrations/seed/004_credit_limits_seed.sql
Also executes directly against the database.
"""

import os
import random
import uuid
from datetime import date, timedelta
from decimal import Decimal

try:
    import psycopg2
except ImportError:
    raise SystemExit("psycopg2 not installed. Run: pip install psycopg2-binary")

# ─── Configuration ───────────────────────────────────────────────────────────

SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
API_DIR = os.path.dirname(SCRIPT_DIR)
OUTPUT_FILE = os.path.join(API_DIR, "migrations", "seed", "004_credit_limits_seed.sql")
DB_URL = "postgresql://mrm@localhost:5432/treasury"

DEALER_USER_ID = "d0000000-0000-0000-0000-000000000001"
DIVHEAD_USER_ID = "d0000000-0000-0000-0000-000000000008"

COUNTERPARTY_IDS = [
    f"e0000000-0000-0000-0000-00000000000{i}" for i in range(1, 10)
] + ["e0000000-0000-0000-0000-000000000010"]

COUNTERPARTY_NAMES = {
    "e0000000-0000-0000-0000-000000000001": "MSB",
    "e0000000-0000-0000-0000-000000000002": "ACB",
    "e0000000-0000-0000-0000-000000000003": "VCB",
    "e0000000-0000-0000-0000-000000000004": "TCB",
    "e0000000-0000-0000-0000-000000000005": "VPB",
    "e0000000-0000-0000-0000-000000000006": "MBB",
    "e0000000-0000-0000-0000-000000000007": "BIDV",
    "e0000000-0000-0000-0000-000000000008": "VietinBank",
    "e0000000-0000-0000-0000-000000000009": "Sacombank",
    "e0000000-0000-0000-0000-000000000010": "SHB",
}

LIMIT_TYPES = ["COLLATERALIZED", "UNCOLLATERALIZED"]

# VND amounts in billions (200B to 2000B range)
LIMIT_AMOUNTS_BN = [200, 300, 400, 500, 600, 700, 800, 1000, 1200, 1500, 2000]

# Counterparties that get unlimited limits (indices into COUNTERPARTY_IDS)
# VCB and BIDV are large state banks → unlimited collateralized
UNLIMITED_COLLATERALIZED = {
    "e0000000-0000-0000-0000-000000000003",  # VCB
    "e0000000-0000-0000-0000-000000000007",  # BIDV
}
UNLIMITED_UNCOLLATERALIZED = {
    "e0000000-0000-0000-0000-000000000003",  # VCB
}


def generate_uuid_for_limit(cp_idx: int, type_idx: int) -> str:
    """Deterministic UUID: SEED-CL-{cp_idx}{type_idx}"""
    return f"c1{cp_idx:02d}{type_idx}000-0000-0000-0000-000000000001"


def generate_limits():
    """Generate 20 credit limit records."""
    records = []
    random.seed(42)
    effective_from = date(2026, 1, 1)
    expiry_date = date(2026, 12, 31)

    for cp_idx, cp_id in enumerate(COUNTERPARTY_IDS):
        for type_idx, limit_type in enumerate(LIMIT_TYPES):
            limit_id = str(uuid.uuid4())
            is_unlimited = False
            limit_amount = None

            if limit_type == "COLLATERALIZED" and cp_id in UNLIMITED_COLLATERALIZED:
                is_unlimited = True
            elif limit_type == "UNCOLLATERALIZED" and cp_id in UNLIMITED_UNCOLLATERALIZED:
                is_unlimited = True
            else:
                # Pick a random amount in billions
                bn = random.choice(LIMIT_AMOUNTS_BN)
                limit_amount = Decimal(bn) * Decimal("1000000000")

            approval_ref = f"QD-HDQT-{2026}-{cp_idx * 2 + type_idx + 1:03d}"
            cp_name = COUNTERPARTY_NAMES[cp_id]

            records.append({
                "id": limit_id,
                "counterparty_id": cp_id,
                "limit_type": limit_type,
                "limit_amount": limit_amount,
                "is_unlimited": is_unlimited,
                "effective_from": effective_from,
                "expiry_date": expiry_date,
                "is_current": True,
                "approval_reference": approval_ref,
                "created_by": DIVHEAD_USER_ID,
                "updated_by": DIVHEAD_USER_ID,
                "cp_name": cp_name,
            })

    return records


def generate_sql(records):
    """Generate SQL INSERT statements."""
    lines = [
        "-- ============================================================================",
        "-- 004_credit_limits_seed.sql — Credit limit seed data (10 counterparties × 2 types)",
        "-- ============================================================================",
        "",
        "-- 20 credit limits: some unlimited (VCB, BIDV), rest with amounts (200B-2000B VND)",
        "",
        "INSERT INTO credit_limits (",
        "    id, counterparty_id, limit_type, limit_amount, is_unlimited,",
        "    effective_from, effective_to, is_current, expiry_date,",
        "    approval_reference, created_by, updated_by",
        ") VALUES",
    ]

    value_lines = []
    for rec in records:
        amt = "NULL" if rec["limit_amount"] is None else str(rec["limit_amount"])
        is_unl = "true" if rec["is_unlimited"] else "false"
        line = (
            f"    ('{rec['id']}', '{rec['counterparty_id']}', '{rec['limit_type']}', "
            f"{amt}, {is_unl}, "
            f"'{rec['effective_from']}', NULL, true, '{rec['expiry_date']}', "
            f"'{rec['approval_reference']}', '{rec['created_by']}', '{rec['updated_by']}')"
        )
        value_lines.append(line)

    lines.append(",\n".join(value_lines) + "")
    lines.append("ON CONFLICT DO NOTHING;")
    lines.append("")

    return "\n".join(lines)


def execute_sql(records):
    """Execute directly against the database."""
    conn = psycopg2.connect(DB_URL)
    cur = conn.cursor()

    # Clean existing seed limits
    cur.execute("DELETE FROM credit_limits WHERE approval_reference LIKE 'QD-HDQT-2026-%'")

    for rec in records:
        cur.execute(
            """
            INSERT INTO credit_limits (
                id, counterparty_id, limit_type, limit_amount, is_unlimited,
                effective_from, effective_to, is_current, expiry_date,
                approval_reference, created_by, updated_by
            ) VALUES (%s, %s, %s, %s, %s, %s, NULL, true, %s, %s, %s, %s)
            ON CONFLICT DO NOTHING
            """,
            (
                rec["id"],
                rec["counterparty_id"],
                rec["limit_type"],
                rec["limit_amount"],
                rec["is_unlimited"],
                rec["effective_from"],
                rec["expiry_date"],
                rec["approval_reference"],
                rec["created_by"],
                rec["updated_by"],
            ),
        )

    conn.commit()
    count = cur.rowcount
    cur.close()
    conn.close()
    return count


def main():
    records = generate_limits()
    sql = generate_sql(records)

    # Write SQL file
    os.makedirs(os.path.dirname(OUTPUT_FILE), exist_ok=True)
    with open(OUTPUT_FILE, "w") as f:
        f.write(sql)
    print(f"✓ Written {len(records)} credit limits to {OUTPUT_FILE}")

    # Print summary
    unlimited_count = sum(1 for r in records if r["is_unlimited"])
    with_amount = len(records) - unlimited_count
    print(f"  → {unlimited_count} unlimited, {with_amount} with amounts")
    for rec in records:
        amt_str = "UNLIMITED" if rec["is_unlimited"] else f"{rec['limit_amount'] / Decimal('1000000000'):.0f}B VND"
        print(f"    {rec['cp_name']:12s} {rec['limit_type']:18s} {amt_str}")

    # Execute against DB
    try:
        execute_sql(records)
        print(f"✓ Executed {len(records)} records against database")
    except Exception as e:
        print(f"✗ DB execution failed: {e}")
        print("  (SQL file was still written successfully)")


if __name__ == "__main__":
    main()
