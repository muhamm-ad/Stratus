import { CheckCircle, XCircle } from "lucide-react";
import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";
import { PROVIDER_META, MOCK_CLI } from "@/lib/bridge";
import type { ProviderID } from "@/types/domain";
import type { Density } from "@/store/useAppStore";

const THEME_OPTIONS: { key: "light" | "dark" | "system"; label: string }[] = [
  { key: "light",  label: "Light"  },
  { key: "dark",   label: "Dark"   },
  { key: "system", label: "System" },
];

const DENSITY_OPTIONS: { key: Density; label: string }[] = [
  { key: "comfortable", label: "Comfortable" },
  { key: "compact",     label: "Compact"     },
];

export function SettingsScreen() {
  const { sso, providerError, prefTheme, autoRefresh, density, setTheme, toggleAutoRefresh, toggleProvider, setDensity } =
    useAppStore(useShallow(s => ({
      sso: s.sso, providerError: s.providerError, prefTheme: s.prefTheme,
      autoRefresh: s.autoRefresh, density: s.density,
      setTheme: s.setTheme, toggleAutoRefresh: s.toggleAutoRefresh,
      toggleProvider: s.toggleProvider, setDensity: s.setDensity,
    })));

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex-none px-6 py-[18px] pb-4" style={{ borderBottom: "1px solid var(--border)" }}>
        <h1 className="m-0 text-[20px] font-semibold tracking-tight text-foreground">Settings</h1>
      </div>

      <div className="flex-1 overflow-auto px-6 py-[22px]">
        <div className="max-w-[680px] flex flex-col gap-[26px]">

          {/* Providers */}
          <section>
            <div className="text-[16px] font-semibold text-foreground mb-[3px]">Providers</div>
            <div className="text-[13px] text-muted-foreground mb-[13px]">Manage your connected cloud accounts.</div>
            <div className="border border-border rounded-[8px] overflow-hidden bg-surface">
              {(Object.keys(PROVIDER_META) as ProviderID[]).map(p => {
                const meta = PROVIDER_META[p];
                const conn = sso[p] === "connected";
                const connecting = sso[p] === "connecting";
                const err = providerError[p];
                const statusColor = err ? "#DC2626" : connecting ? "#D97706" : conn ? "#16A34A" : "#64748B";
                const identity = conn ? meta.identity : connecting ? "authorizing…" : "not connected";
                return (
                  <div key={p} className="flex items-center gap-[13px] px-4 py-[14px]"
                    style={{ borderBottom: "1px solid var(--border)" }}>
                    <span className="inline-block rounded-full flex-none" style={{ width: 11, height: 11, background: meta.color }} />
                    <div className="flex-1 min-w-0">
                      <div className="text-[14px] font-semibold text-foreground">{meta.label}</div>
                      <div className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
                        {identity}
                      </div>
                    </div>
                    <span className="text-[12px] font-semibold" style={{ color: statusColor }}>
                      {err ? "Session expired" : connecting ? "Connecting…" : conn ? "Connected" : "Disconnected"}
                    </span>
                    <button
                      onClick={() => !connecting && toggleProvider(p)}
                      className="rounded-[6px] px-[13px] py-[7px] text-[13px] font-semibold cursor-pointer border transition-opacity"
                      style={conn || connecting
                        ? { background: "transparent", color: "var(--foreground)", borderColor: "var(--border)", opacity: connecting ? 0.6 : 1 }
                        : { background: "var(--primary)", color: "#fff", borderColor: "transparent" }
                      }>
                      {connecting ? "Connecting…" : conn ? "Disconnect" : "Connect"}
                    </button>
                  </div>
                );
              })}
            </div>
          </section>

          {/* Preferences */}
          <section>
            <div className="text-[16px] font-semibold text-foreground mb-[13px]">Preferences</div>
            <div className="border border-border rounded-[8px] bg-surface" style={{ padding: "4px 16px" }}>

              {/* Theme */}
              <div className="flex items-center gap-[13px] py-[14px]" style={{ borderBottom: "1px solid var(--border)" }}>
                <div className="flex-1">
                  <div className="text-[14px] font-medium text-foreground">Theme</div>
                  <div className="text-[12px] text-muted-foreground">Application color scheme</div>
                </div>
                <div className="flex gap-1 p-[3px] rounded-[7px] border border-border" style={{ background: "var(--background)" }}>
                  {THEME_OPTIONS.map(t => (
                    <button key={t.key} onClick={() => setTheme(t.key)}
                      className="px-3 py-1.5 rounded-[5px] border-none cursor-pointer text-[12px] font-semibold transition-colors"
                      style={{
                        background: prefTheme === t.key ? "var(--primary)" : "transparent",
                        color: prefTheme === t.key ? "#fff" : "var(--muted-foreground)",
                      }}>
                      {t.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Row density */}
              <div className="flex items-center gap-[13px] py-[14px]" style={{ borderBottom: "1px solid var(--border)" }}>
                <div className="flex-1">
                  <div className="text-[14px] font-medium text-foreground">Row density</div>
                  <div className="text-[12px] text-muted-foreground">Inventory table spacing</div>
                </div>
                <div className="flex gap-1 p-[3px] rounded-[7px] border border-border" style={{ background: "var(--background)" }}>
                  {DENSITY_OPTIONS.map(d => (
                    <button key={d.key} onClick={() => setDensity(d.key)}
                      className="px-3 py-1.5 rounded-[5px] border-none cursor-pointer text-[12px] font-semibold transition-colors"
                      style={{
                        background: density === d.key ? "var(--primary)" : "transparent",
                        color: density === d.key ? "#fff" : "var(--muted-foreground)",
                      }}>
                      {d.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Auto-refresh */}
              <div className="flex items-center gap-[13px] py-[14px]" style={{ borderBottom: "1px solid var(--border)" }}>
                <div className="flex-1">
                  <div className="text-[14px] font-medium text-foreground">Auto-refresh inventory</div>
                  <div className="text-[12px] text-muted-foreground">Poll providers every 60 seconds</div>
                </div>
                <button onClick={toggleAutoRefresh}
                  className="w-[40px] h-[23px] rounded-[12px] border-none cursor-pointer p-[2px] flex transition-all duration-150"
                  style={{ background: autoRefresh ? "var(--primary)" : "var(--border)", justifyContent: autoRefresh ? "flex-end" : "flex-start" }}>
                  <span className="w-[19px] h-[19px] rounded-full bg-white block" style={{ boxShadow: "0 1px 2px rgba(0,0,0,.2)" }} />
                </button>
              </div>

              {/* CLI detection */}
              <div className="py-[14px]">
                <div className="text-[14px] font-medium text-foreground mb-[10px]">Local CLI detection</div>
                <div className="flex flex-col gap-2">
                  {MOCK_CLI.map(cli => (
                    <div key={cli.name} className="flex items-center gap-2.5 text-[13px]">
                      {cli.installed
                        ? <CheckCircle size={15} style={{ color: "#16A34A", flexShrink: 0 }} />
                        : <XCircle size={15} style={{ color: "#DC2626", flexShrink: 0 }} />
                      }
                      <span className="flex-1 text-[12px]" style={{ fontFamily: "var(--font-mono)" }}>{cli.name}</span>
                      <span className="text-[12px] font-semibold" style={{ color: cli.installed ? "#16A34A" : "#DC2626" }}>
                        {cli.installed ? "detected" : "missing"}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </section>

        </div>
      </div>
    </div>
  );
}
