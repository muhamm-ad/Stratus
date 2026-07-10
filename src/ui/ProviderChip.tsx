import { PROVIDER_META } from "@/lib/bridge";
import type { ProviderID } from "@/types/domain";

interface ProviderChipProps {
  provider: ProviderID;
  size?: "sm" | "md";
}

export function ProviderChip({ provider, size = "md" }: ProviderChipProps) {
  const meta = PROVIDER_META[provider];
  const dotSize = size === "sm" ? 7 : 8;
  return (
    <span className="inline-flex items-center gap-1.5 text-[13px]">
      <span
        className="inline-block rounded-full"
        style={{ width: dotSize, height: dotSize, background: meta.color, flexShrink: 0 }}
      />
      {meta.label}
    </span>
  );
}

interface ProviderDotProps {
  provider: ProviderID;
  size?: number;
}

export function ProviderDot({ provider, size = 8 }: ProviderDotProps) {
  const meta = PROVIDER_META[provider];
  return (
    <span
      className="inline-block rounded-full"
      style={{ width: size, height: size, background: meta.color, flexShrink: 0 }}
    />
  );
}
