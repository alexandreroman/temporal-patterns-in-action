// 6-char base36 is plenty for a per-run ID in a demo.
export const randomSuffix = (): string => Math.random().toString(36).slice(2, 8);
