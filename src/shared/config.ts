const mode = process.env.NEXT_PUBLIC_DATA_MODE || 'mock';
if (mode !== 'mock' && mode !== 'api') throw new Error('NEXT_PUBLIC_DATA_MODE must be mock or api');
export const config = {
  dataMode: mode,
  apiBaseUrl: (process.env.NEXT_PUBLIC_API_BASE_URL || '').replace(/\/$/, ''),
} as const;
