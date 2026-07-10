import { STATE_META } from "@/lib/bridge";
import type { VMState } from "@/types/domain";

interface StatusBadgeProps {
  state: VMState;
}

export function StatusBadge({ state }: StatusBadgeProps) {
  const meta = STATE_META[state] ?? STATE_META.unknown;
  return (
    <span
      className="inline-flex items-center gap-1.5 rounded-[5px] px-2 py-0.5 text-[12px] font-semibold"
      style={{
        color: meta.color,
        background: `${meta.color}1a`,
        border: `1px solid ${meta.color}40`,
      }}
    >
      <span
        className="inline-block rounded-full"
        style={{ width: 6, height: 6, background: meta.color, flexShrink: 0 }}
      />
      {meta.label}
    </span>
  );
}
