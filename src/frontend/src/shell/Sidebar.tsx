import { useState, useRef, useEffect } from "react";
import { useShallow } from "zustand/react/shallow";
import { useAppStore } from "@/store/useAppStore";
import { PROVIDER_META } from "@/lib/bridge";
import { StratusMark } from "@/ui/StratusLogo";
import type { Screen, ProviderID } from "@/types/domain";

const NAV: { screen: Screen; label: string; icon: React.ReactNode }[] = [
  {
    screen: "inventory", label: "Inventory",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
        <rect x="2" y="3" width="20" height="7" rx="1.5" /><rect x="2" y="14" width="20" height="7" rx="1.5" />
        <line x1="6" y1="6.5" x2="6.01" y2="6.5" /><line x1="6" y1="17.5" x2="6.01" y2="17.5" />
      </svg>
    ),
  },
  {
    screen: "sessions", label: "Sessions",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
        <polyline points="4 17 10 11 4 5" /><line x1="12" y1="19" x2="20" y2="19" />
      </svg>
    ),
  },
  {
    screen: "audit", label: "Audit log",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
        <path d="M3 3v18h18" /><path d="M7 14l3-3 3 3 4-5" />
      </svg>
    ),
  },
  {
    screen: "settings", label: "Settings",
    icon: (
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
        <line x1="4" y1="6" x2="20" y2="6" /><line x1="4" y1="12" x2="20" y2="12" /><line x1="4" y1="18" x2="20" y2="18" />
        <circle cx="9" cy="6" r="2.2" fill="var(--surface)" /><circle cx="15" cy="12" r="2.2" fill="var(--surface)" /><circle cx="8" cy="18" r="2.2" fill="var(--surface)" />
      </svg>
    ),
  },
];

function UserMenu({ signOut }: { signOut: () => void }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [open]);

  return (
    <div ref={ref} className="relative mt-2 pt-2 px-2.5" style={{ borderTop: "1px solid var(--border)" }}>
      {/* Popup menu — appears above the profile row */}
      {open && (
        <div
          className="absolute left-2.5 right-2.5 bottom-[calc(100%+4px)] rounded-[8px] py-1 shadow-[0_4px_16px_rgba(15,23,42,.25)] animate-fade-up z-10"
          style={{ background: "var(--surface)", border: "1px solid var(--border)" }}
        >
          <button
            onClick={() => { setOpen(false); signOut(); }}
            className="w-full flex items-center gap-2.5 px-3 py-2 text-[13px] font-medium border-none bg-transparent cursor-pointer transition-colors rounded-[6px] text-left"
            style={{ color: "#DC2626" }}
            onMouseEnter={e => (e.currentTarget.style.background = "rgba(220,38,38,.08)")}
            onMouseLeave={e => (e.currentTarget.style.background = "transparent")}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round">
              <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
              <polyline points="16 17 21 12 16 7" />
              <line x1="21" y1="12" x2="9" y2="12" />
            </svg>
            Sign out
          </button>
        </div>
      )}

      {/* Clickable profile row */}
      <button
        onClick={() => setOpen(o => !o)}
        className="w-full flex items-center gap-2.5 rounded-[7px] px-1 py-1.5 border-none cursor-pointer transition-colors text-left"
        style={{ background: open ? "var(--raised)" : "transparent" }}
        onMouseEnter={e => { if (!open) e.currentTarget.style.background = "var(--raised)"; }}
        onMouseLeave={e => { if (!open) e.currentTarget.style.background = "transparent"; }}
      >
        <div className="w-7 h-7 rounded-full bg-primary text-white flex items-center justify-center text-[12px] font-semibold flex-none">
          DK
        </div>
        <div className="leading-tight min-w-0 flex-1">
          <div className="text-[13px] font-semibold text-foreground whitespace-nowrap overflow-hidden text-ellipsis">
            Dana Khoury
          </div>
          <div className="text-[11px] text-muted-foreground">DevOps</div>
        </div>
        <svg
          width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor"
          strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"
          className="text-muted-foreground flex-none transition-transform"
          style={{ transform: open ? "rotate(180deg)" : "rotate(0deg)" }}
        >
          <polyline points="18 15 12 9 6 15" />
        </svg>
      </button>
    </div>
  );
}

