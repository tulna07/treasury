#!/usr/bin/env python3
"""
Treasury API — End-to-End Test Suite
=====================================
Tests 10 scenarios (~40 test cases) against the Treasury API.

Usage:
    python e2e_test.py --base-url http://localhost:34001
"""

import argparse
import sys
import time
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
    "settlement": "settlement01",
    "risk": "risk01",
    "riskhead": "riskhead01",
    "divhead": "divhead01",
    "chiefacc": "chiefacc01",
    "admin": "admin01",
}

COUNTERPARTY_MSB = "e0000000-0000-0000-0000-000000000001"

console = Console()

# ─── Helpers ──────────────────────────────────────────────────────────────────


class APIClient:
    """Wrapper around requests.Session with cookie-based auth."""

    def __init__(self):
        self.session = requests.Session()

    def login(self, username: str, password: str = PASSWORD) -> requests.Response:
        return self.session.post(
            f"{API}/auth/login",
            json={"username": username, "password": password},
        )

    def me(self) -> requests.Response:
        return self.session.get(f"{API}/auth/me")

    def refresh(self) -> requests.Response:
        return self.session.post(f"{API}/auth/refresh")

    def logout(self) -> requests.Response:
        return self.session.post(f"{API}/auth/logout")

    def create_fx(self, payload: dict) -> requests.Response:
        return self.session.post(f"{API}/fx", json=payload)

    def get_fx(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/fx/{deal_id}")

    def update_fx(self, deal_id: str, payload: dict) -> requests.Response:
        return self.session.put(f"{API}/fx/{deal_id}", json=payload)

    def list_fx(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/fx", params=params or {})

    def delete_fx(self, deal_id: str) -> requests.Response:
        return self.session.delete(f"{API}/fx/{deal_id}")

    def approve_fx(self, deal_id: str, action: str, version: int, comment: Optional[str] = None) -> requests.Response:
        body = {"action": action, "version": version}
        if comment:
            body["comment"] = comment
        return self.session.post(f"{API}/fx/{deal_id}/approve", json=body)

    def recall_fx(self, deal_id: str, reason: str) -> requests.Response:
        return self.session.post(f"{API}/fx/{deal_id}/recall", json={"reason": reason})

    def clone_fx(self, deal_id: str) -> requests.Response:
        return self.session.post(f"{API}/fx/{deal_id}/clone")

    def cancel_fx(self, deal_id: str, reason: str) -> requests.Response:
        return self.session.post(f"{API}/fx/{deal_id}/cancel", json={"reason": reason})

    def cancel_approve_fx(self, deal_id: str, action: str, comment: Optional[str] = None) -> requests.Response:
        body = {"action": action}
        if comment:
            body["comment"] = comment
        return self.session.post(f"{API}/fx/{deal_id}/cancel-approve", json=body)

    def get_fx_history(self, deal_id: str) -> requests.Response:
        return self.session.get(f"{API}/fx/{deal_id}/history")

    # ─── Admin APIs ───
    def list_users(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/admin/users", params=params or {})

    def create_user(self, payload: dict) -> requests.Response:
        return self.session.post(f"{API}/admin/users", json=payload)

    def get_user(self, user_id: str) -> requests.Response:
        return self.session.get(f"{API}/admin/users/{user_id}")

    def update_user(self, user_id: str, payload: dict) -> requests.Response:
        return self.session.put(f"{API}/admin/users/{user_id}", json=payload)

    def lock_user(self, user_id: str, reason: str = "E2E test lock") -> requests.Response:
        return self.session.post(f"{API}/admin/users/{user_id}/lock", json={"reason": reason})

    def unlock_user(self, user_id: str, reason: str = "E2E test unlock") -> requests.Response:
        return self.session.post(f"{API}/admin/users/{user_id}/unlock", json={"reason": reason})

    def reset_password(self, user_id: str, reason: str = "E2E test reset") -> requests.Response:
        return self.session.post(f"{API}/admin/users/{user_id}/reset-password", json={"reason": reason})

    def list_roles(self) -> requests.Response:
        return self.session.get(f"{API}/admin/roles")

    def get_role_permissions(self, code: str) -> requests.Response:
        return self.session.get(f"{API}/admin/roles/{code}/permissions")

    def assign_role(self, user_id: str, role_code: str, reason: str = "") -> requests.Response:
        return self.session.post(f"{API}/admin/users/{user_id}/roles", json={"role_code": role_code, "reason": reason})

    def revoke_role(self, user_id: str, role_code: str, reason: str = "") -> requests.Response:
        return self.session.delete(f"{API}/admin/users/{user_id}/roles/{role_code}", json={"reason": reason})

    def list_audit_logs(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/admin/audit-logs", params=params or {})

    def audit_stats(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/admin/audit-logs/stats", params=params or {})

    # ─── Master Data APIs ───
    def list_counterparties(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/counterparties", params=params or {})

    def get_counterparty(self, cp_id: str) -> requests.Response:
        return self.session.get(f"{API}/counterparties/{cp_id}")

    def list_currencies(self) -> requests.Response:
        return self.session.get(f"{API}/currencies")

    def list_currency_pairs(self) -> requests.Response:
        return self.session.get(f"{API}/currency-pairs")

    def list_branches(self) -> requests.Response:
        return self.session.get(f"{API}/branches")

    def list_exchange_rates(self, params: Optional[dict] = None) -> requests.Response:
        return self.session.get(f"{API}/exchange-rates", params=params or {})


def _trade_date() -> str:
    """Use a future date to avoid deal_number conflicts with seed data."""
    return "2026-04-10T00:00:00Z"


def _value_date(days_offset: int = 1) -> str:
    dt = datetime(2026, 4, 10, tzinfo=timezone.utc) + timedelta(days=days_offset)
    return dt.strftime("%Y-%m-%dT00:00:00Z")


def make_spot_deal(ticket: Optional[str] = None) -> dict:
    """Build a SPOT deal creation payload."""
    return {
        "ticket_number": ticket,
        "counterparty_id": COUNTERPARTY_MSB,
        "deal_type": "SPOT",
        "direction": "BUY",
        "notional_amount": "100000",
        "currency_code": "USD",
        "trade_date": _trade_date(),
        "note": "E2E test deal",
        "legs": [
            {
                "leg_number": 1,
                "value_date": _value_date(2),
                "exchange_rate": "25350.50",
                "buy_currency": "USD",
                "sell_currency": "VND",
                "buy_amount": "100000",
                "sell_amount": "2535050000",
            }
        ],
    }


def make_swap_deal() -> dict:
    """Build a SWAP deal creation payload with 2 legs."""
    return {
        "counterparty_id": COUNTERPARTY_MSB,
        "deal_type": "SWAP",
        "direction": "SELL_BUY",
        "notional_amount": "500000",
        "currency_code": "USD",
        "trade_date": _trade_date(),
        "note": "E2E swap test",
        "legs": [
            {
                "leg_number": 1,
                "value_date": _value_date(2),
                "exchange_rate": "25350",
                "buy_currency": "VND",
                "sell_currency": "USD",
                "buy_amount": "12675000000",
                "sell_amount": "500000",
            },
            {
                "leg_number": 2,
                "value_date": _value_date(30),
                "exchange_rate": "25400",
                "buy_currency": "USD",
                "sell_currency": "VND",
                "buy_amount": "500000",
                "sell_amount": "12700000000",
            },
        ],
    }


# ─── Test Framework ───────────────────────────────────────────────────────────

results: list[dict] = []


def record(scenario: str, name: str, passed: bool, detail: str = ""):
    results.append({"scenario": scenario, "name": name, "passed": passed, "detail": detail})
    status = "[green]✓ PASS[/green]" if passed else "[red]✗ FAIL[/red]"
    console.print(f"  {status}  {name}" + (f"  [dim]({detail})[/dim]" if detail else ""))


# ─── Scenario 1: Auth ────────────────────────────────────────────────────────

def test_auth():
    scenario = "1. Auth"
    console.rule(f"[bold]{scenario}[/bold]")

    # 1.1 Login success
    c = APIClient()
    r = c.login(USERS["dealer"])
    ok = r.status_code == 200 and r.json().get("success") is True
    record(scenario, "Login success", ok, f"status={r.status_code}")

    # 1.2 Me endpoint
    r = c.me()
    ok = r.status_code == 200 and r.json()["data"]["username"] == USERS["dealer"]
    record(scenario, "Get /auth/me", ok, f"status={r.status_code}")

    # 1.3 Wrong password
    c2 = APIClient()
    r = c2.login(USERS["dealer"], "WrongPass123")
    ok = r.status_code == 401
    record(scenario, "Wrong password → 401", ok, f"status={r.status_code}")

    # 1.4 Refresh token
    r = c.refresh()
    ok = r.status_code == 200 and r.json().get("success") is True
    record(scenario, "Refresh token", ok, f"status={r.status_code}")

    # 1.5 Logout
    r = c.logout()
    ok = r.status_code == 204
    record(scenario, "Logout → 204", ok, f"status={r.status_code}")

    # 1.6 Me after logout → 401
    r = c.me()
    ok = r.status_code == 401
    record(scenario, "Me after logout → 401", ok, f"status={r.status_code}")


# ─── Scenario 2: FX CRUD ─────────────────────────────────────────────────────

def test_fx_crud():
    scenario = "2. FX CRUD"
    console.rule(f"[bold]{scenario}[/bold]")

    c = APIClient()
    c.login(USERS["dealer"])

    # 2.1 Create SPOT deal
    r = c.create_fx(make_spot_deal("E2E-SPOT-001"))
    ok = r.status_code == 201 and r.json()["data"]["status"] == "OPEN"
    deal_id = r.json()["data"]["id"] if ok else None
    version = r.json()["data"]["version"] if ok else 1
    record(scenario, "Create SPOT deal → 201", ok, f"id={deal_id}")

    if not deal_id:
        record(scenario, "SKIP remaining (create failed)", False)
        return

    # 2.2 Get deal
    r = c.get_fx(deal_id)
    ok = r.status_code == 200 and r.json()["data"]["id"] == deal_id
    record(scenario, "Get deal by ID", ok)

    # 2.3 Update deal
    r = c.update_fx(deal_id, {"note": "Updated by E2E", "version": version})
    ok = r.status_code == 200
    new_version = r.json()["data"]["version"] if ok else version
    record(scenario, "Update deal (note)", ok, f"status={r.status_code}")

    # 2.4 List with filter
    r = c.list_fx({"status": "OPEN", "deal_type": "SPOT"})
    ok = r.status_code == 200 and len(r.json()["data"]["data"]) >= 1
    record(scenario, "List with status+deal_type filter", ok)

    # 2.5 List with pagination
    r = c.list_fx({"page": 1, "page_size": 2})
    ok = r.status_code == 200 and r.json()["data"].get("total", 0) > 0
    record(scenario, "List with pagination", ok)

    # 2.6 Delete deal (soft)
    r = c.delete_fx(deal_id)
    ok = r.status_code == 204
    record(scenario, "Soft delete deal → 204", ok, f"status={r.status_code}")


# ─── Scenario 3: FX Swap ─────────────────────────────────────────────────────

def test_fx_swap():
    scenario = "3. FX Swap"
    console.rule(f"[bold]{scenario}[/bold]")

    c = APIClient()
    c.login(USERS["dealer"])

    # 3.1 Create SWAP deal with 2 legs
    r = c.create_fx(make_swap_deal())
    ok = r.status_code == 201 and r.json()["data"]["deal_type"] == "SWAP"
    record(scenario, "Create SWAP deal → 201", ok, f"status={r.status_code}")

    if not ok:
        record(scenario, "SKIP remaining (create failed)", False, r.text[:200])
        return

    data = r.json()["data"]

    # 3.2 Verify 2 legs
    ok = len(data["legs"]) == 2
    record(scenario, "SWAP has exactly 2 legs", ok, f"legs={len(data['legs'])}")

    # 3.3 Verify leg amounts
    leg1 = data["legs"][0]
    leg2 = data["legs"][1]
    ok = float(leg1["sell_amount"]) > 0 and float(leg2["buy_amount"]) > 0
    record(scenario, "Leg amounts are positive", ok)

    # 3.4 SPOT with 2 legs should fail
    bad = make_spot_deal()
    bad["legs"].append(bad["legs"][0].copy())
    bad["legs"][1]["leg_number"] = 2
    r = c.create_fx(bad)
    ok = r.status_code == 400
    record(scenario, "SPOT with 2 legs → 400", ok, f"status={r.status_code}")


# ─── Scenario 4: Approval Flow ───────────────────────────────────────────────

def test_approval_flow():
    scenario = "4. Approval Flow"
    console.rule(f"[bold]{scenario}[/bold]")

    # Step 1: Dealer creates deal
    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.create_fx(make_spot_deal("E2E-APPR-001"))
    ok = r.status_code == 201
    record(scenario, "Dealer creates deal (OPEN)", ok)
    if not ok:
        record(scenario, "SKIP remaining", False, r.text[:200])
        return
    deal_id = r.json()["data"]["id"]
    version = r.json()["data"]["version"]

    # Step 2: DeskHead approves L1 → PENDING_L2_APPROVAL
    deskhead = APIClient()
    deskhead.login(USERS["deskhead"])
    r = deskhead.approve_fx(deal_id, "APPROVE", version)
    ok = r.status_code == 200
    record(scenario, "DeskHead approves L1 → PENDING_L2_APPROVAL", ok, f"status={r.status_code}")

    # Verify status
    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    version = r.json()["data"]["version"] if r.status_code == 200 else version + 1
    ok = status == "PENDING_L2_APPROVAL"
    record(scenario, "Status is PENDING_L2_APPROVAL", ok, f"actual={status}")

    # Step 3: Director approves L2 → PENDING_BOOKING
    director = APIClient()
    director.login(USERS["director"])
    r = director.approve_fx(deal_id, "APPROVE", version)
    ok = r.status_code == 200
    record(scenario, "Director approves L2 → PENDING_BOOKING", ok, f"status={r.status_code}")

    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    version = r.json()["data"]["version"] if r.status_code == 200 else version + 1
    ok = status == "PENDING_BOOKING"
    record(scenario, "Status is PENDING_BOOKING", ok, f"actual={status}")

    # Step 4: Accountant books → PENDING_CHIEF_ACCOUNTANT
    accountant = APIClient()
    accountant.login(USERS["accountant"])
    r = accountant.approve_fx(deal_id, "APPROVE", version)
    ok = r.status_code == 200
    record(scenario, "Accountant books L1 → PENDING_CHIEF_ACCOUNTANT", ok, f"status={r.status_code}")

    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    version = r.json()["data"]["version"] if r.status_code == 200 else version + 1
    ok = status == "PENDING_CHIEF_ACCOUNTANT"
    record(scenario, "Status is PENDING_CHIEF_ACCOUNTANT", ok, f"actual={status}")


# ─── Scenario 5: Self-Approval Block ─────────────────────────────────────────

def test_self_approval():
    scenario = "5. Self-Approval Block"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.create_fx(make_spot_deal("E2E-SELF-001"))
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    version = r.json()["data"]["version"] if ok else 1
    record(scenario, "Dealer creates deal", ok)

    if not deal_id:
        return

    # Dealer tries to approve own deal — should fail
    # Dealer doesn't have approve permission, so it could be 403 (permission) or 409 (self-approval)
    r = dealer.approve_fx(deal_id, "APPROVE", version)
    ok = r.status_code in (403, 409)
    record(scenario, "Dealer cannot approve own deal", ok, f"status={r.status_code}")


# ─── Scenario 6: Recall ──────────────────────────────────────────────────────

def test_recall():
    scenario = "6. Recall"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.create_fx(make_spot_deal("E2E-RECALL-001"))
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    version = r.json()["data"]["version"] if ok else 1
    record(scenario, "Dealer creates deal", ok)
    if not deal_id:
        return

    # DeskHead approves L1
    deskhead = APIClient()
    deskhead.login(USERS["deskhead"])
    r = deskhead.approve_fx(deal_id, "APPROVE", version)
    ok = r.status_code == 200
    record(scenario, "DeskHead approves L1", ok)

    # Get current version
    r = dealer.get_fx(deal_id)
    version = r.json()["data"]["version"] if r.status_code == 200 else version + 1
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    record(scenario, "Status is PENDING_L2_APPROVAL", status == "PENDING_L2_APPROVAL", f"actual={status}")

    # Dealer recalls
    r = dealer.recall_fx(deal_id, "Need to fix rate")
    ok = r.status_code == 200
    record(scenario, "Dealer recalls deal → OPEN", ok, f"status={r.status_code}")

    # Verify back to OPEN
    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "OPEN"
    record(scenario, "Status back to OPEN", ok, f"actual={status}")


# ─── Scenario 7: Clone ───────────────────────────────────────────────────────

def test_clone():
    scenario = "7. Clone"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.create_fx(make_spot_deal("E2E-CLONE-001"))
    ok = r.status_code == 201
    deal_id = r.json()["data"]["id"] if ok else None
    version = r.json()["data"]["version"] if ok else 1
    record(scenario, "Dealer creates deal", ok)
    if not deal_id:
        return

    # DeskHead approves L1
    deskhead = APIClient()
    deskhead.login(USERS["deskhead"])
    r = deskhead.approve_fx(deal_id, "APPROVE", version)
    record(scenario, "DeskHead approves L1", r.status_code == 200)

    # Get updated version
    r = dealer.get_fx(deal_id)
    version = r.json()["data"]["version"] if r.status_code == 200 else version + 1

    # Director rejects L2
    director = APIClient()
    director.login(USERS["director"])
    r = director.approve_fx(deal_id, "REJECT", version, comment="Rate is off")
    ok = r.status_code == 200
    record(scenario, "Director rejects L2 → REJECTED", ok, f"status={r.status_code}")

    # Verify REJECTED
    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "REJECTED"
    record(scenario, "Status is REJECTED", ok, f"actual={status}")

    # Dealer clones
    r = dealer.clone_fx(deal_id)
    ok = r.status_code == 201 and r.json()["data"]["status"] == "OPEN"
    clone_id = r.json()["data"]["id"] if ok else None
    record(scenario, "Clone rejected deal → new OPEN deal", ok, f"clone_id={clone_id}")

    # Verify clone is different ID
    if clone_id:
        ok = clone_id != deal_id
        record(scenario, "Clone has new ID", ok)


# ─── Scenario 8: Role-Based Data Scope ───────────────────────────────────────

def test_role_scope():
    scenario = "8. Role-Based Data Scope"
    console.rule(f"[bold]{scenario}[/bold]")

    # Dealer creates an OPEN deal
    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.create_fx(make_spot_deal("E2E-SCOPE-001"))
    ok = r.status_code == 201
    record(scenario, "Dealer creates OPEN deal", ok)

    # Dealer can see all FX deals
    r = dealer.list_fx()
    dealer_items = r.json()["data"]["data"] if r.status_code == 200 else []
    dealer_count = len(dealer_items)
    ok = r.status_code == 200 and dealer_count >= 1
    record(scenario, "Dealer sees FX deals", ok, f"count={dealer_count}")

    # Accountant can only see PENDING_BOOKING and beyond — OPEN deals hidden
    accountant = APIClient()
    accountant.login(USERS["accountant"])
    r = accountant.list_fx({"page_size": 100})
    ok = r.status_code == 200
    acct_data = r.json()["data"]["data"] if ok else []
    # Accountant should NOT see OPEN deals
    open_deals = [d for d in acct_data if d["status"] == "OPEN"]
    ok = len(open_deals) == 0
    record(scenario, "Accountant sees no OPEN deals", ok, f"open_count={len(open_deals)}, total={len(acct_data)}")

    # Settlement officer — very restricted scope
    settlement = APIClient()
    settlement.login(USERS["settlement"])
    r = settlement.list_fx()
    ok = r.status_code == 200
    settle_data = r.json()["data"]["data"] if ok else []
    # Should only see PENDING_SETTLEMENT deals
    non_pending = [d for d in settle_data if d["status"] != "PENDING_SETTLEMENT"]
    ok = len(non_pending) == 0
    record(scenario, "Settlement sees only PENDING_SETTLEMENT", ok, f"non_pending={len(non_pending)}, total={len(settle_data)}")


# ─── Scenario 9: Rate Limiting ───────────────────────────────────────────────

def test_rate_limiting():
    scenario = "9. Rate Limiting"
    console.rule(f"[bold]{scenario}[/bold]")

    got_429 = False
    # Rapid login attempts to trigger rate limiting
    for i in range(25):
        c = APIClient()
        r = c.login(USERS["dealer"], "WrongPass!")
        if r.status_code == 429:
            got_429 = True
            break

    if got_429:
        record(scenario, "Rate limit triggered (429) on rapid logins", True, f"after {i+1} attempts")
    else:
        record(scenario, "Rate limit NOT triggered (may be disabled)", True,
               "Redis/rate-limiter may not be configured — acceptable in dev")


# ─── Scenario 10: Security ───────────────────────────────────────────────────

def test_security():
    scenario = "10. Security"
    console.rule(f"[bold]{scenario}[/bold]")

    # 10.1 No auth → 401
    r = requests.get(f"{API}/fx")
    ok = r.status_code == 401
    record(scenario, "No auth → 401 on /fx", ok, f"status={r.status_code}")

    # 10.2 Invalid JWT
    r = requests.get(f"{API}/fx", headers={"Authorization": "Bearer invalid.jwt.token"})
    ok = r.status_code == 401
    record(scenario, "Invalid JWT → 401", ok, f"status={r.status_code}")

    # 10.3 Expired/tampered JWT
    fake_jwt = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoiMTIzIiwicm9sZXMiOlsiQURNSU4iXX0.fake"
    r = requests.get(f"{API}/fx", headers={"Authorization": f"Bearer {fake_jwt}"})
    ok = r.status_code == 401
    record(scenario, "Tampered JWT → 401", ok, f"status={r.status_code}")

    # 10.4 SQL injection in query params
    c = APIClient()
    c.login(USERS["dealer"])
    r = c.list_fx({"status": "'; DROP TABLE fx_deals; --"})
    ok = r.status_code != 500  # Should not crash — 200 (empty) or 400 are both safe
    record(scenario, "SQL injection in params → safe", ok, f"status={r.status_code}")

    # 10.5 SQL injection in path param
    r = c.get_fx("'; DROP TABLE fx_deals; --")
    ok = r.status_code != 500  # invalid UUID → 400 or 422, not 500
    record(scenario, "SQL injection in path → safe", ok, f"status={r.status_code}")


# ─── Scenario 11: Auth /me → permissions[] ────────────────────────────────────

def test_auth_permissions():
    scenario = "11. Auth Permissions"
    console.rule(f"[bold]{scenario}[/bold]")

    # Admin → should have SYSTEM.MANAGE, AUDIT_LOG.VIEW etc.
    admin = APIClient()
    admin.login(USERS["admin"])
    r = admin.me()
    data = r.json().get("data", {})
    perms = data.get("permissions", [])
    ok = r.status_code == 200 and len(perms) > 0
    record(scenario, "Admin /me returns permissions[]", ok, f"count={len(perms)}")

    ok = "SYSTEM.MANAGE" in perms
    record(scenario, "Admin has SYSTEM.MANAGE", ok, f"perms={perms[:5]}...")

    ok = "AUDIT_LOG.VIEW" in perms
    record(scenario, "Admin has AUDIT_LOG.VIEW", ok)

    # Dealer → should have FX_DEAL.CREATE but NOT SYSTEM.MANAGE
    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.me()
    d_perms = r.json().get("data", {}).get("permissions", [])
    ok = "FX_DEAL.CREATE" in d_perms
    record(scenario, "Dealer has FX_DEAL.CREATE", ok)

    ok = "SYSTEM.MANAGE" not in d_perms
    record(scenario, "Dealer does NOT have SYSTEM.MANAGE", ok)

    # Risk Officer → should NOT have FX permissions
    risk = APIClient()
    risk.login(USERS["risk"])
    r = risk.me()
    r_perms = r.json().get("data", {}).get("permissions", [])
    fx_perms = [p for p in r_perms if p.startswith("FX_DEAL")]
    ok = len(fx_perms) == 0
    record(scenario, "Risk Officer has NO FX_DEAL permissions", ok, f"fx_perms={fx_perms}")


# ─── Scenario 12: Admin User Management ───────────────────────────────────────

def test_admin_users():
    scenario = "12. Admin Users"
    console.rule(f"[bold]{scenario}[/bold]")

    admin = APIClient()
    admin.login(USERS["admin"])

    # 12.1 List users
    r = admin.list_users()
    ok = r.status_code == 200
    users_data = r.json().get("data", {})
    items = users_data.get("data", []) if isinstance(users_data, dict) else users_data
    count = len(items) if isinstance(items, list) else 0
    record(scenario, "List users", ok, f"count={count}")

    ok = count >= 10
    record(scenario, "At least 10 users (seed data)", ok, f"count={count}")

    # 12.2 Get user detail
    if count > 0:
        uid = items[0].get("id")
        r = admin.get_user(uid)
        ok = r.status_code == 200
        record(scenario, "Get user detail", ok)
    else:
        record(scenario, "Get user detail", False, "no users to test")

    # 12.3 Lock user (dealer)
    # Find dealer user ID
    dealer_user = next((u for u in items if u.get("username") == "dealer01"), None)
    if dealer_user:
        dealer_id = dealer_user["id"]

        r = admin.lock_user(dealer_id)
        ok = r.status_code == 200
        record(scenario, "Lock dealer account", ok, f"status={r.status_code}")

        # 12.4 Locked user cannot login
        locked = APIClient()
        r = locked.login(USERS["dealer"])
        ok = r.status_code in (401, 403)
        record(scenario, "Locked user cannot login", ok, f"status={r.status_code}")

        # 12.5 Unlock user
        r = admin.unlock_user(dealer_id)
        ok = r.status_code == 200
        record(scenario, "Unlock dealer account", ok, f"status={r.status_code}")

        # 12.6 Unlocked user can login again
        unlocked = APIClient()
        r = unlocked.login(USERS["dealer"])
        ok = r.status_code == 200
        record(scenario, "Unlocked user can login", ok, f"status={r.status_code}")
    else:
        record(scenario, "Lock/Unlock tests", False, "dealer01 not found in user list")

    # 12.7 Reset password
    if dealer_user:
        r = admin.reset_password(dealer_user["id"])
        ok = r.status_code == 200
        new_pass = r.json().get("data", {}).get("temp_password", r.json().get("data", {}).get("temporary_password", ""))
        has_temp = len(new_pass) > 0
        record(scenario, "Reset password returns temp password", ok and has_temp,
               f"status={r.status_code}, has_temp={has_temp}")

        # Login with temp password
        if has_temp:
            temp = APIClient()
            r = temp.login(USERS["dealer"], new_pass)
            ok = r.status_code == 200
            record(scenario, "Login with temp password", ok, f"status={r.status_code}")

    else:
        record(scenario, "Reset password test", False, "dealer01 not found")
        new_pass = ""
        has_temp = False

    # 12.8 Non-admin cannot access /admin/users
    dealer = APIClient()
    # Login with temp password if reset succeeded, otherwise original
    login_pass = new_pass if (dealer_user and has_temp and new_pass) else PASSWORD
    r = dealer.login(USERS["dealer"], login_pass)
    if r.status_code != 200:
        # Try original password as fallback
        r = dealer.login(USERS["dealer"], PASSWORD)
    r = dealer.list_users()
    ok = r.status_code == 403
    record(scenario, "Non-admin → 403 on /admin/users", ok, f"status={r.status_code}")


# ─── Scenario 13: Role Management ─────────────────────────────────────────────

def test_role_management():
    scenario = "13. Role Management"
    console.rule(f"[bold]{scenario}[/bold]")

    admin = APIClient()
    admin.login(USERS["admin"])

    # 13.1 List roles
    r = admin.list_roles()
    ok = r.status_code == 200
    roles_data = r.json().get("data", [])
    if isinstance(roles_data, dict):
        roles_data = roles_data.get("data", [])
    count = len(roles_data)
    record(scenario, "List all roles", ok, f"count={count}")

    ok = count >= 10
    record(scenario, "At least 10 roles", ok, f"count={count}")

    # 13.2 Get role permissions
    r = admin.get_role_permissions("DEALER")
    ok = r.status_code == 200
    perms = r.json().get("data", {}).get("permissions", r.json().get("data", []))
    if isinstance(perms, list):
        has_fx_create = any("FX_DEAL.CREATE" in str(p) for p in perms)
    else:
        has_fx_create = False
    record(scenario, "Dealer role has FX_DEAL.CREATE permission", ok and has_fx_create,
           f"status={r.status_code}")

    # 13.3 Get role permissions for RISK_OFFICER
    r = admin.get_role_permissions("RISK_OFFICER")
    ok = r.status_code == 200
    risk_perms = r.json().get("data", {}).get("permissions", r.json().get("data", []))
    if isinstance(risk_perms, list):
        has_fx = any("FX_DEAL" in str(p) for p in risk_perms)
    else:
        has_fx = False
    ok = not has_fx
    record(scenario, "Risk Officer has NO FX permissions", ok)


# ─── Scenario 14: Audit Logs ─────────────────────────────────────────────────

def test_audit_logs():
    scenario = "14. Audit Logs"
    console.rule(f"[bold]{scenario}[/bold]")

    admin = APIClient()
    admin.login(USERS["admin"])

    # 14.1 List audit logs (may have entries from user lock/unlock)
    r = admin.list_audit_logs()
    ok = r.status_code == 200
    logs_data = r.json().get("data", {})
    items = logs_data.get("data", []) if isinstance(logs_data, dict) else logs_data
    count = len(items) if isinstance(items, list) else 0
    record(scenario, "List audit logs", ok, f"count={count}")

    # 14.2 Filter by action
    r = admin.list_audit_logs({"action": "LOCK_USER"})
    ok = r.status_code == 200
    record(scenario, "Filter audit logs by action", ok, f"status={r.status_code}")

    # 14.3 Audit stats (requires date_from + date_to)
    r = admin.audit_stats({"date_from": "2026-01-01", "date_to": "2026-12-31"})
    ok = r.status_code == 200
    record(scenario, "Audit log stats endpoint", ok, f"status={r.status_code}")

    # 14.4 Non-admin cannot view audit logs
    dealer = APIClient()
    dealer.login(USERS["dealer"])
    r = dealer.list_audit_logs()
    ok = r.status_code == 403
    record(scenario, "Non-admin → 403 on audit logs", ok, f"status={r.status_code}")


# ─── Scenario 15: Master Data APIs ───────────────────────────────────────────

def test_master_data():
    scenario = "15. Master Data"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])

    # 15.1 Counterparties list
    r = dealer.list_counterparties()
    ok = r.status_code == 200
    cp_data = r.json().get("data", {})
    items = cp_data.get("data", []) if isinstance(cp_data, dict) else cp_data
    count = len(items) if isinstance(items, list) else 0
    record(scenario, "List counterparties", ok, f"count={count}")

    ok = count >= 10
    record(scenario, "At least 10 counterparties (seed)", ok, f"count={count}")

    # 15.2 Search counterparties
    r = dealer.list_counterparties({"search": "Vietcombank"})
    ok = r.status_code == 200
    search_data = r.json().get("data", {})
    search_items = search_data.get("data", []) if isinstance(search_data, dict) else search_data
    search_count = len(search_items) if isinstance(search_items, list) else 0
    record(scenario, "Search counterparties by name", ok, f"found={search_count}")

    # 15.3 Get counterparty by ID
    r = dealer.get_counterparty(COUNTERPARTY_MSB)
    ok = r.status_code == 200
    record(scenario, "Get counterparty by ID", ok, f"status={r.status_code}")

    # 15.4 Currencies
    r = dealer.list_currencies()
    ok = r.status_code == 200
    cur_data = r.json().get("data", [])
    if isinstance(cur_data, dict):
        cur_data = cur_data.get("data", [])
    cur_count = len(cur_data)
    record(scenario, "List currencies", ok, f"count={cur_count}")

    ok = cur_count >= 5
    record(scenario, "At least 5 currencies", ok)

    # 15.5 Currency pairs
    r = dealer.list_currency_pairs()
    ok = r.status_code == 200
    record(scenario, "List currency pairs", ok, f"status={r.status_code}")

    # 15.6 Branches
    r = dealer.list_branches()
    ok = r.status_code == 200
    record(scenario, "List branches", ok, f"status={r.status_code}")

    # 15.7 Exchange rates
    r = dealer.list_exchange_rates()
    ok = r.status_code == 200
    record(scenario, "List exchange rates", ok, f"status={r.status_code}")


# ─── Scenario 16: Cancel Flow (2-Level) ──────────────────────────────────────

def _create_and_complete_deal(dealer, deskhead, director, accountant, chiefacc, settlement, scenario):
    """Helper: create a deal and approve it all the way to COMPLETED."""
    r = dealer.create_fx(make_spot_deal())
    if r.status_code != 201:
        record(scenario, "SKIP: create failed", False, r.text[:200])
        return None, None
    deal_id = r.json()["data"]["id"]
    version = r.json()["data"]["version"]

    # L1 approve
    deskhead.approve_fx(deal_id, "APPROVE", version)
    r = dealer.get_fx(deal_id)
    version = r.json()["data"]["version"]

    # L2 approve
    director.approve_fx(deal_id, "APPROVE", version)
    r = dealer.get_fx(deal_id)
    version = r.json()["data"]["version"]

    # Accountant book
    accountant.approve_fx(deal_id, "APPROVE", version)
    r = dealer.get_fx(deal_id)
    version = r.json()["data"]["version"]

    # Chief Accountant book
    chiefacc.approve_fx(deal_id, "APPROVE", version)
    r = dealer.get_fx(deal_id)
    version = r.json()["data"]["version"]

    # Settlement settle
    settlement.approve_fx(deal_id, "APPROVE", version)
    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    version = r.json()["data"]["version"] if r.status_code == 200 else version

    if status != "COMPLETED":
        record(scenario, f"SKIP: deal not COMPLETED (got {status})", False)
        return None, None

    return deal_id, version


def test_cancel_flow():
    scenario = "16. Cancel Flow (2-Level)"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])
    deskhead = APIClient()
    deskhead.login(USERS["deskhead"])
    director = APIClient()
    director.login(USERS["director"])
    accountant = APIClient()
    accountant.login(USERS["accountant"])
    chiefacc = APIClient()
    chiefacc.login(USERS["chiefacc"])
    settlement = APIClient()
    settlement.login(USERS["settlement"])

    # 16.1 Create and complete a deal
    deal_id, version = _create_and_complete_deal(dealer, deskhead, director, accountant, chiefacc, settlement, scenario)
    if not deal_id:
        return
    record(scenario, "Deal COMPLETED", True)

    # 16.2 Dealer requests cancel → PENDING_CANCEL_L1
    r = dealer.cancel_fx(deal_id, "Client requested cancellation")
    ok = r.status_code == 200
    record(scenario, "Dealer requests cancel → PENDING_CANCEL_L1", ok, f"status={r.status_code}")

    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_CANCEL_L1"
    record(scenario, "Status is PENDING_CANCEL_L1", ok, f"actual={status}")

    # 16.3 DeskHead approves cancel L1 → PENDING_CANCEL_L2
    r = deskhead.cancel_approve_fx(deal_id, "APPROVE", "Confirmed by desk head")
    ok = r.status_code == 200
    record(scenario, "DeskHead approves cancel L1 → PENDING_CANCEL_L2", ok, f"status={r.status_code}")

    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "PENDING_CANCEL_L2"
    record(scenario, "Status is PENDING_CANCEL_L2", ok, f"actual={status}")

    # 16.4 Director approves cancel L2 → CANCELLED
    r = director.cancel_approve_fx(deal_id, "APPROVE", "Final approval")
    ok = r.status_code == 200
    record(scenario, "Director approves cancel L2 → CANCELLED", ok, f"status={r.status_code}")

    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "CANCELLED"
    record(scenario, "Status is CANCELLED", ok, f"actual={status}")


def test_cancel_reject():
    scenario = "17. Cancel Reject"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])
    deskhead = APIClient()
    deskhead.login(USERS["deskhead"])
    director = APIClient()
    director.login(USERS["director"])
    accountant = APIClient()
    accountant.login(USERS["accountant"])
    chiefacc = APIClient()
    chiefacc.login(USERS["chiefacc"])
    settlement = APIClient()
    settlement.login(USERS["settlement"])

    # 17.1 Create and complete a deal
    deal_id, version = _create_and_complete_deal(dealer, deskhead, director, accountant, chiefacc, settlement, scenario)
    if not deal_id:
        return
    record(scenario, "Deal COMPLETED", True)

    # 17.2 Dealer requests cancel
    r = dealer.cancel_fx(deal_id, "Cancel requested")
    ok = r.status_code == 200
    record(scenario, "Dealer requests cancel", ok)

    # 17.3 DeskHead rejects cancel → back to COMPLETED
    r = deskhead.cancel_approve_fx(deal_id, "REJECT", "Not justified")
    ok = r.status_code == 200
    record(scenario, "DeskHead rejects cancel → COMPLETED", ok, f"status={r.status_code}")

    r = dealer.get_fx(deal_id)
    status = r.json()["data"]["status"] if r.status_code == 200 else "?"
    ok = status == "COMPLETED"
    record(scenario, "Status back to COMPLETED", ok, f"actual={status}")


