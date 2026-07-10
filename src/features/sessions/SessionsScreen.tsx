import { Terminal, X } from "lucide-react";
import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";
import { PROVIDER_META } from "@/lib/bridge";
import { EmptyState } from "@/ui/EmptyState";
import { formatDuration, formatAgo } from "@/lib/format";

const COL = "minmax(0,2fr) minmax(0,1fr) minmax(0,1.2fr) minmax(0,1fr) 120px";

export function SessionsScreen() {
  const { sessions, vms, closeSession, sessionSearch, setSessionSearch } = useAppStore(useShallow(s => ({
    sessions: s.sessions,
    vms: s.vms,
    closeSession: s.closeSession,
    sessionSearch: s.sessionSearch,
    setSessionSearch: s.setSessionSearch,
    _tick: s.tickCount,
  })));

  const filtered = sessionSearch.trim()
    ? sessions.filter(se => {
        const vm = vms.find(v => v.id === se.vmId);
        if (!vm) return false;
        const q = sessionSearch.toLowerCase();
        return vm.name.toLowerCase().includes(q) || PROVIDER_META[vm.provider].label.toLowerCase().includes(q);
      })
    : sessions;

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex-none px-6 py-[18px] pb-4 flex items-center gap-[14px]"
        style={{ borderBottom: "1px solid var(--border)" }}>
        <h1 className="m-0 text-[20px] font-semibold tracking-tight text-foreground">Active sessions</h1>
        <span className="text-[13px] text-muted-foreground flex-1">
          {sessions.length} session{sessions.length !== 1 ? "s" : ""}
        </span>
        {/* Search */}
        <div className="relative flex items-center">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--muted-foreground)" strokeWidth="2"
            strokeLinecap="round" strokeLinejoin="round" className="absolute left-[11px] pointer-events-none">
            <circle cx="11" cy="11" r="7" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            value={sessionSearch}
            onChange={e => setSessionSearch(e.target.value)}
            placeholder="Search sessions…"
            className="h-[34px] rounded-[7px] border border-border text-foreground text-[13px] outline-none"
            style={{ width: 240, background: "var(--background)", padding: "0 32px 0 33px" }}
          />
          {sessionSearch && (
            <button onClick={() => setSessionSearch("")}
              className="absolute right-2 w-[18px] h-[18px] border-none bg-transparent text-muted-foreground cursor-pointer flex items-center justify-center p-0">
              <X size={14} />
            </button>
          )}
        </div>
      </div>

      <div className="flex-1 overflow-auto px-6 py-[18px]">
        {filtered.length > 0 ? (
          <div className="border border-border rounded-[8px] overflow-hidden bg-surface">
            {/* Table header */}
            <div className="grid px-4 py-[9px] text-[11px] font-semibold text-muted-foreground uppercase tracking-wider"
              style={{ gridTemplateColumns: COL, background: "var(--raised)", borderBottom: "1px solid var(--border)" }}>
              <div>Target</div>
              <div>Provider</div>
              <div>Opened</div>
              <div>Duration</div>
              <div className="text-right">Action</div>
            </div>
            {filtered.map(session => {
              const vm = vms.find(v => v.id === session.vmId);
              if (!vm) return null;
              const provMeta = PROVIDER_META[vm.provider];
              return (
                <div key={session.sid} className="grid items-center px-4 h-12 text-[13px]"
                  style={{ gridTemplateColumns: COL, borderBottom: "1px solid var(--border)" }}>
                  <div className="flex items-center gap-2.5">
                    <span className="inline-block rounded-full flex-none"
                      style={{ width: 8, height: 8, background: "#16A34A", boxShadow: "0 0 0 3px rgba(22,163,74,.18)" }} />
                    <span className="font-semibold text-foreground">{vm.name}</span>
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="inline-block rounded-full" style={{ width: 8, height: 8, background: provMeta.color }} />
                    <span>{provMeta.label}</span>
                  </div>
                  <div className="text-[12px] text-muted-foreground">{formatAgo(session.openedAt)}</div>
                  <div className="text-[12px]" style={{ fontFamily: "var(--font-mono)" }}>
                    {formatDuration(Date.now() - session.openedAt)}
                  </div>
                  <div className="text-right">
                    <button onClick={() => closeSession(session.sid)}
                      className="bg-transparent rounded-[6px] px-3 py-1.5 text-[12px] font-semibold cursor-pointer hover:opacity-80 transition-opacity"
                      style={{ color: "#DC2626", border: "1px solid rgba(220,38,38,.4)" }}>
                      Close
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <EmptyState
            icon={<Terminal size={26} strokeWidth={1.8} />}
            title={sessionSearch ? "No session matches" : "No active session"}
            description={sessionSearch ? "Try a different search." : "Connect to a VM from the inventory to start one."}
          />
        )}
      </div>
    </div>
  );
}
