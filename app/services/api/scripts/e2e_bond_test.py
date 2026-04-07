#!/usr/bin/env python3
"""
Treasury API — Bond Module E2E Test Suite
==========================================
Tests ~30 scenarios against the Bond API endpoints.

Usage:
    python e2e_bond_test.py --base-url http://localhost:34001
"""

import argparse
import sys
import uuid
from datetime import datetime, timedelta, timezone
from typing import Optional

import requests
from rich.console import Console
from rich.table import Table
from rich.panel import Panel
from rich import box

# ─── Config ───────────────────────────────────────────────────────────────────

BASE_URL = "http://localhost:34001"
API = f"{BASE_URL}/api/v1"
PASSWORD = "P@ssw0rd123"

USERS = {
    "dealer": "dealer01",
    "deskhead": "deskhead01",
    "director": "director01",
    "accountant": "accountant01",
    "chiefacc": "chiefacc01",
    "divhead": "divhead01",
    "admin": "admin01",
}

COUNTERPARTY_MSB = "e0000000-0000-0000-0000-000000000001"

console = Console()

# ─── API Client ──────────────────────────────────────────────────────────────


class BondAPIClient:
    """Wrapper around requests.Session with cookie-based auth for Bond API."""

    def __init__(self):
        self.session = requests.Session()

    def login(self, username: str, password: str = PASSWORD) -> requests.Response:
        return self.session.post(
            f"{API}/auth/login",
            json={"username": username, "password": password},
        )

    def me(self) -> requests.Response:
        return self.session.get(f"{API}/auth/me")

    def logout(self) -> requests.Response:
        return self.session.post(f"{API}/auth/logout")

    # ─── Bond CRUD ───
    def create_bond(self, payload: dict) -> requests.Response:
        return self.session.post(f"{API}/bonds", json=payload)

    def get_bond(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/bonds/{deal_id}")

    def list_bond(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/bonds", params=params or {})

    def update_bond(self, deal_id: str, payload: dict) -> requests.Response:
        return self.session.put(f"{API}/bonds/{deal_id}", json=payload)

    def delete_bond(self, deal_id: str) -> requests.Response:
        return self.session.delete(f"{API}/bonds/{deal_id}")

    # ─── Bond Workflow ───
    def approve_bond(self, deal_id: str, action: str, comment: Optional[str] = None) -> requests.Response:
        body = {"action": action}
        if comment:
            body["comment"] = comment
        return self.session.post(f"{API}/bonds/{deal_id}/approve", json=body)

    def recall_bond(self, deal_id: str, reason: str) -> requests.Response:
        return self.session.post(f"{API}/bonds/{deal_id}/recall", json={"reason": reason})

    def cancel_bond(self, deal_id: str, reason: str) -> requests.Response:
        return self.session.post(f"{API}/bonds/{deal_id}/cancel", json={"reason": reason})

    def cancel_approve_bond(self, deal_id: str, action: str, comment: Optional[str] = None) -> requests.Response:
        body = {"action": action}
        if comment:
            body["comment"] = comment
        return self.session.post(f"{API}/bonds/{deal_id}/cancel-approve", json=body)

    def clone_bond(self, deal_id: str) -> requests.Response:
        return self.session.post(f"{API}/bonds/{deal_id}/clone")

    def get_bond_history(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/bonds/{deal_id}/history")

    def list_inventory(self) -> requests.Response:
        return self.session.get(f"{API}/bonds/inventory")


# ─── Helpers ──────────────────────────────────────────────────────────────────


def _trade_date() -> str:
    """Use a future date to avoid deal_number conflicts with seed data."""
    return "2026-04-15T00:00:00Z"


def _value_date(days_offset: int = 0) -> str:
    dt = datetime(2026, 4, 15, tzinfo=timezone.utc) + timedelta(days=days_offset)
    return dt.strftime("%Y-%m-%dT00:00:00Z")


def _maturity_date() -> str:
    dt = datetime(2027, 4, 15, tzinfo=timezone.utc)
    return dt.strftime("%Y-%m-%dT00:00:00Z")


def make_govi_buy(bond_code: Optional[str] = None) -> dict:
    """Build a GOVERNMENT bond BUY deal payload."""
    return {
        "bond_category": "GOVERNMENT",
        "trade_date": _trade_date(),
        "value_date": _value_date(),
        "direction": "BUY",
        "counterparty_id": COUNTERPARTY_MSB,
        "transaction_type": "OUTRIGHT",
        "bond_code_manual": bond_code or f"GV-E2E-{uuid.uuid4().hex[:8]}",
        "issuer": "Kho bạc Nhà nước",
        "coupon_rate": "5.5",
        "maturity_date": _maturity_date(),
        "quantity": 100,
        "face_value": "100000",
        "discount_rate": "0",
        "clean_price": "98500",
        "settlement_price": "99000",
        "total_value": "9900000",
        "portfolio_type": "HTM",
        "payment_date": _value_date(),
        "remaining_tenor_days": 365,
        "confirmation_method": "EMAIL",
        "contract_prepared_by": "INTERNAL",
        "note": "E2E bond test",
    }


def make_fi_buy() -> dict:
    """Build a FINANCIAL_INSTITUTION bond BUY deal payload."""
    return {
        "bond_category": "FINANCIAL_INSTITUTION",
        "trade_date": _trade_date(),
        "value_date": _value_date(),
        "direction": "BUY",
        "counterparty_id": COUNTERPARTY_MSB,
        "transaction_type": "OUTRIGHT",
        "bond_code_manual": f"FI-E2E-{uuid.uuid4().hex[:8]}",
        "issuer": "Vietcombank",
        "coupon_rate": "6.2",
        "maturity_date": _maturity_date(),
        "quantity": 50,
        "face_value": "100000",
        "discount_rate": "0",
        "clean_price": "99000",
        "settlement_price": "99500",
        "total_value": "4975000",
        "portfolio_type": "AFS",
        "payment_date": _value_date(),
        "remaining_tenor_days": 365,
        "confirmation_method": "REUTERS",
        "contract_prepared_by": "COUNTERPARTY",
    }


def make_cctg_buy() -> dict:
    """Build a CERTIFICATE_OF_DEPOSIT bond BUY deal payload."""
    return {
        "bond_category": "CERTIFICATE_OF_DEPOSIT",
        "trade_date": _trade_date(),
        "value_date": _value_date(),
        "direction": "BUY",
        "counterparty_id": COUNTERPARTY_MSB,
        "transaction_type": "OUTRIGHT",
        "bond_code_manual": f"CD-E2E-{uuid.uuid4().hex[:8]}",
        "issuer": "BIDV",
        "coupon_rate": "7.0",
        "maturity_date": _maturity_date(),
        "quantity": 30,
        "face_value": "1000000",
        "discount_rate": "0",
        "clean_price": "995000",
        "settlement_price": "997000",
        "total_value": "29910000",
        "portfolio_type": "HFT",
        "payment_date": _value_date(),
        "remaining_tenor_days": 180,
        "confirmation_method": "EMAIL",
        "contract_prepared_by": "INTERNAL",
    }


def make_govi_sell(bond_code: str, quantity: int = 50) -> dict:
    """Build a GOVERNMENT bond SELL deal payload."""
    return {
        "bond_category": "GOVERNMENT",
        "trade_date": _trade_date(),
        "value_date": _value_date(),
        "direction": "SELL",
        "counterparty_id": COUNTERPARTY_MSB,
        "transaction_type": "OUTRIGHT",
        "bond_code_manual": bond_code,
        "issuer": "Kho bạc Nhà nước",
        "coupon_rate": "5.5",
        "maturity_date": _maturity_date(),
        "quantity": quantity,
        "face_value": "100000",
        "discount_rate": "0",
        "clean_price": "98500",
        "settlement_price": "99000",
        "total_value": str(quantity * 99000),
        "portfolio_type": "HTM",
        "payment_date": _value_date(),
        "remaining_tenor_days": 365,
        "confirmation_method": "EMAIL",
        "contract_prepared_by": "INTERNAL",
    }


# ─── Test Framework ───────────────────────────────────────────────────────────

results: list[dict] = []


def record(scenario: str, name: str, passed: bool, detail: str = ""):
    results.append({"scenario": scenario, "name": name, "passed": passed, "detail": detail})
    status = "[green]✓ PASS[/green]" if passed else "[red]✗ FAIL[/red]"
    console.print(f"  {status}  {name}" + (f"  [dim]({detail})[/dim]" if detail else ""))


# ─── Helper: approve through full chain → COMPLETED ──────────────────────────


def _approve_to_completed(deal_id: str, dealer, deskhead, director, accountant, chiefacc, scenario: str) -> bool:
    """Approve a bond deal through the full chain to COMPLETED. Returns True on success."""
    # L1: DeskHead
    r = deskhead.approve_bond(deal_id, "APPROVE")
    if r.status_code != 200:
        record(scenario, "SKIP: L1 approve failed", False, r.text[:200])
        return False

    # L2: Director
    r = director.approve_bond(deal_id, "APPROVE")
    if r.status_code != 200:
        record(scenario, "SKIP: L2 approve failed", False, r.text[:200])
        return False

    # Accountant
    r = accountant.approve_bond(deal_id, "APPROVE")
    if r.status_code != 200:
        record(scenario, "SKIP: Accountant approve failed", False, r.text[:200])
        return False

    # Chief Accountant
    r = chiefacc.approve_bond(deal_id, "APPROVE")
    if r.status_code != 200:
        record(scenario, "SKIP: ChiefAccountant approve failed", False, r.text[:200])
        return False

    # Verify
    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    if status != "COMPLETED":
        record(scenario, f"SKIP: expected COMPLETED, got {status}", False)
        return False
    return True


# ─── Scenario 1: Create Govi Bond Buy → Full Approval → COMPLETED ───────────


def test_govi_full_approval():
    scenario = "1. Govi Bond Full Approval"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    # 1.1 Create Govi Bond BUY
    r = dealer.create_bond(make_govi_buy())
    ok = r.status_code == 201 and r.json()["data"]["status"] == "OPEN"
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create Govi Bond BUY → 201 OPEN", ok, f"id={deal_id}")
    if not deal_id:
        record(scenario, "SKIP remaining (create failed)", False, r.text[:200] if r else "")
        return

    # 1.2 Verify deal number starts with G
    deal_number = r.json()["data"]["deal_number"]
    ok = deal_number.startswith("G-")
    record(scenario, f"Deal number starts with G-: {deal_number}", ok)

    # 1.3 Verify bond category
    ok = r.json()["data"]["bond_category"] == "GOVERNMENT"
    record(scenario, "Bond category is GOVERNMENT", ok)

    # 1.4 Full approval chain
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])
    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])
    chiefacc = BondAPIClient()
    chiefacc.login(USERS["chiefacc"])

    # L1: DeskHead
    r = deskhead.approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "DeskHead approves L1 → PENDING_L2_APPROVAL", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_L2_APPROVAL"
    record(scenario, "Status is PENDING_L2_APPROVAL", ok, f"actual={status}")

    # L2: Director
    r = director.approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "Director approves L2 → PENDING_BOOKING", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_BOOKING"
    record(scenario, "Status is PENDING_BOOKING", ok, f"actual={status}")

    # Accountant
    r = accountant.approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "Accountant approves → PENDING_CHIEF_ACCOUNTANT", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_CHIEF_ACCOUNTANT"
    record(scenario, "Status is PENDING_CHIEF_ACCOUNTANT", ok, f"actual={status}")

    # Chief Accountant → COMPLETED
    r = chiefacc.approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "ChiefAccountant approves → COMPLETED", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "COMPLETED"
    record(scenario, "Status is COMPLETED", ok, f"actual={status}")


