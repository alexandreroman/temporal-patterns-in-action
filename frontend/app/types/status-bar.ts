export type StatusTone = "idle" | "running" | "success" | "error";

/** What a pattern status bar derives from the event stream. */
export interface StatusState {
  tone: StatusTone;
  message: string;
}
