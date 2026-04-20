import { onMounted, watch } from "vue";

export type CodeLang = "go" | "java" | "python" | "typescript";

const STORAGE_KEY = "tpa:code-lang";
const DEFAULT: CodeLang = "go";

function isCodeLang(value: unknown): value is CodeLang {
  return value === "go" || value === "java" || value === "python" || value === "typescript";
}

// Shared reactive state across pages + localStorage persistence across reloads.
// Reading localStorage happens in onMounted to avoid SSR/client hydration mismatch.
export function useCodeLang() {
  const lang = useState<CodeLang>("code-lang", () => DEFAULT);

  onMounted(() => {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (isCodeLang(stored)) lang.value = stored;

    watch(lang, (next) => {
      window.localStorage.setItem(STORAGE_KEY, next);
    });
  });

  return lang;
}