# ─── Scenario 2: FI Bond Full Approval ───────────────────────────────────────


def test_fi_full_approval():
    scenario = "2. FI Bond Full Approval"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    r = dealer.create_bond(make_fi_buy())
    ok = r.status_code == 201 and r.json()["data"]["status"] == "OPEN"
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create FI Bond BUY → 201 OPEN", ok, f"id={deal_id}")
    if not deal_id:
        return

    # Verify F- prefix
    deal_number = r.json()["data"]["deal_number"]
    ok = deal_number.startswith("F-")
    record(scenario, f"Deal number starts with F-: {deal_number}", ok)

    ok = r.json()["data"]["bond_category"] == "FINANCIAL_INSTITUTION"
    record(scenario, "Bond category is FINANCIAL_INSTITUTION", ok)

    # Full approval
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])
    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])
    chiefacc = BondAPIClient()
    chiefacc.login(USERS["chiefacc"])

    ok = _approve_to_completed(deal_id, dealer, deskhead, director, accountant, chiefacc, scenario)
    record(scenario, "Full approval → COMPLETED", ok)


# ─── Scenario 3: CCTG Full Approval ──────────────────────────────────────────


def test_cctg_full_approval():
    scenario = "3. CCTG Full Approval"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    r = dealer.create_bond(make_cctg_buy())
    ok = r.status_code == 201 and r.json()["data"]["status"] == "OPEN"
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create CCTG BUY → 201 OPEN", ok, f"id={deal_id}")
    if not deal_id:
        return

    # Verify F- prefix for CCTG too
    deal_number = r.json()["data"]["deal_number"]
    ok = deal_number.startswith("F-")
    record(scenario, f"CCTG deal number starts with F-: {deal_number}", ok)

    ok = r.json()["data"]["bond_category"] == "CERTIFICATE_OF_DEPOSIT"
    record(scenario, "Bond category is CERTIFICATE_OF_DEPOSIT", ok)

    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])
    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])
    chiefacc = BondAPIClient()
    chiefacc.login(USERS["chiefacc"])

    ok = _approve_to_completed(deal_id, dealer, deskhead, director, accountant, chiefacc, scenario)
    record(scenario, "Full approval → COMPLETED", ok)


