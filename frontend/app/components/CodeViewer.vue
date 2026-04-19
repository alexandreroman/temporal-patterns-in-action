<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from "vue";
import { createHighlighter, type ThemedToken } from "shiki";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

const props = defineProps<{
  sources: Record<string, CodeSource>;
  highlight: [number, number] | null;
}>();

const lang = useCodeLang();

const highlighter = await createHighlighter({
  themes: ["github-light", "github-dark"],
  langs: ["go", "java", "python", "typescript"],
});

const TOKENIZED = computed<Record<string, ThemedToken[][]>>(() => {
  const out: Record<string, ThemedToken[][]> = {};
  for (const [key, src] of Object.entries(props.sources)) {
    out[key] = highlighter.codeToTokens(src.lines.join("\n"), {
      lang: key as CodeLang,
      themes: { light: "github-light", dark: "github-dark" },
      defaultColor: false,
    }).tokens;
  }
  return out;
});

const currentTokens = computed(() => TOKENIZED.value[lang.value] ?? []);

const gutterWidth = computed(() => `${String(currentTokens.value.length).length}ch`);

const scrollerRef = ref<HTMLElement | null>(null);
const lineRefs = ref<(HTMLElement | null)[]>([]);

const fullscreen = ref(false);

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === "Escape") fullscreen.value = false;
};

watch(fullscreen, (on) => {
  document.body.style.overflow = on ? "hidden" : "";
  if (on) window.addEventListener("keydown", onKeydown);
  else window.removeEventListener("keydown", onKeydown);
});

onBeforeUnmount(() => {
  document.body.style.overflow = "";
  window.removeEventListener("keydown", onKeydown);
});

// Why: scrollIntoView() would scroll the page too; manually moving the scroller keeps the jump local.
watch(
  [lang, () => props.highlight],
  ([, highlight]) => {
    const scroller = scrollerRef.value;
    if (!scroller) return;

    if (highlight === null) {
      scroller.scrollTo({ top: 0, behavior: "smooth" });
      return;
    }

    const [start, end] = highlight;
    const startEl = lineRefs.value[start];
    const endEl = lineRefs.value[end] ?? startEl;
    if (!startEl || !endEl) return;

    const scrollerRect = scroller.getBoundingClientRect();
    const offset = startEl.getBoundingClientRect().top - scrollerRect.top + scroller.scrollTop;
    const rangeHeight = endEl.getBoundingClientRect().bottom - startEl.getBoundingClientRect().top;

    const visibleTop = scroller.scrollTop;
    const visibleBottom = visibleTop + scroller.clientHeight;
    if (offset >= visibleTop && offset + rangeHeight <= visibleBottom) return;

    const desired = offset - (scroller.clientHeight - rangeHeight) / 2;
    const max = Math.max(0, scroller.scrollHeight - scroller.clientHeight);
    const clamped = Math.max(0, Math.min(desired, max));
    scroller.scrollTo({ top: clamped, behavior: "smooth" });
  },
  { flush: "post" },
);
</script>

<template>
  <div
    class="overflow-hidden border border-slate-200 dark:border-slate-700"
    :class="fullscreen ? 'fixed inset-0 z-50 flex flex-col' : 'rounded-xl'"
  >
    <div class="flex border-b border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-900">
      <button
        v-for="(src, key) in sources"
        :key="key"
        class="px-4 py-2 text-xs font-mono transition-colors border-b-2"
        :class="
          key === lang
            ? 'border-blue-500 text-slate-900 dark:text-slate-100'
            : 'border-transparent text-slate-400 hover:text-slate-600 dark:hover:text-slate-300'
        "
        @click="lang = key as CodeLang"
      >
        {{ src.label }}
      </button>
      <button
        class="ml-auto px-3 py-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
        :aria-label="fullscreen ? 'Exit fullscreen' : 'Enter fullscreen'"
        :title="fullscreen ? 'Exit fullscreen' : 'Enter fullscreen'"
        @click="fullscreen = !fullscreen"
      >
        <svg
          aria-hidden="true"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="h-[14px] w-[14px]"
        >
          <template v-if="fullscreen">
            <path d="M4 14h6v6" />
            <path d="M20 10h-6V4" />
            <path d="M14 10l7-7" />
            <path d="M3 21l7-7" />
          </template>
          <template v-else>
            <path d="M15 3h6v6" />
            <path d="M9 21H3v-6" />
            <path d="M21 3l-7 7" />
            <path d="M3 21l7-7" />
          </template>
        </svg>
      </button>
    </div>
    <div
      ref="scrollerRef"
      class="shiki-code overflow-auto bg-white dark:bg-slate-900 p-4 font-mono leading-relaxed"
      :class="fullscreen ? 'flex-1 min-h-0 text-[15px]' : 'max-h-80 text-[11px]'"
    >
      <span
        v-for="(tokens, idx) in currentTokens"
        :key="idx"
        :ref="(el) => (lineRefs[idx] = el as HTMLElement | null)"
        class="flex whitespace-pre rounded px-2 py-px transition-colors duration-300"
        :class="
          highlight && idx >= highlight[0] && idx <= highlight[1]
            ? 'bg-blue-50 dark:bg-blue-950'
            : ''
        "
      >
        <span
          class="shrink-0 select-none pr-4 text-right tabular-nums text-slate-400 dark:text-slate-600"
          :style="{ width: gutterWidth }"
          aria-hidden="true"
          >{{ idx + 1 }}</span
        >
        <span class="shiki-line flex-1">
          <template v-if="tokens.length">
            <span v-for="(token, tIdx) in tokens" :key="tIdx" :style="token.htmlStyle">{{
              token.content
            }}</span>
          </template>
          <template v-else>&nbsp;</template>
        </span>
      </span>
    </div>
  </div>
</template>

<style>
/* Dual-theme Shiki emits `--shiki-light` / `--shiki-dark` CSS vars on every
   token. `defaultColor: false` means no inline color is set, so we pick one
   explicitly based on the ancestor `.dark` class (matches main.css). */
.shiki-code .shiki-line span {
  color: var(--shiki-light);
}
.dark .shiki-code .shiki-line span {
  color: var(--shiki-dark);
}
</style>
