import { type ButtonHTMLAttributes, forwardRef } from "react";
import { cn } from "@/lib/utils";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "subtle" | "danger" | "ghost";
  size?: "sm" | "md" | "lg";
}

const variants: Record<string, string> = {
  primary:   "bg-primary text-primary-foreground hover:opacity-90 border-transparent",
  secondary: "bg-transparent text-foreground border border-border hover:bg-raised",
  subtle:    "bg-transparent text-muted-foreground border-transparent hover:bg-raised hover:text-foreground",
  danger:    "bg-danger text-white border-transparent hover:opacity-90",
  ghost:     "bg-transparent text-muted-foreground border-transparent hover:text-foreground",
};

const sizes: Record<string, string> = {
  sm:  "h-7 px-3 text-[12px] font-semibold rounded-[6px]",
  md:  "h-8 px-4 text-[13px] font-semibold rounded-[6px]",
  lg:  "h-10 px-5 text-[14px] font-semibold rounded-[6px]",
};

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant = "primary", size = "md", ...props }, ref) => (
    <button
      ref={ref}
      className={cn(
        "inline-flex items-center justify-center gap-2 transition-all duration-150",
        "disabled:opacity-60 disabled:cursor-not-allowed",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        variants[variant],
        sizes[size],
        className,
      )}
      {...props}
    />
  )
);

Button.displayName = "Button";
