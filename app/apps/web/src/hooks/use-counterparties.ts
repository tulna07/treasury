"use client";

import { useFxDeals } from "./use-fx";

export interface Counterparty {
  id: string;
  code: string;
  name: string;
}

const SEED_COUNTERPARTIES: Counterparty[] = [
  { id: "e0000000-0000-0000-0000-000000000001", code: "MSB", name: "Maritime Bank" },
  { id: "e0000000-0000-0000-0000-000000000002", code: "ACB", name: "ACB" },
  { id: "e0000000-0000-0000-0000-000000000003", code: "VCB", name: "Vietcombank" },
  { id: "e0000000-0000-0000-0000-000000000004", code: "TCB", name: "Techcombank" },
  { id: "e0000000-0000-0000-0000-000000000005", code: "VPB", name: "VPBank" },
  { id: "e0000000-0000-0000-0000-000000000006", code: "MBB", name: "MB Bank" },
  { id: "e0000000-0000-0000-0000-000000000007", code: "BID", name: "BIDV" },
  { id: "e0000000-0000-0000-0000-000000000008", code: "CTG", name: "VietinBank" },
  { id: "e0000000-0000-0000-0000-000000000009", code: "STB", name: "Sacombank" },
  { id: "e0000000-0000-0000-0000-000000000010", code: "SHB", name: "SHB" },
];

export function useCounterparties(): Counterparty[] {
  const { data } = useFxDeals({ page_size: 100 });

  if (!data?.data?.length) return SEED_COUNTERPARTIES;

  const map = new Map<string, Counterparty>();
  for (const deal of data.data) {
    if (deal.counterparty_id && !map.has(deal.counterparty_id)) {
      map.set(deal.counterparty_id, {
        id: deal.counterparty_id,
        code: deal.counterparty_code,
        name: deal.counterparty_name,
      });
    }
  }

  const fromDeals = Array.from(map.values());
  if (fromDeals.length >= 5) return fromDeals;

  // Merge: deals first, then fill from seed
  const merged = new Map(SEED_COUNTERPARTIES.map((cp) => [cp.id, cp]));
  for (const cp of fromDeals) merged.set(cp.id, cp);
  return Array.from(merged.values());
}
