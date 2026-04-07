"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Image from "next/image";
import {
  IconUser,
  IconLock,
  IconEye,
  IconEyeOff,
  IconShield,
  IconCash,
  IconChartBar,
  IconWorld,
  IconSettings,
  IconFileInvoice,
  IconCurrencyDollar,
  IconLoader2,
  IconUserCog,
  IconScale,
  IconBuildingBank,
} from "@tabler/icons-react";
import { toast } from "sonner";
import { useAuthStore } from "@/lib/auth-store";
import { DEV_ACCOUNTS, type DevAccount } from "@/lib/mock-users";

const DEV_ICONS: Record<string, React.ElementType> = {
  dealer01: IconCurrencyDollar,
  deskhead01: IconFileInvoice,
  director01: IconChartBar,
  divhead01: IconBuildingBank,
  risk01: IconShield,
  riskhead01: IconScale,
  accountant01: IconCash,
  chiefacc01: IconUserCog,
  settlement01: IconWorld,
  admin01: IconSettings,
};

export default function LoginPage() {
  const router = useRouter();
  const login = useAuthStore((s) => s.login);
  const isLoading = useAuthStore((s) => s.isLoading);
  const [showPassword, setShowPassword] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  const handleLogin = async (user: string, pass: string) => {
    try {
      await login(user, pass);
      router.push("/");
    } catch (err: unknown) {
      const message =
        err && typeof err === "object" && "error" in err
          ? (err as { error: string }).error
          : "Login failed. Please check your credentials.";
      toast.error(message);
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!username || !password) return;
    handleLogin(username, password);
  };

  const handleDevLogin = (account: DevAccount) => {
    setUsername(account.username);
    setPassword(account.password);
    handleLogin(account.username, account.password);
  };

  return (
    <div className="relative flex min-h-screen items-center justify-center bg-gradient-to-br from-background via-background to-muted px-4 py-8">
      {/* Subtle grid pattern */}
      <div
        className="fixed inset-0 opacity-[0.03] dark:opacity-[0.04]"
        style={{
          backgroundImage:
            "radial-gradient(circle, hsl(var(--foreground)) 1px, transparent 1px)",
          backgroundSize: "32px 32px",
        }}
      />

      {/* Floating orbs — theme-aware */}
      <div className="fixed inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-[10%] left-[15%] size-64 rounded-full bg-[#EF7922]/[0.06] dark:bg-[#EF7922]/[0.03] blur-3xl animate-float-slow" />
        <div className="absolute top-[60%] right-[10%] size-80 rounded-full bg-[#30C2E3]/[0.06] dark:bg-[#30C2E3]/[0.03] blur-3xl animate-float-medium" />
        <div className="absolute bottom-[15%] left-[40%] size-56 rounded-full bg-[#EF7922]/[0.04] dark:bg-[#EF7922]/[0.02] blur-3xl animate-float-reverse" />
      </div>

      {/* Content */}
      <div className="relative z-10 flex w-full max-w-md flex-col items-center gap-8">
        {/* Logo & Branding */}
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="flex size-20 items-center justify-center rounded-2xl border border-[#EF7922]/20 bg-card/80 shadow-sm backdrop-blur-md">
            <Image
              src="/logo.svg"
              alt="KienlongBank"
              width={48}
              height={48}
              className="dark:invert drop-shadow-[0_0_8px_rgba(239,121,34,0.2)]"
            />
          </div>
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-foreground">
              Treasury Management
            </h1>
            <p className="mt-1 text-sm font-medium text-[#30C2E3] dark:text-[#30C2E3]">
              Hệ thống Quản trị Nguồn vốn — KienlongBank
            </p>
          </div>
        </div>

        {/* Login Card */}
        <div className="w-full rounded-2xl border bg-card/80 p-8 shadow-xl backdrop-blur-xl">
          {/* Gradient accent line */}
          <div className="mx-auto mb-6 h-1 w-16 rounded-full bg-gradient-to-r from-[#EF7922] to-[#30C2E3]" />

          <h2 className="mb-6 text-center text-lg font-semibold text-foreground">
            Đăng nhập
          </h2>

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <label htmlFor="username" className="text-sm font-medium text-muted-foreground">
                Username
              </label>
              <div className="relative">
                <IconUser className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground/50" />
                <input
                  id="username"
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="Enter username"
                  disabled={isLoading}
                  className="flex h-10 w-full rounded-lg border bg-background pl-10 pr-3 py-2 text-sm text-foreground placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[#EF7922]/50 focus-visible:border-[#EF7922]/30 transition-colors disabled:opacity-50"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <label htmlFor="password" className="text-sm font-medium text-muted-foreground">
                Mật khẩu
              </label>
              <div className="relative">
                <IconLock className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground/50" />
                <input
                  id="password"
                  type={showPassword ? "text" : "password"}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Nhập mật khẩu"
                  disabled={isLoading}
                  className="flex h-10 w-full rounded-lg border bg-background pl-10 pr-10 py-2 text-sm text-foreground placeholder:text-muted-foreground/40 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-[#EF7922]/50 focus-visible:border-[#EF7922]/30 transition-colors disabled:opacity-50"
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground/50 hover:text-muted-foreground transition-colors"
                >
                  {showPassword ? <IconEyeOff className="size-4" /> : <IconEye className="size-4" />}
                </button>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <input
                id="remember"
                type="checkbox"
                className="size-3.5 rounded border accent-[#EF7922]"
              />
              <label htmlFor="remember" className="text-xs text-muted-foreground">
                Ghi nhớ đăng nhập
              </label>
            </div>

            <button
              type="submit"
              disabled={isLoading || !username || !password}
              className="inline-flex h-10 w-full items-center justify-center rounded-lg bg-gradient-to-r from-[#EF4F25] via-[#EF7922] to-[#30C2E3] px-4 py-2 text-sm font-semibold text-white shadow-lg shadow-[#EF7922]/10 transition-all hover:shadow-xl hover:shadow-[#EF7922]/20 hover:brightness-110 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[#EF7922]/50 disabled:opacity-50 disabled:pointer-events-none"
            >
              {isLoading ? (
                <>
                  <IconLoader2 className="mr-2 size-4 animate-spin" />
                  Đang đăng nhập...
                </>
              ) : (
                "Đăng nhập"
              )}
            </button>
          </form>
        </div>

        {/* Dev Mode Panel */}
        <div className="w-full rounded-2xl border bg-card/60 p-6 backdrop-blur-lg">
          <div className="mb-4 flex items-center justify-center gap-3">
            <div className="h-px flex-1 bg-gradient-to-r from-transparent to-border" />
            <span className="text-[10px] font-medium uppercase tracking-widest text-muted-foreground/50">
              Dev Quick Login
            </span>
            <div className="h-px flex-1 bg-gradient-to-l from-transparent to-border" />
          </div>

          <div className="grid grid-cols-2 gap-2">
            {DEV_ACCOUNTS.map((account) => {
              const Icon = DEV_ICONS[account.username] ?? IconUser;
              return (
                <button
                  key={account.username}
                  onClick={() => handleDevLogin(account)}
                  disabled={isLoading}
                  className="group flex items-center gap-2.5 rounded-xl border bg-background/50 p-2.5 text-left transition-all hover:border-[#EF7922]/25 hover:bg-[#EF7922]/[0.06] disabled:opacity-50 disabled:pointer-events-none"
                >
                  <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-muted text-muted-foreground transition-colors group-hover:bg-[#EF7922]/10 group-hover:text-[#EF7922]">
                    <Icon className="size-4" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-xs font-medium leading-tight text-foreground/80 group-hover:text-foreground">
                      {account.label}
                    </p>
                    <p className="truncate text-[10px] text-muted-foreground/60">
                      {account.department}
                    </p>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        {/* Footer */}
        <p className="text-[11px] text-muted-foreground/40">
          &copy; 2026 KienlongBank. All rights reserved.
        </p>
      </div>
    </div>
  );
}
