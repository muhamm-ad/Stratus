import { type ReactNode, useEffect } from "react";

interface DialogProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}

export function Dialog({ open, onClose, children }: DialogProps) {
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 bg-black/55 z-[60] flex items-center justify-center px-6 animate-pop"
      onClick={onClose}
    >
      <div
        className="w-[380px] max-w-full bg-background border border-border rounded-[12px] shadow-[0_12px_40px_rgba(2,6,23,.4)] p-6 animate-pop"
        onClick={e => e.stopPropagation()}
      >
        {children}
      </div>
    </div>
  );
}
