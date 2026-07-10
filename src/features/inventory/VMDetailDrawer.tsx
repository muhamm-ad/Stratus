import { X, Terminal, CheckCircle, Lock, Copy } from "lucide-react";
import { useAppStore } from "@/store/useAppStore";
import { useShallow } from "zustand/react/shallow";
import { PROVIDER_META, STATE_META, MOCK_AUDIT } from "@/lib/bridge";
import { Sheet } from "@/ui/sheet";
import { StatusBadge } from "@/ui/StatusBadge";
import { ProviderChip } from "@/ui/ProviderChip";

function CopyBtn({ value, title }: { value: string; title: string }) {
  const copyToClipboard = () => {
    navigator.clipboard.writeText(value).catch(() => {
      /* silent fail in environments without clipboard API */
    });
  };
  return (
    <button
      onClick={copyToClipboard}
      title={title}
      className="w-5 h-5 border-none bg-transparent text-muted-foreground cursor-pointer flex items-center justify-center p-0 hover:text-foreground transition-colors"
    >
      <Copy size={13} />
    </button>
  );
}

export function VMDetailDrawer() {
  const { selectedVmId, vms, closeDrawer, connect, setFilterTag } = useAppStore(useShallow(s => ({
    selectedVmId: s.selectedVmId,
    vms: s.vms,
    closeDrawer: s.closeDrawer,
    connect: s.connect,
    setFilterTag: s.setFilterTag,
  })));

  const vm = vms.find(v => v.id === selectedVmId) ?? null;
  if (!vm) return null;

  const stateMeta = STATE_META[vm.state] ?? STATE_META.unknown;
  const provMeta = PROVIDER_META[vm.provider];
  const canConnect = vm.canConnect;

  // Recent activity from audit log for this VM
  const recentActivity = MOCK_AUDIT.filter(a => a.vmName === vm.name).slice(0, 3);

  return (
    <Sheet open={!!vm} onClose={closeDrawer}>
      {/* Header */}
      <div className="px-5 py-[18px] pb-4" style={{ borderBottom: "1px solid var(--border)" }}>
        <div className="flex items-start gap-3">
          <span
            className="inline-block rounded-full mt-[7px] flex-none"
            style={{
              width: 9, height: 9, background: stateMeta.color,
              boxShadow: vm.state === "running" ? `0 0 0 3px ${stateMeta.color}30` : undefined,
            }}
          />
          <div className="flex-1 min-w-0">
            <div className="text-[17px] font-semibold whitespace-nowrap overflow-hidden text-ellipsis text-foreground">
              {vm.name}
            </div>
            <div className="flex items-center gap-2">
              <span className="text-[12px] text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
                {vm.id}
              </span>
              <CopyBtn value={vm.id} title="Copy instance ID" />
            </div>
          </div>
          <button
            onClick={closeDrawer}
            className="w-[30px] h-[30px] border border-border rounded-[6px] text-muted-foreground cursor-pointer flex items-center justify-center flex-none hover:text-foreground transition-colors"
            style={{ background: "var(--surface)" }}
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Details */}
      <div className="flex-1 overflow-auto px-5 py-[18px] flex flex-col gap-0.5">
        {/* Key-value rows */}
        {[
          {
            label: "Provider",
            value: <ProviderChip provider={vm.provider} />,
          },
          {
            label: "State",
            value: <StatusBadge state={vm.state} />,
          },
          {
            label: "Region",
            value: <span style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}>{vm.region}</span>,
          },
          {
            label: "Type",
            value: <span style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}>{vm.size}</span>,
          },
          {
            label: "Private IP",
            value: (
              <span className="inline-flex items-center gap-2">
                <span style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}>{vm.privateIP}</span>
                <CopyBtn value={vm.privateIP} title="Copy IP" />
              </span>
            ),
          },
          {
            label: "Connection method",
            value: (
              <span className="inline-flex items-center gap-1.5" style={{ fontFamily: "var(--font-mono)", fontSize: 13 }}>
                <Terminal size={14} />
                {provMeta.method}
              </span>
            ),
          },
        ].map(row => (
          <div
            key={row.label}
            className="flex items-center justify-between py-[11px]"
            style={{ borderBottom: "1px solid var(--border)" }}
          >
            <span className="text-[13px] text-muted-foreground">{row.label}</span>
            {row.value}
          </div>
        ))}

        {/* Tags — clickable to filter */}
        <div className="pt-3 pb-1">
          <div className="text-[13px] text-muted-foreground mb-[9px]">Tags · click to filter</div>
          <div className="flex flex-wrap gap-[7px]">
            {vm.tags.map(tag => (
              <button
                key={tag}
                onClick={() => { setFilterTag(tag); closeDrawer(); }}
                className="rounded-[5px] px-2 py-0.5 text-[11px] text-muted-foreground cursor-pointer transition-colors hover:border-primary hover:text-primary"
                style={{
                  fontFamily: "var(--font-mono)",
                  background: "var(--surface)",
                  border: "1px solid var(--border)",
                }}
              >
                {tag}
              </button>
            ))}
          </div>
        </div>

        {/* Permission notice */}
        <div
          className="flex items-start gap-2.5 mt-[14px] p-3 rounded-[8px] text-[12px] leading-relaxed text-foreground"
          style={{
            background: canConnect ? "rgba(22,163,74,.10)" : "rgba(217,119,6,.12)",
            border: `1px solid ${canConnect ? "rgba(22,163,74,.35)" : "rgba(217,119,6,.4)"}`,
          }}
        >
          {canConnect
            ? <CheckCircle size={16} style={{ color: "#16A34A", flexShrink: 0, marginTop: 1 }} />
            : <Lock size={16} style={{ color: "#D97706", flexShrink: 0, marginTop: 1 }} />
          }
          <span>
            {canConnect
              ? `You have permission to open a ${provMeta.method} session on this VM.`
              : `You can view this VM but lack permission to open a ${provMeta.method} session on it.`
            }
          </span>
        </div>

        {/* Recent activity */}
        {recentActivity.length > 0 && (
          <div className="pt-4">
            <div className="text-[13px] text-muted-foreground mb-[9px]">Recent activity</div>
            <div className="flex flex-col gap-[7px]">
              {recentActivity.map((a, i) => {
                const ok = a.result === "success";
                return (
                  <div key={i} className="flex items-center gap-2.5 text-[12px]">
                    <span
                      className="inline-flex items-center gap-1 rounded-[5px] px-2 py-0.5 font-semibold text-[11px]"
                      style={{ color: ok ? "#16A34A" : "#DC2626", background: (ok ? "#16A34A" : "#DC2626") + "1a" }}
                    >
                      {a.result}
                    </span>
                    <span className="text-muted-foreground" style={{ fontFamily: "var(--font-mono)" }}>
                      {a.timestamp}
                    </span>
                  </div>
                );
              })}
            </div>
          </div>
        )}
      </div>

      {/* Connect button */}
      <div className="px-5 py-4" style={{ borderTop: "1px solid var(--border)" }}>
        <button
          onClick={() => connect(vm)}
          disabled={!canConnect}
          className="w-full flex items-center justify-center gap-2 rounded-[6px] py-3 text-[14px] font-semibold border-none cursor-pointer disabled:cursor-not-allowed transition-opacity"
          style={canConnect
            ? { background: "var(--primary)", color: "#fff" }
            : { background: "var(--surface)", color: "var(--muted-foreground)", border: "1px solid var(--border)" }
          }
        >
          <Terminal size={17} />
          {canConnect ? `Connect via ${provMeta.method}` : "Insufficient permissions"}
        </button>
      </div>
    </Sheet>
  );
}
