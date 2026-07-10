import { X } from "lucide-react";
import { useState } from "react";
import { PROVIDER_META, MOCK_AUDIT } from "@/lib/bridge";
import { EmptyState } from "@/ui/EmptyState";
import { Search } from "lucide-react";

const COL = "minmax(0,1.4fr) minmax(0,1.2fr) minmax(0,1.6fr) minmax(0,.9fr) minmax(0,1fr) minmax(0,.9fr)";

export function AuditScreen() {
  const [query, setQuery] = useState("");

  const filtered = query.trim()
    ? MOCK_AUDIT.filter(a => {
        const q = query.toLowerCase();
        return (
          a.user.toLowerCase().includes(q) ||
          a.vmName.toLowerCase().includes(q) ||
          PROVIDER_META[a.provider].label.toLowerCase().includes(q) ||
          a.method.toLowerCase().includes(q)
        );
      })
    : MOCK_AUDIT;

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      {/* Header */}
      <div className="flex-none px-6 py-[18px] pb-4 flex items-center gap-3"
        style={{ borderBottom: "1px solid var(--border)" }}>
        <h1 className="m-0 text-[20px] font-semibold tracking-tight text-foreground">Audit log</h1>
        <span className="text-[13px] text-muted-foreground flex-1">
          {filtered.length} of {MOCK_AUDIT.length} entries
        </span>

        {/* Search */}
        <div className="relative flex items-center">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="var(--muted-foreground)" strokeWidth="2"
            strokeLinecap="round" strokeLinejoin="round" className="absolute left-[11px] pointer-events-none">
            <circle cx="11" cy="11" r="7" /><line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            value={query}
            onChange={e => setQuery(e.target.value)}
            placeholder="Search user, VM, method…"
            className="h-[34px] rounded-[7px] border border-border text-foreground text-[13px] outline-none"
            style={{ width: 240, background: "var(--background)", padding: "0 32px 0 33px" }}
          />
          {query && (
            <button onClick={() => setQuery("")}
              className="absolute right-2 w-[18px] h-[18px] border-none bg-transparent text-muted-foreground cursor-pointer flex items-center justify-center p-0">
              <X size={14} />
            </button>
          )}
        </div>

        {/* Period badge */}
        <span className="inline-flex items-center gap-1.5 text-[12px] text-muted-foreground border border-border rounded-[6px] px-[11px] py-1.5 flex-none">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <rect x="3" y="4" width="18" height="18" rx="2" /><line x1="16" y1="2" x2="16" y2="6" />
            <line x1="8" y1="2" x2="8" y2="6" /><line x1="3" y1="10" x2="21" y2="10" />
          </svg>
          Last 7 days
        </span>
      </div>

      <div className="flex-1 overflow-auto px-6 py-[18px]">
        {filtered.length > 0 ? (
          <div className="border border-border rounded-[8px] overflow-hidden bg-surface">
            {/* Header */}
            <div className="grid px-4 py-[9px] text-[11px] font-semibold text-muted-foreground uppercase tracking-wider"
              style={{ gridTemplateColumns: COL, background: "var(--raised)", borderBottom: "1px solid var(--border)" }}>
              <div>Timestamp</div>
              <div>User</div>
              <div>VM</div>
              <div>Provider</div>
              <div>Method</div>
              <div>Result</div>
            </div>
            {filtered.map((entry, i) => {
              const provMeta = PROVIDER_META[entry.provider];
              const ok = entry.result === "success";
              const resultColor = ok ? "#16A34A" : "#DC2626";
              return (
                <div key={i} className="grid items-center px-4 h-[42px] text-[13px] text-foreground"
                  style={{ gridTemplateColumns: COL, borderBottom: "1px solid var(--border)" }}>
                  <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
                    {entry.timestamp}
                  </div>
                  <div>{entry.user}</div>
                  <div className="font-medium whitespace-nowrap overflow-hidden text-ellipsis">{entry.vmName}</div>
                  <div className="flex items-center gap-2">
                    <span className="inline-block rounded-full" style={{ width: 8, height: 8, background: provMeta.color }} />
                    {provMeta.label}
                  </div>
                  <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
                    {entry.method}
                  </div>
                  <div>
                    <span className="inline-flex items-center gap-1.5 text-[12px] font-semibold rounded-[5px] px-2 py-0.5"
                      style={{ color: resultColor, background: `${resultColor}1a` }}>
                      {entry.result}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <EmptyState
            icon={<Search size={26} strokeWidth={1.8} />}
            title="No entries match your search"
            description="Try a different user, VM, provider, or method."
          />
        )}
      </div>
    </div>
  );
}
