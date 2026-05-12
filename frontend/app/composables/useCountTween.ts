import { onBeforeUnmount, ref, watch } from "vue";

/**
 * Tween a counter so viewers see it tick up (e.g. 50 → 100) instead of
 * snapping. Honors prefers-reduced-motion, snaps on reset (target <
 * previous), and snaps when the page is hidden — Chrome pauses
 * requestAnimationFrame on hidden tabs, so the tween would otherwise stay
 * pinned at 0 for the entire run.
 */
export function useCountTween(source: () => number) {
  const displayed = ref(0);
  let frame: number | null = null;
  const cancel = () => {
    if (frame !== null) {
      cancelAnimationFrame(frame);
      frame = null;
    }
  };

  watch(
    source,
    (target, previous) => {
      cancel();
      const from = displayed.value;
      if (target === from) return;

      const reduceMotion =
        typeof window !== "undefined" &&
        window.matchMedia?.("(prefers-reduced-motion: reduce)").matches;
      const hidden = typeof document !== "undefined" && document.hidden;
      if (hidden || reduceMotion || target < (previous ?? 0)) {
        displayed.value = target;
        return;
      }

      const delta = target - from;
      // Scale duration with jump size so big leaps feel proportional, capped at 800ms.
      const duration = Math.min(800, 300 + Math.min(delta, 500));
      const start = performance.now();

      const step = (now: number) => {
        const t = Math.min(1, (now - start) / duration);
        const eased = 1 - Math.pow(1 - t, 3); // easeOutCubic
        displayed.value = Math.round(from + delta * eased);
        if (t < 1) {
          frame = requestAnimationFrame(step);
        } else {
          frame = null;
        }
      };
      frame = requestAnimationFrame(step);
    },
    { immediate: true },
  );

  onBeforeUnmount(cancel);
  return displayed;
}
