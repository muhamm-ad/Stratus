import { AlertTriangle } from "lucide-react";
import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";
import { PROVIDER_META, STATE_META } from "@/lib/bridge";
import { StatusBadge } from "@/ui/StatusBadge";
import { ProviderChip } from "@/ui/ProviderChip";
import { EmptyState } from "@/ui/EmptyState";
import { Skeleton } from "@/ui/skeleton";
import type { VMInstance, ProviderID } from "@/types/domain";
import type { FilterProvider, FilterState } from "@/store/useAppStore";
import { Search } from "lucide-react";

function useFilteredVms() {
  const { vms, sso, providerLoading, providerError, search, filterProvider, filterState, filterRegion } = useAppStore(useShallow(s => ({
    vms: s.vms,
    sso: s.sso,
    providerLoading: s.providerLoading,
    providerError: s.providerError,
    search: s.search,
    filterProvider: s.filterProvider,
    filterState: s.filterState,
    filterRegion: s.filterRegion,
  })));

  const ready = (k: ProviderID) => sso[k] === "connected" && !providerLoading[k] && !providerError[k];

  const matches = (vm: VMInstance): boolean => {
    if (filterProvider !== "all" && vm.provider !== filterProvider) return false;
    if (filterRegion !== "all" && vm.region !== filterRegion) return false;
    if (filterState !== "all") {
      if (filterState === "transitioning") {
				if (vm.state !== "transitioning") return false;
			} else if (vm.state !== filterState) return false;
    }
    if (search.trim()) {
      const q = search.trim().toLowerCase();
      const hay = `${vm.name} ${vm.id} ${vm.region} ${vm.tags.join(" ")}`.toLowerCase();
      if (!hay.includes(q)) return false;
    }
    return true;
  };

  const connectedKeys = (Object.keys(PROVIDER_META) as ProviderID[]).filter(k => sso[k] === "connected");
  const rows = vms.filter(vm => ready(vm.provider) && matches(vm));
  const anyLoading = connectedKeys.some(k => providerLoading[k]);
  const errorKeys = connectedKeys.filter(k => providerError[k]);
  const showEmpty = !anyLoading && rows.length === 0 && connectedKeys.some(ready);

  return { rows, anyLoading, connectedKeys, errorKeys, showEmpty, ready, matches };
}

/* ---- Error banner ---- */
function ErrorBanner({ providerKey }: { providerKey: ProviderID }) {
  const reconnect = useAppStore(s => s.reconnect);
  const meta = PROVIDER_META[providerKey];
  return (
    <div className="flex items-center gap-3 rounded-[8px] px-3.5 py-3 mb-3.5"
      style={{ background: "rgba(220,38,38,.10)", border: "1px solid rgba(220,38,38,.35)" }}
    >
      <AlertTriangle size={17} style={{ color: "#DC2626", flexShrink: 0 }} />
      <div className="flex-1 text-[13px] text-foreground">
        <strong className="font-semibold">{meta.label}:</strong> session expired, reconnect to load these VMs
      </div>
      <button
        onClick={() => reconnect(providerKey)}
        className="bg-danger text-white border-none rounded-[6px] px-3 py-1.5 text-[12px] font-semibold cursor-pointer"
      >
        Reconnect
      </button>
    </div>
  );
}

/* ---- Skeleton rows ---- */
function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-4 h-[46px]" style={{ borderBottom: "1px solid var(--border)" }}>
          <Skeleton className="w-5 h-5 rounded-full" />
          <Skeleton className="w-[180px] h-3" />
          <span className="flex-1" />
          <Skeleton className="w-[70px] h-6" />
        </div>
      ))}
    </>
  );
}

/* ---- Table header ---- */
const COL_GRID = "minmax(0,3fr) minmax(0,1fr) minmax(0,1.2fr) minmax(0,1.2fr) minmax(0,1fr) 104px";

function TableHeader() {
  return (
    <div
      className="grid px-4 py-2.5 text-[11px] font-semibold text-muted-foreground uppercase tracking-[.05em]"
      style={{ gridTemplateColumns: COL_GRID, background: "var(--raised)", borderBottom: "1px solid var(--border)" }}
    >
      <div>Name</div>
      <div>Provider</div>
      <div>Region</div>
      <div>Type</div>
      <div>State</div>
      <div className="text-right">Action</div>
    </div>
  );
}

