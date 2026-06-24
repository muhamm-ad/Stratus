import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";
import { PROVIDER_META } from "@/lib/bridge";
import type { ProviderID } from "@/types/domain";
import { InventoryFilters } from "./InventoryFilters";
import { InventoryTable } from "./InventoryTable";
import { VMDetailDrawer } from "./VMDetailDrawer";

function useResultLabel() {
  const { vms, sso, providerLoading, providerError, search, filterProvider, filterState, filterRegion, filterTag } =
    useAppStore(useShallow(s => ({
      vms: s.vms, sso: s.sso, providerLoading: s.providerLoading, providerError: s.providerError,
      search: s.search, filterProvider: s.filterProvider, filterState: s.filterState,
      filterRegion: s.filterRegion, filterTag: s.filterTag,
    })));

  const connectedKeys = (Object.keys(PROVIDER_META) as ProviderID[]).filter(k => sso[k] === "connected");
  const ready = (k: ProviderID) => sso[k] === "connected" && !providerLoading[k] && !providerError[k];
  const anyLoading = connectedKeys.some(k => providerLoading[k]);

  const matches = (vm: { provider: ProviderID; region: string; state: string; name: string; iid: string; tags: string[] }) => {
    if (filterProvider !== "all" && vm.provider !== filterProvider) return false;
    if (filterRegion !== "all" && vm.region !== filterRegion) return false;
    if (filterTag && !vm.tags.some(t => t.includes(filterTag))) return false;
    if (filterState !== "all") {
      if (filterState === "transitioning" ? vm.state !== "transitioning" : vm.state !== filterState) return false;
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      if (!`${vm.name} ${vm.iid} ${vm.region} ${vm.tags.join(" ")}`.toLowerCase().includes(q)) return false;
    }
    return true;
  };

  const rows = vms.filter(vm => ready(vm.provider) && matches(vm));
  return `${rows.length} of ${vms.length} VMs${anyLoading ? " · loading…" : ""}`;
}

export function InventoryScreen() {
  const resultLabel = useResultLabel();
  return (
    <>
      <div className="flex-1 flex flex-col overflow-hidden">
        <InventoryFilters resultLabel={resultLabel} />
        <InventoryTable />
      </div>
      <VMDetailDrawer />
    </>
  );
}
