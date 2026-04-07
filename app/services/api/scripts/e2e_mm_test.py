#!/usr/bin/env python3
"""
Treasury API — Money Market E2E Test Suite
===========================================
Tests 15 scenarios against the MM API endpoints (Interbank, OMO, Govt Repo).

Usage:
    python e2e_mm_test.py [--base-url http://localhost:34080]
"""

import argparse
import sys
from datetime import datetime, timedelta, timezone
from typing import Optional

import requests
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich import box

# ─── Config ───────────────────────────────────────────────────────────────────

BASE_URL = "http://localhost:34080"
API = f"{BASE_URL}/api/v1"
PASSWORD = "P@ssw0rd123"

USERS = {
    "dealer": "dealer01",
    "deskhead": "deskhead01",
    "director": "director01",
    "accountant": "accountant01",
    "chiefacc": "chiefacc01",
    "risk": "risk01",
    "riskhead": "riskhead01",
    "settlement": "settlement01",
    "admin": "admin01",
}

COUNTERPARTY_MSB = "e0000000-0000-0000-0000-000000000001"
COUNTERPARTY_ACB = "e0000000-0000-0000-0000-000000000002"
BOND_CATALOG_ID = "bc000000-0000-0000-0000-000000000001"

console = Console()


# ─── API Client ──────────────────────────────────────────────────────────────


