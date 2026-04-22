<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import type { EventEnvelope } from "~~/shared/events";

interface ChatMessage {
  id: string;
  role: "user" | "llm" | "tool" | "system" | "retry";
  content: string;
  toolName?: string;
}

const props = defineProps<{
  events: EventEnvelope[];
  pendingPrompt: string | null;
}>();

// Stable key for the initial user prompt so the fallback placeholder and the
// event-driven message render as the *same* DOM node. Otherwise the key flip
// (`pending-prompt` → envelope id) triggers a TransitionGroup leave+enter, and
// the `.msg-row` flash animation keeps the leaving element around for ~1s —
// briefly showing two copies of the same prompt before one disappears.
const USER_PROMPT_KEY = "user-prompt";

const messages = computed<ChatMessage[]>(() => {
  const out: ChatMessage[] = [];
  let seenUserPrompt = false;

  for (const env of props.events) {
    const data = env.data as Record<string, unknown>;
    switch (env.type) {
      case "agent.user.prompt": {
        const prompt = typeof data.prompt === "string" ? data.prompt : "";
        if (prompt) {
          out.push({ id: USER_PROMPT_KEY, role: "user", content: prompt });
          seenUserPrompt = true;
        }
        break;
      }
      case "agent.llm.responded": {
        const msg = (data.message ?? {}) as Record<string, unknown>;
        const content = typeof msg.content === "string" ? msg.content : "";
        const toolCall = data.toolCall as { name?: string } | null;
        out.push({
          id: env.id,
          role: "llm",
          content,
          toolName: toolCall?.name,
        });
        break;
      }
      case "agent.tool.completed": {
        const name = typeof data.name === "string" ? data.name : "";
        const output = typeof data.output === "string" ? data.output : "";
        out.push({ id: env.id, role: "tool", content: output, toolName: name });
        break;
      }
      case "agent.approval.received": {
        const approved = Boolean(data.approved);
        out.push({
          id: env.id,
          role: "system",
          content: approved ? "User approved the travel plan." : "User rejected the travel plan.",
        });
        break;
      }
      case "progress.step.failed": {
        if (data.step !== "call-llm") break;
        const attempt = typeof data.attempt === "number" ? data.attempt : 1;
        out.push({
          id: env.id,
          role: "retry",
          content:
            `LLM call failed on attempt ${attempt}. Temporal retries automatically ` +
            `— conversation history, tool results, and prior tokens are preserved, ` +
            `so the agent resumes exactly where it left off.`,
        });
        break;
      }
    }
  }

  // Fallback: user prompt hasn't propagated through NATS yet but the UI
  // already knows what was submitted. Keep the panel responsive.
  if (!seenUserPrompt && props.pendingPrompt) {
    out.unshift({ id: USER_PROMPT_KEY, role: "user", content: props.pendingPrompt });
  }

  return out;
});

const scroller = ref<HTMLElement | null>(null);

watch(
  () => messages.value.length,
  () => {
    void nextTick(() => {
      const el = scroller.value;
      if (el) el.scrollTop = el.scrollHeight;
    });
  },
);

const ROLE_LABEL: Record<ChatMessage["role"], string> = {
  user: "USER",
  llm: "LLM",
  tool: "TOOL",
  system: "SYSTEM",
  retry: "TEMPORAL",
};

const ROLE_CLASS: Record<ChatMessage["role"], string> = {
  user: "text-blue-600 dark:text-blue-300",
  llm: "text-violet-600 dark:text-violet-300",
  tool: "text-emerald-600 dark:text-emerald-300",
  system: "text-amber-600 dark:text-amber-300",
  retry: "text-emerald-700 dark:text-emerald-300",
};
</script>

<template>
  <div
    class="flex flex-col overflow-hidden rounded-xl border border-slate-200 dark:border-slate-700 lg:h-full"
  >
    <div
      class="border-b border-slate-200 px-4 py-2 text-xs font-medium text-slate-700 dark:border-slate-700 dark:text-slate-300"
    >
      Agent conversation
    </div>
    <div ref="scroller" class="h-72 overflow-y-auto lg:h-auto lg:min-h-0 lg:flex-1">
      <p v-if="messages.length === 0" class="py-6 text-center text-xs text-slate-400">
        No messages yet.
      </p>
      <TransitionGroup name="msg">
        <div
          v-for="m in messages"
          :key="m.id"
          class="msg-row border-b border-slate-100 px-4 py-2 text-[12px] last:border-0 dark:border-slate-800"
          :class="
            m.role === 'retry'
              ? 'border-l-2 border-l-emerald-400 bg-emerald-50/60 dark:border-l-emerald-500 dark:bg-emerald-950/30'
              : ''
          "
        >
          <div class="mb-0.5 text-[10px] font-semibold tracking-wide" :class="ROLE_CLASS[m.role]">
            {{ ROLE_LABEL[m.role] }}
          </div>
          <div
            class="leading-relaxed"
            :class="
              m.role === 'retry'
                ? 'text-emerald-900 dark:text-emerald-100'
                : 'text-slate-700 dark:text-slate-200'
            "
          >
            {{ m.content }}
          </div>
          <div
            v-if="m.toolName"
            class="mt-1 inline-block rounded-md bg-slate-100 px-1.5 py-0.5 font-mono text-[10px] text-slate-600 dark:bg-slate-800 dark:text-slate-300"
          >
            {{ m.role === "llm" ? "tool_use" : "result" }}: {{ m.toolName }}
          </div>
        </div>
      </TransitionGroup>
    </div>
  </div>
</template>

<style scoped>
.msg-enter-active {
  transition:
    opacity 0.3s ease-out,
    transform 0.3s ease-out;
}
.msg-enter-from {
  opacity: 0;
  transform: translateY(-4px);
}
.msg-row {
  animation: msg-flash 1s ease-out;
}
@keyframes msg-flash {
  0% {
    background-color: rgb(139 92 246 / 0.12);
  }
  100% {
    background-color: transparent;
  }
}
</style>