# ─── Scenario 4: Sell Without Inventory → Block ──────────────────────────────


def test_sell_no_inventory():
    scenario = "4. Sell No Inventory"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    bond_code = f"NO-INV-{uuid.uuid4().hex[:8]}"
    r = dealer.create_bond(make_govi_sell(bond_code, 100))
    ok = r.status_code == 422  # INSUFFICIENT_INVENTORY
    record(scenario, "Sell without inventory → 422", ok, f"status={r.status_code}")


# ─── Scenario 5: Buy Then Sell → Inventory Flow ──────────────────────────────


def test_buy_then_sell():
    scenario = "5. Buy Then Sell (Inventory)"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])
    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])
    chiefacc = BondAPIClient()
    chiefacc.login(USERS["chiefacc"])

    bond_code = f"INV-FLOW-{uuid.uuid4().hex[:8]}"

    # 5.1 Create BUY deal
    r = dealer.create_bond(make_govi_buy(bond_code))
    ok = r.status_code == 201
    buy_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create BUY deal", ok, f"id={buy_id}")
    if not buy_id:
        return

    # 5.2 Approve to COMPLETED (inventory should increase)
    ok = _approve_to_completed(buy_id, dealer, deskhead, director, accountant, chiefacc, scenario)
    record(scenario, "BUY deal COMPLETED", ok)
    if not ok:
        return

    # 5.3 Now sell (should succeed if inventory was created)
    r = dealer.create_bond(make_govi_sell(bond_code, 50))
    ok = r.status_code == 201
    sell_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create SELL deal (50 units)", ok, f"id={sell_id}, status={r.status_code}")

    # 5.4 Oversell should fail
    if ok:
        r = dealer.create_bond(make_govi_sell(bond_code, 9999))
        ok = r.status_code == 422
        record(scenario, "Oversell blocked → 422", ok, f"status={r.status_code}")