/* ---- Table row ---- */
function TableRow({ vm }: { vm: VMInstance }) {
  const { openVm, connect } = useAppStore(useShallow(s => ({ openVm: s.openVm, connect: s.connect })));
  const stateMeta = STATE_META[vm.state] ?? STATE_META.unknown;

  return (
    <div
      onClick={() => openVm(vm.id)}
      className="grid items-center px-4 h-[46px] text-[13px] cursor-pointer transition-colors"
      style={{
        gridTemplateColumns: COL_GRID,
        borderBottom: "1px solid var(--border)",
      }}
      onMouseEnter={e => (e.currentTarget.style.background = "var(--raised)")}
      onMouseLeave={e => (e.currentTarget.style.background = "")}
    >
      {/* Name */}
      <div className="flex items-center gap-2.5 min-w-0">
        <span
          className="inline-block rounded-full flex-none"
          style={{
            width: 9, height: 9, background: stateMeta.color,
            boxShadow: vm.state === "running" ? `0 0 0 3px ${stateMeta.color}30` : undefined,
          }}
        />
        <div className="min-w-0">
          <div className="font-semibold whitespace-nowrap overflow-hidden text-ellipsis text-foreground">{vm.name}</div>
          <div className="text-[11px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.id}</div>
        </div>
      </div>

      {/* Provider */}
      <ProviderChip provider={vm.provider} />

      {/* Region */}
      <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.region}</div>

      {/* Type */}
      <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.size}</div>

      {/* State */}
      <div><StatusBadge state={vm.state} /></div>

      {/* Connect */}
      <div className="text-right">
        <button
          onClick={e => { e.stopPropagation(); connect(vm); }}
          disabled={!vm.canConnect}
          className="border-none rounded-[6px] px-3 py-1.5 text-[12px] font-semibold cursor-pointer disabled:opacity-75 disabled:cursor-not-allowed transition-opacity"
          style={vm.canConnect
            ? { background: "var(--primary)", color: "#fff" }
            : { background: "transparent", color: "var(--muted-foreground)", border: "1px solid var(--border)" }
          }
        >
          {vm.canConnect ? "Connect" : "No access"}
        </button>
      </div>
    </div>
  );
}

/* ---- Card ---- */
function VMCard({ vm }: { vm: VMInstance }) {
  const { openVm, connect } = useAppStore(useShallow(s => ({ openVm: s.openVm, connect: s.connect })));
  const stateMeta = STATE_META[vm.state] ?? STATE_META.unknown;
  const provMeta = PROVIDER_META[vm.provider];

  return (
    <div
      onClick={() => openVm(vm.id)}
      className="border border-border rounded-[8px] bg-surface p-4 cursor-pointer flex flex-col gap-3 transition-colors"
      onMouseEnter={e => (e.currentTarget.style.borderColor = "var(--primary)")}
      onMouseLeave={e => (e.currentTarget.style.borderColor = "")}
    >
      <div className="flex items-start gap-2.5">
        <span
          className="inline-block rounded-full mt-[5px]"
          style={{ width: 9, height: 9, background: stateMeta.color, flexShrink: 0,
            boxShadow: vm.state === "running" ? `0 0 0 3px ${stateMeta.color}30` : undefined }}
        />
        <div className="flex-1 min-w-0">
          <div className="font-semibold text-[14px] whitespace-nowrap overflow-hidden text-ellipsis text-foreground">{vm.name}</div>
          <div className="text-[11px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.id}</div>
        </div>
        <StatusBadge state={vm.state} />
      </div>

      <div className="flex gap-2 flex-wrap text-[11px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
        <span className="inline-flex items-center gap-1.5">
          <span className="inline-block rounded-full" style={{ width: 7, height: 7, background: provMeta.color }} />
          {provMeta.label}
        </span>
        <span>·</span><span>{vm.region}</span><span>·</span><span>{vm.size}</span>
      </div>

      <button
        onClick={e => { e.stopPropagation(); connect(vm); }}
        disabled={!vm.canConnect}
        className="w-full rounded-[6px] py-2 text-[13px] font-semibold border-none cursor-pointer disabled:opacity-75 disabled:cursor-not-allowed transition-opacity"
        style={vm.canConnect
          ? { background: "var(--primary)", color: "#fff" }
          : { background: "transparent", color: "var(--muted-foreground)", border: "1px solid var(--border)" }
        }
      >
        {vm.canConnect ? "Connect" : "No access"}
      </button>
    </div>
  );
}

