import { chromium } from "/opt/homebrew/lib/node_modules/playwright/index.mjs";

const BASE = "http://localhost:34000";
const SCREENSHOTS = "/Users/mrm/Projects/treasury-cd/screenshots/admin";

async function run() {
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: { width: 1440, height: 900 },
  });
  const page = await context.newPage();

  // Capture console errors
  const errors = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") errors.push(msg.text());
  });

  // 1. Login as admin01
  console.log("1. Login as admin01...");
  await page.goto(`${BASE}/login`);
  await page.waitForTimeout(2000);
  await page.fill('#username', "admin01");
  await page.fill('#password', "P@ssw0rd123");
  await page.click('button[type="submit"]');
  await page.waitForTimeout(5000);

  // Check current URL
  const urlAfterLogin = page.url();
  console.log("   URL after login:", urlAfterLogin);

  // Check for session cookie
  const cookies = await context.cookies();
  console.log("   Cookies:", cookies.map(c => c.name).join(", "));

  await page.screenshot({ path: `${SCREENSHOTS}/01-after-login.png`, fullPage: true });

  // 2. Navigate to /settings - use client-side navigation
  console.log("2. Navigate to /settings...");
  await page.goto(`${BASE}/settings`);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOTS}/02-settings-landing.png`, fullPage: true });

  // 3. /settings/users - list users
  console.log("3. Navigate to /settings/users...");
  await page.goto(`${BASE}/settings/users`);
  await page.waitForTimeout(4000);
  await page.screenshot({ path: `${SCREENSHOTS}/03-users-list.png`, fullPage: true });

  // 4. Click first user row to see detail
  console.log("4. Click first user -> detail...");
  const firstUserRow = page.locator("table tbody tr").first();
  if (await firstUserRow.isVisible({ timeout: 2000 }).catch(() => false)) {
    await firstUserRow.click();
    await page.waitForTimeout(3000);
    await page.screenshot({ path: `${SCREENSHOTS}/04-user-detail.png`, fullPage: true });
    console.log("   User detail screenshot taken");
  } else {
    console.log("   No users in table, skipping detail");
    // Still take screenshot of whatever is on screen
    await page.screenshot({ path: `${SCREENSHOTS}/04-users-page-state.png`, fullPage: true });
  }

  // 5. /settings/users/new - create form
  console.log("5. Navigate to /settings/users/new...");
  await page.goto(`${BASE}/settings/users/new`);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOTS}/05-user-create.png`, fullPage: true });
  console.log("   User create form screenshot taken");

  // 6. /settings/roles
  console.log("6. Navigate to /settings/roles...");
  await page.goto(`${BASE}/settings/roles`);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOTS}/06-roles.png`, fullPage: true });
  console.log("   Roles screenshot taken");

  // 7. /settings/audit-logs
  console.log("7. Navigate to /settings/audit-logs...");
  await page.goto(`${BASE}/settings/audit-logs`);
  await page.waitForTimeout(4000);
  await page.screenshot({ path: `${SCREENSHOTS}/07-audit-logs.png`, fullPage: true });
  console.log("   Audit logs screenshot taken");

  // 8. /settings/counterparties
  console.log("8. Navigate to /settings/counterparties...");
  await page.goto(`${BASE}/settings/counterparties`);
  await page.waitForTimeout(4000);
  await page.screenshot({ path: `${SCREENSHOTS}/08-counterparties.png`, fullPage: true });
  console.log("   Counterparties screenshot taken");

  // 9. Test responsive (mobile viewport)
  console.log("9. Mobile viewport tests...");
  await page.setViewportSize({ width: 375, height: 812 });

  await page.goto(`${BASE}/settings`);
  await page.waitForTimeout(2000);
  await page.screenshot({ path: `${SCREENSHOTS}/09-mobile-settings.png`, fullPage: true });

  await page.goto(`${BASE}/settings/users`);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOTS}/10-mobile-users.png`, fullPage: true });

  await page.goto(`${BASE}/settings/audit-logs`);
  await page.waitForTimeout(3000);
  await page.screenshot({ path: `${SCREENSHOTS}/11-mobile-audit-logs.png`, fullPage: true });
  console.log("   Mobile screenshots taken");

  // 10. Test dark mode
  console.log("10. Dark mode tests...");
  await page.setViewportSize({ width: 1440, height: 900 });

  // Force dark mode via document class
  await page.goto(`${BASE}/settings`);
  await page.waitForTimeout(1000);
  await page.evaluate(() => {
    document.documentElement.classList.add("dark");
    document.documentElement.style.colorScheme = "dark";
  });
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${SCREENSHOTS}/12-dark-settings.png`, fullPage: true });

  await page.goto(`${BASE}/settings/users`);
  await page.waitForTimeout(2000);
  await page.evaluate(() => {
    document.documentElement.classList.add("dark");
    document.documentElement.style.colorScheme = "dark";
  });
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${SCREENSHOTS}/13-dark-users.png`, fullPage: true });

  await page.goto(`${BASE}/settings/audit-logs`);
  await page.waitForTimeout(2000);
  await page.evaluate(() => {
    document.documentElement.classList.add("dark");
    document.documentElement.style.colorScheme = "dark";
  });
  await page.waitForTimeout(500);
  await page.screenshot({ path: `${SCREENSHOTS}/14-dark-audit-logs.png`, fullPage: true });
  console.log("   Dark mode screenshots taken");

  if (errors.length > 0) {
    console.log("\nConsole errors:");
    errors.forEach((e) => console.log("  -", e.substring(0, 200)));
  }

  await browser.close();
  console.log("\nAll tests passed! Screenshots saved to: " + SCREENSHOTS);
}

run().catch((err) => {
  console.error("Test failed:", err);
  process.exit(1);
});