# ─── Scenario 6: Cancel Completed Deal (2-Level) ─────────────────────────────


def test_cancel_flow():
    scenario = "6. Cancel Flow (2-Level)"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])
    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])
    chiefacc = BondAPIClient()
    chiefacc.login(USERS["chiefacc"])

    # Create and complete a deal
    r = dealer.create_bond(make_govi_buy())
    if r.status_code != 201:
        record(scenario, "SKIP: create failed", False, r.text[:200])
        return
    deal_id = r.json()["data"]["id"]

    ok = _approve_to_completed(deal_id, dealer, deskhead, director, accountant, chiefacc, scenario)
    if not ok:
        return
    record(scenario, "Deal COMPLETED", True)

    # 6.1 Request cancel → PENDING_CANCEL_L1
    r = dealer.cancel_bond(deal_id, "Khách hàng hủy giao dịch")
    ok = r.status_code == 200
    record(scenario, "Request cancel → PENDING_CANCEL_L1", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_CANCEL_L1"
    record(scenario, "Status is PENDING_CANCEL_L1", ok, f"actual={status}")

    # 6.2 DeskHead approves cancel L1 → PENDING_CANCEL_L2
    r = deskhead.cancel_approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "DeskHead approves cancel L1", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_CANCEL_L2"
    record(scenario, "Status is PENDING_CANCEL_L2", ok, f"actual={status}")

    # 6.3 Director approves cancel L2 → CANCELLED
    r = director.cancel_approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "Director approves cancel L2 → CANCELLED", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "CANCELLED"
    record(scenario, "Status is CANCELLED", ok, f"actual={status}")


# ─── Scenario 7: Cancel Reject ───────────────────────────────────────────────


def test_cancel_reject():
    scenario = "7. Cancel Reject"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])
    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])
    chiefacc = BondAPIClient()
    chiefacc.login(USERS["chiefacc"])

    r = dealer.create_bond(make_govi_buy())
    if r.status_code != 201:
        record(scenario, "SKIP: create failed", False)
        return
    deal_id = r.json()["data"]["id"]

    ok = _approve_to_completed(deal_id, dealer, deskhead, director, accountant, chiefacc, scenario)
    if not ok:
        return

    # Request cancel
    dealer.cancel_bond(deal_id, "Yêu cầu hủy")

    # DeskHead rejects → back to COMPLETED
    r = deskhead.cancel_approve_bond(deal_id, "REJECT", "Không đủ lý do")
    ok = r.status_code == 200
    record(scenario, "DeskHead rejects cancel", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "COMPLETED"
    record(scenario, "Status back to COMPLETED", ok, f"actual={status}")


# ─── Scenario 8: Recall Deal ─────────────────────────────────────────────────


def test_recall():
    scenario = "8. Recall"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    r = dealer.create_bond(make_govi_buy())
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create deal", ok)
    if not deal_id:
        return

    # DeskHead approves L1
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    r = deskhead.approve_bond(deal_id, "APPROVE")
    ok = r.status_code == 200
    record(scenario, "DeskHead approves L1", ok)

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_L2_APPROVAL"
    record(scenario, "Status is PENDING_L2_APPROVAL", ok, f"actual={status}")

    # Dealer recalls → back to OPEN
    r = dealer.recall_bond(deal_id, "Sai đối tác, cần chỉnh sửa")
    ok = r.status_code == 200
    record(scenario, "Dealer recalls → OPEN", ok, f"status={r.status_code}")

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "OPEN"
    record(scenario, "Status back to OPEN", ok, f"actual={status}")


# ─── Scenario 9: Clone Rejected Deal ─────────────────────────────────────────


def test_clone_rejected():
    scenario = "9. Clone Rejected"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    r = dealer.create_bond(make_govi_buy())
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create deal", ok)
    if not deal_id:
        return

    # DeskHead approves, Director rejects
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    deskhead.approve_bond(deal_id, "APPROVE")

    director = BondAPIClient()
    director.login(USERS["director"])
    r = director.approve_bond(deal_id, "REJECT", comment="Giá không phù hợp")
    ok = r.status_code == 200
    record(scenario, "Director rejects → REJECTED", ok)

    r = dealer.get_bond(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "REJECTED"
    record(scenario, "Status is REJECTED", ok, f"actual={status}")

    # Clone rejected deal
    r = dealer.clone_bond(deal_id)
    ok = r.status_code == 201 and r.json()["data"]["status"] == "OPEN"
    clone_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Clone rejected deal → OPEN", ok, f"clone_id={clone_id}")

    if clone_id:
        ok = clone_id != deal_id
        record(scenario, "Clone has new ID", ok)


# ─── Scenario 10: Permission — Accountant Cannot Create ──────────────────────


def test_accountant_cannot_create():
    scenario = "10. Accountant Cannot Create"
    console.rule(f"[bold]{scenario}[/bold]")

    accountant = BondAPIClient()
    accountant.login(USERS["accountant"])

    r = accountant.create_bond(make_govi_buy())
    ok = r.status_code == 403
    record(scenario, "Accountant create bond → 403", ok, f"status={r.status_code}")


# ─── Scenario 11: Validation — Negative Quantity ─────────────────────────────


def test_validation_negative_quantity():
    scenario = "11. Validation: Negative Quantity"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    payload = make_govi_buy()
    payload["quantity"] = -10
    r = dealer.create_bond(payload)
    ok = r.status_code == 400
    record(scenario, "Negative quantity → 400", ok, f"status={r.status_code}")


# ─── Scenario 12: Validation — Zero Quantity ─────────────────────────────────


def test_validation_zero_quantity():
    scenario = "12. Validation: Zero Quantity"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    payload = make_govi_buy()
    payload["quantity"] = 0
    r = dealer.create_bond(payload)
    ok = r.status_code == 400
    record(scenario, "Zero quantity → 400", ok, f"status={r.status_code}")


# ─── Scenario 13: Validation — Missing Required Fields ───────────────────────


def test_validation_missing_fields():
    scenario = "13. Validation: Missing Fields"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    # Missing issuer
    payload = make_govi_buy()
    payload["issuer"] = ""
    r = dealer.create_bond(payload)
    ok = r.status_code == 400
    record(scenario, "Missing issuer → 400", ok, f"status={r.status_code}")

    # Missing counterparty
    payload = make_govi_buy()
    payload["counterparty_id"] = ""
    r = dealer.create_bond(payload)
    ok = r.status_code in (400, 422)
    record(scenario, "Missing counterparty → 400/422", ok, f"status={r.status_code}")

    # BUY without portfolio_type
    payload = make_govi_buy()
    del payload["portfolio_type"]
    r = dealer.create_bond(payload)
    ok = r.status_code == 400
    record(scenario, "BUY without portfolio_type → 400", ok, f"status={r.status_code}")


# ─── Scenario 14: Self-Approval Block ────────────────────────────────────────


def test_self_approval():
    scenario = "14. Self-Approval Block"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    r = dealer.create_bond(make_govi_buy())
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Dealer creates deal", ok)
    if not deal_id:
        return

    # Dealer tries to approve own deal
    r = dealer.approve_bond(deal_id, "APPROVE")
    ok = r.status_code in (403, 409)
    record(scenario, "Dealer cannot approve own deal", ok, f"status={r.status_code}")


# ─── Scenario 15: Bond CRUD ──────────────────────────────────────────────────


def test_bond_crud():
    scenario = "15. Bond CRUD"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    # Create
    r = dealer.create_bond(make_govi_buy())
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create bond → 201", ok, f"id={deal_id}")
    if not deal_id:
        return

    # Get
    r = dealer.get_bond(deal_id)
    ok = r.status_code == 200 and r.json()["data"]["id"] == deal_id
    record(scenario, "Get bond by ID", ok)

    # List
    r = dealer.list_bond({"page": 1, "page_size": 10})
    ok = r.status_code == 200 and r.json()["data"]["total"] > 0
    record(scenario, "List bonds with pagination", ok)

    # List with filter
    r = dealer.list_bond({"status": "OPEN", "bond_category": "GOVERNMENT"})
    ok = r.status_code == 200
    record(scenario, "List with status+category filter", ok)

    # Delete (soft)
    r = dealer.delete_bond(deal_id)
    ok = r.status_code == 204
    record(scenario, "Soft delete bond → 204", ok, f"status={r.status_code}")

    # Get after delete → 404
    r = dealer.get_bond(deal_id)
    ok = r.status_code == 404
    record(scenario, "Get deleted bond → 404", ok, f"status={r.status_code}")


# ─── Scenario 16: Approval History ───────────────────────────────────────────


def test_approval_history():
    scenario = "16. Approval History"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])

    r = dealer.create_bond(make_govi_buy())
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Create deal", ok)
    if not deal_id:
        return

    # Approve L1
    deskhead.approve_bond(deal_id, "APPROVE")

    # Get history
    r = dealer.get_bond_history(deal_id)
    ok = r.status_code == 200
    record(scenario, "GET /bond/{id}/history → 200", ok, f"status={r.status_code}")

    if ok:
        entries = r.json().get("data", [])
        ok = len(entries) >= 1
        record(scenario, f"History has entries: {len(entries)}", ok)

        if len(entries) >= 1:
            first = entries[0]
            ok = "action_type" in first and "performer_name" in first
            record(scenario, "Entry has required fields", ok)


