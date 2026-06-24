import { LayoutList, LayoutGrid, List, ChevronDown, RefreshCw, X } from "lucide-react";
import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";
import type { FilterProvider, FilterState } from "@/store/useAppStore";
import { PROVIDER_META } from "@/lib/bridge";
import type { ViewMode } from "@/types/domain";

type ChipDef<T extends string> = { key: T; label: string; color?: string };

const PROVIDER_CHIPS: ChipDef<FilterProvider>[] = [
  { key: "all",   label: "All" },
  { key: "aws",   label: "AWS",   color: "#FF9900" },
  { key: "azure", label: "Azure", color: "#0078D4" },
  { key: "gcp",   label: "GCP",   color: "#4285F4" },
];

const STATE_CHIPS: ChipDef<FilterState>[] = [
  { key: "all",        label: "All states" },
  { key: "running",    label: "Running",       color: "#16A34A" },
  { key: "stopped",    label: "Stopped",       color: "#DC2626" },
  { key: "transitioning", label: "Transitioning", color: "#D97706" },
  { key: "unknown",    label: "Unknown",       color: "#64748B" },
];

const VIEW_MODES: { key: ViewMode; label: string; Icon: React.ComponentType<{ size?: number }> }[] = [
  { key: "table",   label: "Table",   Icon: LayoutList as React.ComponentType<{ size?: number }> },
  { key: "cards",   label: "Cards",   Icon: LayoutGrid as React.ComponentType<{ size?: number }> },
  { key: "grouped", label: "Grouped", Icon: List       as React.ComponentType<{ size?: number }> },
];

function ChipGroup<T extends string>({ chips, active, onSelect }: {
  chips: ChipDef<T>[]; active: T; onSelect: (k: T) => void;
}) {
  return (
    <div className="flex gap-1 rounded-[7px] p-[3px]"
      style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
      {chips.map(c => {
        const isActive = c.key === active;
        return (
          <button key={c.key} onClick={() => onSelect(c.key)}
            className="inline-flex items-center gap-1.5 h-7 px-3 rounded-[5px] border-none cursor-pointer text-[12px] transition-colors"
            style={{
              background: isActive ? "var(--primary)" : "transparent",
              color: isActive ? "#fff" : "var(--muted-foreground)",
              fontWeight: isActive ? 600 : 500,
            }}>
            {c.color && <span className="inline-block rounded-full" style={{ width: 7, height: 7, background: c.color, flexShrink: 0 }} />}
            {c.label}
          </button>
        );
      })}
    </div>
  );
}

export function InventoryFilters({ resultLabel }: { resultLabel: string }) {
  const {
    filterProvider, filterState, filterRegion, filterTag,
    view, vms, search,
    setFilterProvider, setFilterState, setFilterRegion,
    setSearch, setView, clearFilters, refresh, providerLoading,
  } = useAppStore(useShallow(s => ({
    filterProvider: s.filterProvider, filterState: s.filterState,
    filterRegion: s.filterRegion, filterTag: s.filterTag,
    view: s.view, vms: s.vms, search: s.search,
    setFilterProvider: s.setFilterProvider, setFilterState: s.setFilterState,
    setFilterRegion: s.setFilterRegion, setSearch: s.setSearch,
    setView: s.setView, clearFilters: s.clearFilters,
    refresh: s.refresh, providerLoading: s.providerLoading,
  })));

  const anyLoading = Object.values(providerLoading).some(Boolean);
  const regions = Array.from(new Set(vms.map(v => v.region))).sort();

  return (
    <div className="flex-none px-6 pt-[18px] pb-[14px]" style={{ borderBottom: "1px solid var(--border)" }}>
      {/* Row 1: title + result + refresh + view switcher */}
      <div className="flex items-center gap-3 mb-[14px]">
        <h1 className="m-0 text-[20px] font-semibold tracking-tight text-foreground">Inventory</h1>
        <span className="text-[13px] text-muted-foreground">{resultLabel}</span>
        <div className="flex-1" />

        <button
          onClick={refresh}
          title="Refresh inventory"
          className="w-[34px] h-[34px] border border-border rounded-[6px] text-muted-foreground cursor-pointer flex items-center justify-center hover:text-foreground flex-none"
          style={{ background: "var(--background)" }}
        >
          <RefreshCw size={16} className={anyLoading ? "animate-spin-slow" : ""} />
        </button>

        {/* View switcher */}
        <div className="flex gap-1 rounded-[7px] p-[3px]"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}>
          {VIEW_MODES.map(({ key, label, Icon }) => {
            const isActive = view === key;
            return (
              <button key={key} onClick={() => setView(key)} title={label}
                className="inline-flex items-center gap-1.5 h-7 px-3 rounded-[5px] border-none cursor-pointer text-[12px] font-semibold transition-all"
                style={{
                  background: isActive ? "var(--background)" : "transparent",
                  color: isActive ? "var(--foreground)" : "var(--muted-foreground)",
                  boxShadow: isActive ? "0 1px 2px rgba(15,23,42,.12)" : "none",
                }}>
                <Icon size={15} />{label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Row 2: search + chips + region */}
      <div className="flex items-center gap-2.5 flex-wrap">
        {/* Search */}
        <div className="relative flex items-center">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--muted-foreground)" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
            className="absolute left-[11px] pointer-events-none">
            <circle cx="11" cy="11" r="7" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search name, ID, region, tag…"
            className="h-[34px] rounded-[7px] border border-border text-foreground text-[13px] outline-none focus:ring-2 focus:ring-ring"
            style={{ width: 260, background: "var(--background)", padding: "0 32px 0 33px" }}
          />
          {search && (
            <button onClick={() => setSearch("")}
              className="absolute right-2 w-[18px] h-[18px] border-none bg-transparent text-muted-foreground cursor-pointer flex items-center justify-center p-0">
              <X size={14} />
            </button>
          )}
        </div>

        <ChipGroup chips={PROVIDER_CHIPS} active={filterProvider} onSelect={setFilterProvider} />
        <ChipGroup chips={STATE_CHIPS} active={filterState} onSelect={setFilterState} />

        {/* Region */}
        <div className="relative flex items-center">
          <select
            value={filterRegion}
            onChange={e => setFilterRegion(e.target.value)}
            className="h-[34px] rounded-[7px] border border-border text-foreground text-[13px] font-medium pl-3 pr-8 cursor-pointer outline-none"
            style={{ background: "var(--surface)" }}
          >
            <option value="all">All regions</option>
            {regions.map(r => <option key={r} value={r}>{r}</option>)}
          </select>
          <ChevronDown size={14} className="absolute right-2 text-muted-foreground pointer-events-none" />
        </div>

        {/* Active tag filter chip */}
        {filterTag && (
          <button
            onClick={clearFilters}
            className="inline-flex items-center gap-2 h-[34px] px-3 rounded-[7px] font-semibold text-[12px] cursor-pointer"
            style={{ border: "1px solid var(--primary)", background: "rgba(99,102,241,.12)", color: "var(--primary)", fontFamily: "var(--font-mono)" }}
          >
            {filterTag}
            <X size={13} />
          </button>
        )}
      </div>
    </div>
  );
}
