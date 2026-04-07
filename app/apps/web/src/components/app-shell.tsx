"use client";

import { usePathname } from "next/navigation";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@/components/theme-provider";
import { I18nProvider } from "@/lib/i18n";
import { AuthGuard } from "@/components/auth-guard";
import { AbilityProvider } from "@/hooks/use-ability";
import { AppSidebar } from "@/components/app-sidebar";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { useState } from "react";

const PUBLIC_PATHS = ["/login"];

export function AppShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isPublic = PUBLIC_PATHS.some((p) => pathname.startsWith(p));

  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            retry: 1,
            refetchOnWindowFocus: false,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider
        attribute="class"
        defaultTheme="system"
        enableSystem
        disableTransitionOnChange
      >
        <I18nProvider>
          {isPublic ? (
            children
          ) : (
            <AuthGuard>
              <AbilityProvider>
                <TooltipProvider>
                  <SidebarProvider>
                    <AppSidebar />
                    <SidebarInset>{children}</SidebarInset>
                  </SidebarProvider>
                </TooltipProvider>
              </AbilityProvider>
            </AuthGuard>
          )}
        </I18nProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}