# ─── Scenario 17: Reject at Each Level ───────────────────────────────────────


def test_reject_at_each_level():
    scenario = "17. Reject at Each Level"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])
    deskhead = BondAPIClient()
    deskhead.login(USERS["deskhead"])
    director = BondAPIClient()
    director.login(USERS["director"])

    # 17.1 DeskHead rejects → REJECTED
    r = dealer.create_bond(make_govi_buy())
    deal_id = r.json()["data"]["id"] if r.status_code == 201 else None
    if deal_id:
        r = deskhead.approve_bond(deal_id, "REJECT", comment="Giá không hợp lý")
        ok = r.status_code == 200
        record(scenario, "DeskHead rejects → REJECTED", ok)
        r = dealer.get_bond(deal_id)
        status = r.json()["data"]["status"] if r.status_code == 200 else "?"
        ok = status == "REJECTED"
        record(scenario, "Status is REJECTED (L1)", ok, f"actual={status}")

    # 17.2 Director rejects → REJECTED
    r = dealer.create_bond(make_govi_buy())
    deal_id = r.json()["data"]["id"] if r.status_code == 201 else None
    if deal_id:
        deskhead.approve_bond(deal_id, "APPROVE")
        r = director.approve_bond(deal_id, "REJECT", comment="Rủi ro quá cao")
        ok = r.status_code == 200
        record(scenario, "Director rejects → REJECTED", ok)
        r = dealer.get_bond(deal_id)
        status = r.json()["data"]["status"] if r.status_code == 200 else "?"
        ok = status == "REJECTED"
        record(scenario, "Status is REJECTED (L2)", ok, f"actual={status}")


