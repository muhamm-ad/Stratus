import { type ReactNode, useEffect } from "react";
import { cn } from "@/lib/utils";

interface SheetProps {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
  className?: string;
}

export function Sheet({ open, onClose, children, className }: SheetProps) {
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    document.addEventListener("keydown", handler);
    return () => document.removeEventListener("keydown", handler);
  }, [open, onClose]);

  if (!open) return null;

  return (
    <>
      <div
        className="fixed inset-0 bg-black/50 z-40 animate-pop"
        onClick={onClose}
      />
      <aside
        className={cn(
          "fixed top-0 right-0 bottom-0 w-[400px] max-w-[92vw]",
          "bg-background border-l border-border shadow-[0_4px_24px_rgba(15,23,42,.25)]",
          "z-50 flex flex-col animate-drawer-in",
          className,
        )}
      >
        {children}
      </aside>
    </>
  );
}