/* ---- Grouped view ---- */
function GroupedView({ matches }: { matches: (vm: VMInstance) => boolean }) {
  const { vms, sso, providerLoading, providerError, reconnect } = useAppStore(useShallow(s => ({
    vms: s.vms, sso: s.sso, providerLoading: s.providerLoading, providerError: s.providerError, reconnect: s.reconnect,
  })));

  const connectedKeys = (Object.keys(PROVIDER_META) as ProviderID[]).filter(k => sso[k] === "connected");

  return (
    <div className="flex flex-col gap-[18px]">
      {connectedKeys.map(k => {
        const meta = PROVIDER_META[k];
        const loading = providerLoading[k];
        const error = providerError[k];
        const groupVms = vms.filter(vm => vm.provider === k && matches(vm));

        return (
          <div key={k}>
            <div className="flex items-center gap-2.5 mb-2.5">
              <span className="inline-block rounded-full" style={{ width: 11, height: 11, background: meta.color }} />
              <span className="text-[15px] font-semibold text-foreground">{meta.label}</span>
              {!loading && !error && (
                <span
                  className="text-[12px] font-semibold px-2 py-0.5 rounded-[20px]"
                  style={{ color: "var(--muted-foreground)", background: "var(--surface)", border: "1px solid var(--border)" }}
                >
                  {groupVms.length} VM{groupVms.length !== 1 ? "s" : ""}
                </span>
              )}
              {loading && (
                <span className="inline-flex items-center gap-1.5 text-[12px] text-muted-foreground">
                  <span className="inline-block w-3 h-3 border-2 border-muted-foreground border-t-transparent rounded-full animate-spin" />
                  loading…
                </span>
              )}
              <span className="flex-1" />
              <span className="text-[11px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
                via {meta.method}
              </span>
            </div>

            {error && (
              <div className="flex items-center gap-3 rounded-[8px] px-3.5 py-3"
                style={{ background: "rgba(220,38,38,.10)", border: "1px solid rgba(220,38,38,.35)" }}
              >
                <AlertTriangle size={16} style={{ color: "#DC2626", flexShrink: 0 }} />
                <span className="flex-1 text-[13px]">Session expired — reconnect to load these VMs.</span>
                <button
                  onClick={() => reconnect(k)}
                  className="bg-danger text-white border-none rounded-[6px] px-3 py-1.5 text-[12px] font-semibold cursor-pointer"
                >
                  Reconnect
                </button>
              </div>
            )}

            {!loading && !error && (
              <div className="border border-border rounded-[8px] overflow-hidden bg-surface">
                {groupVms.map(vm => (
                  <GroupedRow key={vm.id} vm={vm} />
                ))}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
}

const GRP_GRID = "minmax(0,3fr) minmax(0,1.2fr) minmax(0,1.2fr) minmax(0,1fr) 104px";

function GroupedRow({ vm }: { vm: VMInstance }) {
  const { openVm, connect } = useAppStore(useShallow(s => ({ openVm: s.openVm, connect: s.connect })));
  const stateMeta = STATE_META[vm.state] ?? STATE_META.unknown;

  return (
    <div
      onClick={() => openVm(vm.id)}
      className="grid items-center px-4 h-[44px] text-[13px] cursor-pointer transition-colors"
      style={{ gridTemplateColumns: GRP_GRID, borderBottom: "1px solid var(--border)" }}
      onMouseEnter={e => (e.currentTarget.style.background = "var(--raised)")}
      onMouseLeave={e => (e.currentTarget.style.background = "")}
    >
      <div className="flex items-center gap-2.5 min-w-0">
        <span className="inline-block rounded-full" style={{ width: 9, height: 9, background: stateMeta.color, flexShrink: 0 }} />
        <span className="font-semibold whitespace-nowrap overflow-hidden text-ellipsis text-foreground">{vm.name}</span>
        <span className="text-[11px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.id}</span>
      </div>
      <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.region}</div>
      <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>{vm.size}</div>
      <div><StatusBadge state={vm.state} /></div>
      <div className="text-right">
        <button
          onClick={e => { e.stopPropagation(); connect(vm); }}
          disabled={!vm.canConnect}
          className="border-none rounded-[6px] px-3 py-1.5 text-[12px] font-semibold cursor-pointer disabled:opacity-75 disabled:cursor-not-allowed"
          style={vm.canConnect
            ? { background: "var(--primary)", color: "#fff" }
            : { background: "transparent", color: "var(--muted-foreground)", border: "1px solid var(--border)" }
          }
        >
          {vm.canConnect ? "Connect" : "No access"}
        </button>
      </div>
    </div>
  );
}

/* ---- Main export ---- */
export function InventoryTable() {
  const view = useAppStore(s => s.view);
  const clearFilters = useAppStore(s => s.clearFilters);
  const { rows, anyLoading, errorKeys, showEmpty, matches } = useFilteredVms();

  return (
    <div className="flex-1 overflow-auto px-6 py-4 pb-7">
      {/* Error banners (table/cards view only) */}
      {view !== "grouped" && errorKeys.map(k => <ErrorBanner key={k} providerKey={k} />)}

      {/* Table view */}
      {view === "table" && (
        <div className="border border-border rounded-[8px] overflow-hidden bg-surface">
          <TableHeader />
          {rows.map(vm => <TableRow key={vm.id} vm={vm} />)}
          {anyLoading && <SkeletonRows />}
          {showEmpty && (
            <EmptyState
              icon={<Search size={26} strokeWidth={1.8} />}
              title="No VM matches these filters"
              description="Try a different provider, region, or state — or clear your search."
              action={{ label: "Clear all filters", onClick: clearFilters }}
            />
          )}
        </div>
      )}

      {/* Cards view */}
      {view === "cards" && (
        <>
          <div className="grid gap-3.5" style={{ gridTemplateColumns: "repeat(auto-fill,minmax(280px,1fr))" }}>
            {rows.map(vm => <VMCard key={vm.id} vm={vm} />)}
          </div>
          {showEmpty && (
            <EmptyState
              icon={<Search size={26} strokeWidth={1.8} />}
              title="No VM matches these filters"
              description="Try a different provider, region, or state — or clear your search."
              action={{ label: "Clear all filters", onClick: clearFilters }}
            />
          )}
        </>
      )}

      {/* Grouped view */}
      {view === "grouped" && <GroupedView matches={matches} />}
    </div>
  );
}
