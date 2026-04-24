/**
 * Human-readable labels for the four backing services used by the batch
 * pattern. Shared by `BatchEventStream` (event log) and `BatchSlots`
 * (active-slot chips) so they stay consistent.
 */
export const BATCH_SERVICE_LABEL: Record<string, string> = {
  resize: "Resize",
  thumbnail: "Thumbnail",
  cdn: "CDN",
  metadata: "Metadata",
};