# ─── Scenario 18: Inventory Endpoint ─────────────────────────────────────────


def test_inventory():
    scenario = "18. Inventory"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    r = dealer.list_inventory()
    ok = r.status_code == 200
    record(scenario, "GET /bond/inventory → 200", ok, f"status={r.status_code}")


# ─── Scenario 19: Transaction Types ──────────────────────────────────────────


def test_transaction_types():
    scenario = "19. Transaction Types"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = BondAPIClient()
    dealer.login(USERS["dealer"])

    for tx_type in ["OUTRIGHT", "REPO", "REVERSE_REPO"]:
        payload = make_govi_buy()
        payload["transaction_type"] = tx_type
        r = dealer.create_bond(payload)
        ok = r.status_code == 201 and r.json()["data"]["transaction_type"] == tx_type
        record(scenario, f"Create {tx_type} → 201", ok, f"status={r.status_code}")


# ─── Scenario 20: No Auth → 401 ──────────────────────────────────────────────


def test_no_auth():
    scenario = "20. No Auth"
    console.rule(f"[bold]{scenario}[/bold]")

    r = requests.get(f"{API}/bonds")
    ok = r.status_code == 401
    record(scenario, "No auth → 401 on /bond", ok, f"status={r.status_code}")

    r = requests.post(f"{API}/bonds", json=make_govi_buy())
    ok = r.status_code == 401
    record(scenario, "No auth → 401 on POST /bond", ok, f"status={r.status_code}")


