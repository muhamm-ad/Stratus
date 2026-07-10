/** Stratus brand mark — the 3-cloud node graph */
export function StratusMark({ size = 64 }: { size?: number }) {
  return (
    <svg viewBox="0 0 64 64" width={size} height={size} xmlns="http://www.w3.org/2000/svg">
      <g fill="none" stroke="#94A3B8" strokeWidth="3" strokeLinecap="round">
        <path d="M16 17 L32 45" />
        <path d="M32 14 L32 45" />
        <path d="M48 17 L32 45" />
      </g>
      <circle cx="16" cy="16" r="4.5" fill="#FF9900" />
      <circle cx="32" cy="13" r="4.5" fill="#0078D4" />
      <circle cx="48" cy="16" r="4.5" fill="#EA4335" />
      <circle cx="32" cy="47" r="6.5" fill="#16A34A" />
    </svg>
  );
}

/** Icon: Stratus mark on indigo rounded square */
export function StratusIcon({ size = 28 }: { size?: number }) {
  // const rx = Math.round(size * 0.25);
  // return (
  //   <svg viewBox="0 0 28 28" width={size} height={size} xmlns="http://www.w3.org/2000/svg" style={{ flexShrink: 0 }}>
  //     <rect width="28" height="28" rx={rx} fill="var(--primary)" />
  //     {/* mark centered: viewBox 64, scale to 20px at centre of 28px square → offset 4 */}
  //     <g transform="translate(4, 3) scale(0.3125)">
  //       <g fill="none" stroke="#C7D2FE" strokeWidth="3" strokeLinecap="round">
  //         <path d="M16 17 L32 45" />
  //         <path d="M32 14 L32 45" />
  //         <path d="M48 17 L32 45" />
  //       </g>
  //       <circle cx="16" cy="16" r="4.5" fill="white" />
  //       <circle cx="32" cy="13" r="4.5" fill="white" />
  //       <circle cx="48" cy="16" r="4.5" fill="white" />
  //       <circle cx="32" cy="47" r="6.5" fill="white" />
  //     </g>
  //   </svg>
  // );
  return <StratusMark size={size} />;
}
