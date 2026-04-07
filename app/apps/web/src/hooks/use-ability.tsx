"use client";

import * as React from "react";
import { createContextualCan } from "@casl/react";
import {
  defineAbilityFor,
  type AppAbility,
  type Actions,
  type Subjects,
} from "@/lib/ability";
import { createMongoAbility } from "@casl/ability";
import { useAuthStore } from "@/lib/auth-store";

// ─── Context ─────────────────────────────────────────────────
const AbilityContext = React.createContext<AppAbility>(
  createMongoAbility<[Actions, Subjects]>()
);

// ─── CASL <Can> component ────────────────────────────────────
export const Can = createContextualCan(AbilityContext.Consumer);

// ─── Hook ────────────────────────────────────────────────────
export function useAbility(): AppAbility {
  return React.useContext(AbilityContext);
}

// ─── Provider ────────────────────────────────────────────────
export function AbilityProvider({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user);

  const ability = React.useMemo(() => {
    const permissions = user?.permissions ?? [];
    return defineAbilityFor(permissions);
  }, [user?.permissions]);

  return (
    <AbilityContext.Provider value={ability}>
      {children}
    </AbilityContext.Provider>
  );
}