export function Sidebar() {
  const { screen, go, vms, sessions, sso, providerError, signOut } = useAppStore(useShallow(s => ({
    screen: s.screen, go: s.go, vms: s.vms, sessions: s.sessions,
    sso: s.sso, providerError: s.providerError, signOut: s.signOut,
  })));

  const totalVms = vms.length;
  const sessionCount = sessions.length;

  return (
		<nav
			className="w-[220px] flex-none flex flex-col py-3 px-2.5"
			style={{
				background: "var(--surface)",
				borderRight: "1px solid var(--border)",
			}}
		>
			{/* Brand */}
			<div className="flex w-full items-center justify-center gap-2.5 pb-[14px]">
				<StratusMark size={28} />
				<div className="leading-tight">
					<div className="text-[15px] font-bold tracking-tight text-foreground">
						Stratus
					</div>
					{/* <div className="text-[10.5px] font-medium text-muted-foreground">VM gateway</div> */}
				</div>
			</div>

			{/* Nav label */}
			<div className="text-[11px] font-semibold text-muted-foreground uppercase tracking-[.06em] px-2.5 pb-2">
				Navigation
			</div>

			{/* Nav items */}
			{NAV.map(({ screen: s, label, icon }) => {
				const active = screen === s;
				return (
					<button
						key={s}
						onClick={() => go(s)}
						className="flex items-center gap-3 w-full px-3 py-2 rounded-[7px] mb-0.5 text-[14px] font-medium border-none cursor-pointer transition-colors duration-100 text-left"
						style={{
							background: active ? "var(--primary)" : "transparent",
							color: active ? "#fff" : "var(--foreground)",
						}}
					>
						{icon}
						<span className="flex-1">{label}</span>
						{s === "inventory" && (
							<span
								className="text-[11px] font-semibold px-2 py-0.5 rounded-[20px]"
								style={
									active
										? { background: "rgba(255,255,255,.22)", color: "#fff" }
										: {
												background: "var(--background)",
												color: "var(--muted-foreground)",
												border: "1px solid var(--border)",
											}
								}
							>
								{totalVms}
							</span>
						)}
						{s === "sessions" && sessionCount > 0 && (
							<span
								className="text-[11px] font-semibold px-2 py-0.5 rounded-[20px]"
								style={
									active
										? { background: "rgba(255,255,255,.22)", color: "#fff" }
										: {
												background: "var(--background)",
												color: "var(--muted-foreground)",
												border: "1px solid var(--border)",
											}
								}
							>
								{sessionCount}
							</span>
						)}
					</button>
				);
			})}

			<div className="flex-1" />

			{/* Providers + sign-out + user */}
			<div
				style={{
					borderTop: "1px solid var(--border)",
					paddingTop: 10,
					marginTop: 8,
				}}
			>
				<div className="text-[11px] font-semibold text-muted-foreground px-2.5 pb-[7px]">
					Providers
				</div>
				{(Object.keys(PROVIDER_META) as ProviderID[]).map((p) => {
					const meta = PROVIDER_META[p];
					const conn = sso[p] === "connected";
					const err = providerError[p];
					const statusColor = err ? "#DC2626" : conn ? "#16A34A" : "#64748B";
					return (
						<div
							key={p}
							className="flex items-center gap-2.5 px-2.5 py-1.5 text-[12px]"
						>
							<span
								className="inline-block rounded-full flex-none"
								style={{ width: 8, height: 8, background: meta.color }}
							/>
							<span className="flex-1 font-medium text-foreground">
								{meta.label}
							</span>
							<span
								className="text-[11px] font-semibold"
								style={{ color: statusColor }}
							>
								{err ? "Expired" : conn ? "OK" : "Off"}
							</span>
						</div>
					);
				})}

				{/* User profile — click to open popup */}
				<UserMenu signOut={signOut} />
			</div>
		</nav>
	);
}
