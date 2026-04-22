import { onBeforeUnmount, ref, watch } from "vue";

/**
 * Tween a counter so viewers see it tick up (e.g. 50 → 100) instead of
 * snapping. Honors prefers-reduced-motion and snaps on reset (target <
 * previous), so starting a new run resets cleanly.
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
      if (reduceMotion || target < (previous ?? 0)) {
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
