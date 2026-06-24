import { useShallow } from "zustand/react/shallow";
import { useAppStore } from "@/store/useAppStore";
import { CheckCircle, Loader2, Moon, Sun } from "lucide-react";

export function StatusBar() {
  const { vms, sessions, sso, providerLoading, theme, toggleTheme } = useAppStore(useShallow(s => ({
    vms: s.vms, sessions: s.sessions, sso: s.sso,
    providerLoading: s.providerLoading, theme: s.theme, toggleTheme: s.toggleTheme,
  })));

  const anyLoading = Object.values(providerLoading).some(Boolean);
  const connectedCount = Object.values(sso).filter(v => v === "connected").length;

  return (
    <footer
      className="h-[30px] flex-none flex items-center gap-4 px-4 text-[12px] text-muted-foreground"
      style={{ background: "var(--raised)", borderTop: "1px solid var(--border)" }}
    >
      {/* Inventory icon */}
      <span className="flex items-center gap-1.5">
        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
          <rect x="2" y="3" width="20" height="7" rx="1.5" /><rect x="2" y="14" width="20" height="7" rx="1.5" />
        </svg>
        {vms.length} VMs
      </span>
      <span className="w-px h-3.5 bg-border" />
      <span>{connectedCount} / 3 providers connected</span>
      <span className="w-px h-3.5 bg-border" />
      <span className="flex items-center gap-1.5">
        {anyLoading
          ? <Loader2 size={13} className="animate-spin" />
          : <CheckCircle size={13} style={{ color: "#16A34A" }} />
        }
        {anyLoading ? "Syncing…" : "Synced"}
      </span>

      <span className="flex-1" />

      {/* Keyboard shortcuts */}
      <span className="opacity-70" style={{ fontFamily: "var(--font-mono)" }}>
        / search · ↑↓ move · Enter open · C connect
      </span>
      <span className="w-px h-3.5 bg-border" />
      <span style={{ fontFamily: "var(--font-mono)" }}>{sessions.length} active session(s)</span>
      <span className="w-px h-3.5 bg-border" />

      {/* Theme toggle */}
      <button
        onClick={toggleTheme}
        className="flex items-center gap-1.5 bg-transparent border-none text-muted-foreground cursor-pointer text-[12px] font-medium p-0 hover:text-foreground transition-colors"
      >
        {theme === "dark"
          ? <><Moon size={14} /> Dark</>
          : <><Sun size={14} /> Light</>
        }
      </button>
    </footer>
  );
}
