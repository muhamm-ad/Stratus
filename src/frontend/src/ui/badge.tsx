import { type HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  variant?: "default" | "outline";
}

export function Badge({ className, variant = "default", ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-[5px] px-2 py-0.5 text-[12px] font-semibold",
        variant === "outline" && "border border-border bg-transparent text-muted-foreground",
        className,
      )}
      {...props}
    />
  );
}