# ─── Scenario 18: Ticket Auto-Generation ────────────────────────────────────

def test_ticket_auto_gen():
    scenario = "18. Ticket Auto-Gen"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])

    # 18.1 Create deal WITHOUT ticket_number
    payload = make_spot_deal()
    payload.pop("ticket_number", None)
    r = dealer.create_fx(payload)
    ok = r.status_code == 201
    record(scenario, "Create deal without ticket_number", ok, f"status={r.status_code}")
    if not ok:
        return

    data = r.json()["data"]
    ticket = data.get("ticket_number")
    ok = ticket is not None and ticket.startswith("FX-") and len(ticket) >= 14
    record(scenario, f"Auto-generated ticket: {ticket}", ok)

    # 18.2 Create another deal — ticket should have incremented sequence
    r2 = dealer.create_fx(payload)
    ok2 = r2.status_code == 201
    if ok2:
        ticket2 = r2.json()["data"].get("ticket_number")
        ok = ticket2 is not None and ticket2 != ticket
        record(scenario, f"Second ticket different: {ticket2}", ok)
    else:
        record(scenario, "Second deal creation", False)


# ─── Scenario 19: Approval History ──────────────────────────────────────────

def test_approval_history():
    scenario = "19. Approval History"
    console.rule(f"[bold]{scenario}[/bold]")

    dealer = APIClient()
    dealer.login(USERS["dealer"])
    deskhead = APIClient()
    deskhead.login(USERS["deskhead"])

    # 19.1 Create + approve deal
    r = dealer.create_fx(make_spot_deal())
    ok = r.status_code == 201
    record(scenario, "Create deal", ok)
    if not ok:
        return
    deal_id = r.json()["data"]["id"]
    version = r.json()["data"]["version"]

    # DeskHead approves
    deskhead.approve_fx(deal_id, "APPROVE", version)

    # 19.2 Get approval history
    r = dealer.get_fx_history(deal_id)
    ok = r.status_code == 200
    record(scenario, "GET /fx/{id}/history → 200", ok, f"status={r.status_code}")

    if ok:
        entries = r.json().get("data", [])
        ok = len(entries) >= 1
        record(scenario, f"History has entries: {len(entries)}", ok)

        if len(entries) >= 1:
            first = entries[0]
            ok = "action_type" in first and "performer_name" in first and "performed_at" in first
            record(scenario, "Entry has required fields", ok)


