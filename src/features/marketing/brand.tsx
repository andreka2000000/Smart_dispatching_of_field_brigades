export function BrandMark({ size = 30 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 32 32" fill="none" aria-hidden="true">
      <path
        d="M5 25V7L16 19 27 7v18"
        stroke="currentColor"
        strokeWidth="2.1"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <circle cx="16" cy="19" r="2.6" fill="currentColor" />
      <circle cx="5" cy="25" r="2.6" fill="currentColor" />
      <circle cx="27" cy="25" r="2.6" fill="currentColor" />
    </svg>
  );
}