# ─── Main ─────────────────────────────────────────────────────────────────────


def print_summary():
    table = Table(title="Bond E2E Test Results", box=box.ROUNDED)
    table.add_column("Scenario", style="bold")
    table.add_column("Test", min_width=40)
    table.add_column("Result", justify="center")
    table.add_column("Detail", style="dim")

    for r in results:
        status = "[green]PASS[/green]" if r["passed"] else "[red]FAIL[/red]"
        table.add_row(r["scenario"], r["name"], status, r["detail"])

    console.print()
    console.print(table)

    total = len(results)
    passed = sum(1 for r in results if r["passed"])
    failed = total - passed

    color = "green" if failed == 0 else "red"
    console.print(Panel(
        f"[bold {color}]{passed}/{total} passed, {failed} failed[/bold {color}]",
        title="Summary",
        border_style=color,
    ))

    return failed == 0


def main():
    global API, BASE_URL
    parser = argparse.ArgumentParser(description="Treasury Bond API E2E Tests")
    parser.add_argument("--base-url", default="http://localhost:34001", help="API base URL")
    args = parser.parse_args()

    BASE_URL = args.base_url.rstrip("/")
    API = f"{BASE_URL}/api/v1"

    console.print(Panel(
        f"[bold]Treasury Bond API E2E Test Suite[/bold]\nTarget: {BASE_URL}",
        border_style="blue",
    ))

    # Verify API is up
    try:
        r = requests.get(f"{BASE_URL}/health", timeout=5)
        if r.status_code != 200:
            console.print(f"[red]Health check failed: {r.status_code}[/red]")
            sys.exit(1)
    except requests.ConnectionError:
        console.print(f"[red]Cannot connect to {BASE_URL}[/red]")
        sys.exit(1)

    console.print("[green]Health check OK[/green]\n")

    # Run all scenarios
    test_govi_full_approval()
    test_fi_full_approval()
    test_cctg_full_approval()
    test_sell_no_inventory()
    test_buy_then_sell()
    test_bond_crud()
    test_recall()
    test_clone_rejected()
    test_self_approval()
    test_accountant_cannot_create()
    test_validation_negative_quantity()
    test_validation_zero_quantity()
    test_validation_missing_fields()
    test_reject_at_each_level()
    test_approval_history()
    test_inventory()
    test_transaction_types()
    test_no_auth()
    # Destructive tests last
    test_cancel_flow()
    test_cancel_reject()

    all_passed = print_summary()
    sys.exit(0 if all_passed else 1)


if __name__ == "__main__":
    main()
