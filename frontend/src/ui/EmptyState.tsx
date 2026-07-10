import type { ReactNode } from "react";
import { Button } from "@/ui/button";

interface EmptyStateProps {
  icon: ReactNode;
  title: string;
  description?: string;
  action?: { label: string; onClick: () => void };
}

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-20 px-5 text-center">
      <div className="w-14 h-14 rounded-[14px] bg-surface border border-border flex items-center justify-center mb-4 text-muted-foreground">
        {icon}
      </div>
      <div className="text-[16px] font-semibold mb-1.5">{title}</div>
      {description && (
        <div className="text-[13px] text-muted-foreground mb-5 max-w-[300px]">{description}</div>
      )}
      {action && (
        <Button onClick={action.onClick}>{action.label}</Button>
      )}
    </div>
  );
}