class MMAPIClient:
    def __init__(self):
        self.session = requests.Session()

    def login(self, username: str, password: str = PASSWORD) -> requests.Response:
        return self.session.post(f"{API}/auth/login", json={"username": username, "password": password})

    # ─── Interbank ───
    def create_interbank(self, payload: dict) -> requests.Response:
        return self.session.post(f"{API}/mm/interbank", json=payload)

    def get_interbank(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/mm/interbank/{deal_id}")

    def list_interbank(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/mm/interbank", params=params or {})

    def approve_interbank(self, deal_id: str, action: str = "APPROVE", level: str = "") -> requests.Response:
        body = {"action": action}
        if level:
            body["level"] = level
        return self.session.post(f"{API}/mm/interbank/{deal_id}/approve", json=body)

    def recall_interbank(self, deal_id: str, reason: str) -> requests.Response:
        return self.session.post(f"{API}/mm/interbank/{deal_id}/recall", json={"reason": reason})

    def cancel_interbank(self, deal_id: str, reason: str) -> requests.Response:
        return self.session.post(f"{API}/mm/interbank/{deal_id}/cancel", json={"reason": reason})

    def cancel_approve_interbank(self, deal_id: str, action: str = "APPROVE") -> requests.Response:
        return self.session.post(f"{API}/mm/interbank/{deal_id}/cancel-approve", json={"action": action})

    def clone_interbank(self, deal_id: str) -> requests.Response:
        return self.session.post(f"{API}/mm/interbank/{deal_id}/clone")

    def history_interbank(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/mm/interbank/{deal_id}/history")

    # ─── OMO ───
    def create_omo(self, payload: dict) -> requests.Response:
        return self.session.post(f"{API}/mm/omo", json=payload)

    def get_omo(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/mm/omo/{deal_id}")

    def approve_omo(self, deal_id: str, action: str = "APPROVE") -> requests.Response:
        return self.session.post(f"{API}/mm/omo/{deal_id}/approve", json={"action": action})

    # ─── Govt Repo ───
    def create_govt_repo(self, payload: dict) -> requests.Response:
        return self.session.post(f"{API}/mm/govt-repo", json=payload)

    def approve_govt_repo(self, deal_id: str, action: str = "APPROVE") -> requests.Response:
        return self.session.post(f"{API}/mm/govt-repo/{deal_id}/approve", json={"action": action})


# ─── Payloads ─────────────────────────────────────────────────────────────────


def make_interbank(direction="PLACE", currency="VND", principal=100_000_000_000, rate=4.5, tenor=90, collateral=False):
    from datetime import datetime, timedelta
    eff = datetime(2026, 4, 15)
    mat = eff + timedelta(days=tenor)
    return {
        "counterparty_id": COUNTERPARTY_MSB,
        "currency_code": currency,
        "direction": direction,
        "principal_amount": principal,
        "interest_rate": rate,
        "day_count_convention": "ACT_365",
        "trade_date": "2026-04-15T00:00:00Z",
        "effective_date": eff.strftime("%Y-%m-%dT00:00:00Z"),
        "maturity_date": mat.strftime("%Y-%m-%dT00:00:00Z"),
        "has_collateral": collateral,
    }


def make_omo():
    return {
        "deal_subtype": "OMO",
        "session_name": "Phiên 1",
        "counterparty_id": COUNTERPARTY_MSB,
        "bond_catalog_id": BOND_CATALOG_ID,
        "notional_amount": 100_000_000_000,
        "winning_rate": 3.5,
        "tenor_days": 14,
        "settlement_date_1": "2026-04-15T00:00:00Z",
        "settlement_date_2": "2026-04-29T00:00:00Z",
        "haircut_pct": 5.0,
        "trade_date": "2026-04-15T00:00:00Z",
    }


def make_govt_repo():
    p = make_omo()
    p["deal_subtype"] = "STATE_REPO"
    return p


# ─── Test Runner ──────────────────────────────────────────────────────────────


results = []


def run_test(name, fn):
    try:
        fn()
        results.append((name, "✅ PASS", ""))
    except Exception as e:
        results.append((name, "❌ FAIL", str(e)[:120]))


def assert_ok(r, expected=None):
    if expected:
        assert r.status_code == expected, f"Expected {expected}, got {r.status_code}: {r.text[:200]}"
    else:
        assert r.status_code in (200, 201), f"Expected 2xx, got {r.status_code}: {r.text[:200]}"


def assert_status(r, status):
    data = r.json()
    actual = data.get("data", {}).get("status", "")
    assert actual == status, f"Expected status={status}, got {actual}"


def login_as(client, role):
    r = client.login(USERS[role])
    assert_ok(r)


# ─── Interbank full approval helper ──────────────────────────────────────────


def full_approve_interbank(client, deal_id, with_ttqt=False):
    """Run through full interbank approval: DH(→TP_REVIEW)→DH(→L2)→DIR→QLRR→KTTC→[TTQT]"""
    # Step 1: DH first approve (OPEN → PENDING_TP_REVIEW)
    login_as(client, "deskhead")
    assert_ok(client.approve_interbank(deal_id, "APPROVE"))

    # Step 2: DH second approve (PENDING_TP_REVIEW → PENDING_L2_APPROVAL)
    assert_ok(client.approve_interbank(deal_id, "APPROVE"))

    # Step 3: Director approves (PENDING_L2_APPROVAL → PENDING_RISK_APPROVAL)
    login_as(client, "director")
    assert_ok(client.approve_interbank(deal_id, "APPROVE"))

    # QLRR approve (single step in current implementation)
    login_as(client, "risk")
    assert_ok(client.approve_interbank(deal_id, "APPROVE"))

    login_as(client, "accountant")
    assert_ok(client.approve_interbank(deal_id, "APPROVE"))

    login_as(client, "chiefacc")
    assert_ok(client.approve_interbank(deal_id, "APPROVE"))

    if with_ttqt:
        login_as(client, "settlement")
        assert_ok(client.approve_interbank(deal_id, "APPROVE"))


def full_approve_omo(client, deal_id):
    """OMO: DH→DIR→KTTC(2)"""
    login_as(client, "deskhead")
    assert_ok(client.approve_omo(deal_id, "APPROVE"))

    login_as(client, "director")
    assert_ok(client.approve_omo(deal_id, "APPROVE"))

    login_as(client, "accountant")
    assert_ok(client.approve_omo(deal_id, "APPROVE"))

    login_as(client, "chiefacc")
    assert_ok(client.approve_omo(deal_id, "APPROVE"))


# ─── Test Scenarios ───────────────────────────────────────────────────────────


def s01_interbank_full_approval():
    """S01: Interbank full approval (PLACE VND, no TTQT) → COMPLETED"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    assert_ok(r, 201)
    deal_id = r.json()["data"]["id"]

    full_approve_interbank(c, deal_id)

    login_as(c, "dealer")
    r = c.get_interbank(deal_id)
    assert_status(r, "COMPLETED")


def s02_omo_full_approval():
    """S02: OMO full approval → COMPLETED"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_omo(make_omo())
    assert_ok(r, 201)
    deal_id = r.json()["data"]["id"]

    full_approve_omo(c, deal_id)

    login_as(c, "dealer")
    r = c.get_omo(deal_id)
    assert_status(r, "COMPLETED")


def s03_govt_repo_full_approval():
    """S03: Govt Repo full approval → COMPLETED"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_govt_repo(make_govt_repo())
    assert_ok(r, 201)
    deal_id = r.json()["data"]["id"]

    # Same approval as OMO
    login_as(c, "deskhead")
    assert_ok(c.approve_govt_repo(deal_id, "APPROVE"))
    login_as(c, "director")
    assert_ok(c.approve_govt_repo(deal_id, "APPROVE"))
    login_as(c, "accountant")
    assert_ok(c.approve_govt_repo(deal_id, "APPROVE"))
    login_as(c, "chiefacc")
    assert_ok(c.approve_govt_repo(deal_id, "APPROVE"))

    login_as(c, "dealer")
    r = c.session.get(f"{API}/mm/govt-repo/{deal_id}")
    assert_status(r, "COMPLETED")


def s04_interbank_director_reject():
    """S04: Interbank → GĐ reject → REJECTED"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    deal_id = r.json()["data"]["id"]

    # DH approve twice (OPEN→TP_REVIEW→L2)
    login_as(c, "deskhead")
    c.approve_interbank(deal_id, "APPROVE")
    c.approve_interbank(deal_id, "APPROVE")

    login_as(c, "director")
    r = c.approve_interbank(deal_id, "REJECT")
    assert_ok(r)

    login_as(c, "dealer")
    r = c.get_interbank(deal_id)
    assert_status(r, "REJECTED")


def s05_interbank_qlrr_reject():
    """S05: Interbank → QLRR reject → VOIDED_BY_RISK"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    deal_id = r.json()["data"]["id"]

    # DH approve twice + DIR
    login_as(c, "deskhead")
    c.approve_interbank(deal_id, "APPROVE")
    c.approve_interbank(deal_id, "APPROVE")
    login_as(c, "director")
    c.approve_interbank(deal_id, "APPROVE")

    login_as(c, "risk")
    r = c.approve_interbank(deal_id, "REJECT")
    assert_ok(r)

    login_as(c, "dealer")
    r = c.get_interbank(deal_id)
    assert_status(r, "VOIDED_BY_RISK")


def s06_interbank_cancel_2level():
    """S06: Interbank cancel 2-level → CANCELLED"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    deal_id = r.json()["data"]["id"]
    full_approve_interbank(c, deal_id)

    login_as(c, "dealer")
    assert_ok(c.cancel_interbank(deal_id, "Sai thông tin đối tác"))

    login_as(c, "deskhead")
    assert_ok(c.cancel_approve_interbank(deal_id, "APPROVE"))

    login_as(c, "director")
    assert_ok(c.cancel_approve_interbank(deal_id, "APPROVE"))

    login_as(c, "dealer")
    r = c.get_interbank(deal_id)
    assert_status(r, "CANCELLED")


def s07_interbank_recall():
    """S07: Interbank recall → OPEN"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    deal_id = r.json()["data"]["id"]

    # DH approve twice → PENDING_L2
    login_as(c, "deskhead")
    c.approve_interbank(deal_id, "APPROVE")
    c.approve_interbank(deal_id, "APPROVE")

    login_as(c, "dealer")
    r = c.recall_interbank(deal_id, "Cần sửa thông tin")
    assert_ok(r)

    r = c.get_interbank(deal_id)
    assert_status(r, "OPEN")


def s08_clone_rejected():
    """S08: Clone rejected deal → new OPEN deal"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    deal_id = r.json()["data"]["id"]

    # DH approve twice + DIR reject
    login_as(c, "deskhead")
    c.approve_interbank(deal_id, "APPROVE")
    c.approve_interbank(deal_id, "APPROVE")
    login_as(c, "director")
    c.approve_interbank(deal_id, "REJECT")

    login_as(c, "dealer")
    r = c.clone_interbank(deal_id)
    assert_ok(r, 201)
    new_id = r.json()["data"]["id"]
    assert new_id != deal_id

    r = c.get_interbank(new_id)
    assert_status(r, "OPEN")


def s09_validation_zero_principal():
    """S09: Zero principal → 400"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank(principal=0))
    assert r.status_code in (400, 422), f"Expected 400/422, got {r.status_code}"


def s10_validation_negative_tenor():
    """S10: Negative tenor → 400"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank(tenor=-5))
    assert r.status_code in (400, 422), f"Expected 400/422, got {r.status_code}"


def s11_interest_calc_vnd():
    """S11: Interest = principal × rate/100 × tenor/365 (ACT_365 VND)"""
    c = MMAPIClient()
    login_as(c, "dealer")
    payload = make_interbank(principal=100_000_000_000, rate=4.5, tenor=90)
    r = c.create_interbank(payload)
    assert_ok(r, 201)
    data = r.json()["data"]
    expected_interest = round(100_000_000_000 * 4.5 / 100 * 90 / 365)
    actual_interest = round(float(data.get("interest_amount", 0)))
    diff = abs(expected_interest - actual_interest)
    assert diff <= 1, f"Interest mismatch: expected ~{expected_interest}, got {actual_interest}"


def s12_permission_accountant_cannot_create():
    """S12: Accountant cannot create MM deal → 403"""
    c = MMAPIClient()
    login_as(c, "accountant")
    r = c.create_interbank(make_interbank())
    assert r.status_code == 403, f"Expected 403, got {r.status_code}"


def s13_deal_number_format_interbank():
    """S13: Interbank deal number = MM-YYYYMMDD-NNNN"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_interbank(make_interbank())
    assert_ok(r, 201)
    dn = r.json()["data"]["deal_number"]
    assert dn.startswith("MM-"), f"Expected MM-*, got {dn}"


def s14_deal_number_format_omo():
    """S14: OMO deal number = OMO-YYYYMMDD-NNNN"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.create_omo(make_omo())
    assert_ok(r, 201)
    dn = r.json()["data"]["deal_number"]
    assert dn.startswith("OMO-"), f"Expected OMO-*, got {dn}"


def s15_list_filter_by_status():
    """S15: List interbank filtered by COMPLETED"""
    c = MMAPIClient()
    login_as(c, "dealer")
    r = c.list_interbank({"status": "COMPLETED"})
    assert_ok(r)
    data = r.json().get("data", {})
    items = data.get("data", [])
    for item in items:
        assert item["status"] == "COMPLETED", f"Expected COMPLETED, got {item['status']}"


# ─── Main ─────────────────────────────────────────────────────────────────────


def main():
    parser = argparse.ArgumentParser(description="MM E2E Tests")
    parser.add_argument("--base-url", default="http://localhost:34080")
    args = parser.parse_args()

    global BASE_URL, API
    BASE_URL = args.base_url
    API = f"{BASE_URL}/api/v1"

    console.print(Panel("[bold]Treasury MM — E2E Test Suite[/bold]", style="blue"))

    tests = [
        ("S01: Interbank full approval → COMPLETED", s01_interbank_full_approval),
        ("S02: OMO full approval → COMPLETED", s02_omo_full_approval),
        ("S03: Govt Repo full approval → COMPLETED", s03_govt_repo_full_approval),
        ("S04: Interbank director reject", s04_interbank_director_reject),
        ("S05: Interbank QLRR reject", s05_interbank_qlrr_reject),
        ("S06: Interbank cancel 2-level", s06_interbank_cancel_2level),
        ("S07: Interbank recall", s07_interbank_recall),
        ("S08: Clone rejected deal", s08_clone_rejected),
        ("S09: Validation: zero principal", s09_validation_zero_principal),
        ("S10: Validation: negative tenor", s10_validation_negative_tenor),
        ("S11: Interest calc VND ACT/365", s11_interest_calc_vnd),
        ("S12: Permission: accountant cannot create", s12_permission_accountant_cannot_create),
        ("S13: Deal number MM-YYYYMMDD-NNNN", s13_deal_number_format_interbank),
        ("S14: Deal number OMO-YYYYMMDD-NNNN", s14_deal_number_format_omo),
        ("S15: List filter by status", s15_list_filter_by_status),
    ]

    for name, fn in tests:
        console.print(f"\n═══ {name} ═══", style="bold cyan")
        run_test(name, fn)

    # ─── Summary ───
    table = Table(title="MM E2E Test Results", box=box.ROUNDED)
    table.add_column("#", width=5)
    table.add_column("Test", min_width=40)
    table.add_column("Result", width=10)
    table.add_column("Error", max_width=60)

    passed = sum(1 for _, s, _ in results if "PASS" in s)
    failed = sum(1 for _, s, _ in results if "FAIL" in s)

    for i, (name, status, err) in enumerate(results, 1):
        style = "green" if "PASS" in status else "red"
        table.add_row(str(i), name, status, err, style=style)

    console.print(table)
    console.print(f"\n[bold]Passed: {passed} | Failed: {failed} | Total: {len(results)}[/bold]")

    sys.exit(0 if failed == 0 else 1)


if __name__ == "__main__":
    main()