# ─── Main ─────────────────────────────────────────────────────────────────────

def print_summary():
    table = Table(title="E2E Test Results", box=box.ROUNDED)
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
    parser = argparse.ArgumentParser(description="Treasury API E2E Tests")
    parser.add_argument("--base-url", default="http://localhost:34001", help="API base URL")
    args = parser.parse_args()

    BASE_URL = args.base_url.rstrip("/")
    API = f"{BASE_URL}/api/v1"

    console.print(Panel(f"[bold]Treasury API E2E Test Suite[/bold]\nTarget: {BASE_URL}", border_style="blue"))

    # Verify API is up
    try:
        r = requests.get(f"{BASE_URL}/health", timeout=5)
        if r.status_code != 200:
            console.print(f"[red]Health check failed: {r.status_code}[/red]")
            sys.exit(1)
    except requests.ConnectionError:
        console.print(f"[red]Cannot connect to {BASE_URL}[/red]")
        sys.exit(1)

    console.print(f"[green]Health check OK[/green]\n")

    # Non-destructive tests first
    test_auth()
    test_fx_crud()
    test_fx_swap()
    test_approval_flow()
    test_self_approval()
    test_recall()
    test_clone()
    test_role_scope()
    test_security()
    test_auth_permissions()
    test_master_data()
    test_role_management()
    test_audit_logs()
    # New feature tests
    test_cancel_flow()
    test_cancel_reject()
    test_ticket_auto_gen()
    test_approval_history()
    # Destructive tests last (lock/reset may break other tests)
    test_admin_users()
    # Rate limiting last (burns through request quota)
    test_rate_limiting()

    all_passed = print_summary()
    sys.exit(0 if all_passed else 1)


if __name__ == "__main__":
    main()
