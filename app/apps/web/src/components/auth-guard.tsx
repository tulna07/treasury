"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { useAuthStore } from "@/lib/auth-store";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const fetchUser = useAuthStore((s) => s.fetchUser);
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    // On mount: if already authenticated (sessionStorage), skip the network call.
    // Otherwise validate the server session once.
    if (useAuthStore.getState().isAuthenticated) {
      setChecked(true);
    } else {
      fetchUser().finally(() => setChecked(true));
    }
  }, [fetchUser]);

  useEffect(() => {
    if (checked && !isAuthenticated) {
      router.replace("/login");
    }
  }, [checked, isAuthenticated, router]);

  if (!checked || !isAuthenticated) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="size-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    );
  }

  return <>{children}</>;
}
