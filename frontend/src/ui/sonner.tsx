import { useAppStore } from "@/store/useAppStore";
import { CheckCircle, XCircle, AlertTriangle, X } from "lucide-react";

export function Toaster() {
  const toasts = useAppStore(s => s.toasts);
  const dismiss = useAppStore(s => s.dismissToast);

  if (toasts.length === 0) return null;

  return (
    <div className="fixed right-[18px] bottom-[42px] z-[70] flex flex-col gap-2.5 items-end">
      {toasts.map(t => {
        const iconColor = t.kind === "error" ? "#DC2626" : t.kind === "warning" ? "#D97706" : "#16A34A";
        const borderColor = t.kind === "error" ? "#DC2626" : t.kind === "warning" ? "#D97706" : "#16A34A";
        const Icon = t.kind === "error" ? XCircle : t.kind === "warning" ? AlertTriangle : CheckCircle;

        return (
          <div
            key={t.id}
            className="flex items-start gap-3 w-[320px] bg-surface border border-border rounded-[8px] p-3 shadow-[0_4px_12px_rgba(15,23,42,.18)] animate-fade-up"
            style={{ borderLeft: `3px solid ${borderColor}` }}
          >
            <Icon size={17} style={{ color: iconColor, flexShrink: 0, marginTop: 1 }} />
            <div className="flex-1 min-w-0">
              <div className="text-[13px] font-semibold text-foreground">{t.title}</div>
              {t.body && <div className="text-[12px] text-muted-foreground mt-0.5">{t.body}</div>}
            </div>
            {t.action && (
              <button
                onClick={t.action.fn}
                className="text-primary text-[12px] font-semibold whitespace-nowrap bg-transparent border-none cursor-pointer p-0"
              >
                {t.action.label}
              </button>
            )}
            <button
              onClick={() => dismiss(t.id)}
              className="text-muted-foreground bg-transparent border-none cursor-pointer p-0 flex"
            >
              <X size={15} />
            </button>
          </div>
        );
      })}
    </div>
  );
}
