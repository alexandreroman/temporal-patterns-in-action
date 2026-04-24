<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

type HighlightKey = "init" | "llmCall" | "toolCheck" | "toolExec" | "approval" | "finalAnswer";

interface AgentSource extends CodeSource {
  hl: Record<HighlightKey, [number, number]>;
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const lang = useCodeLang();

const SOURCES: Record<CodeLang, AgentSource> = {
  go: {
    label: "Go",
    lines: [
      "func TravelAgentWorkflow(ctx workflow.Context, req UserRequest) (Plan, error) {",
      "    history := []Message{{Role: RoleUser, Content: req.Prompt}}",
      '    approvalCh := workflow.GetSignalChannel(ctx, "approval")',
      "    var a *Activities",
      "",
      "    for i := 1; i <= maxIterations; i++ {",
      "        // Activity: call LLM with conversation + scenario",
      "        var resp LLMResponse",
      "        if err := workflow.ExecuteActivity(ctx,",
      "            a.CallLLM, LLMRequest{Scenario: req.Scenario,",
      "                Loop: i, History: history},",
      "        ).Get(ctx, &resp); err != nil {",
      "            return Plan{}, err",
      "        }",
      "        history = append(history, resp.Message)",
      "",
      "        // LLM wants to use an MCP tool",
      "        if resp.ToolCall != nil {",
      "            var result ToolResult",
      "            if err := workflow.ExecuteActivity(ctx,",
      "                a.ExecuteMCPTool, *resp.ToolCall,",
      "            ).Get(ctx, &result); err != nil {",
      "                return Plan{}, err",
      "            }",
      "            history = append(history, toolMsg(result))",
      "            continue",
      "        }",
      "",
      "        // LLM needs human approval — suspend on a signal",
      "        if resp.NeedsApproval {",
      "            var decision ApprovalDecision",
      "            approvalCh.Receive(ctx, &decision)",
      "            if !decision.Approved {",
      '                return Plan{}, temporal.NewNonRetryableApplicationError("rejected", "AgentRejected", nil)',
      "            }",
      "            continue",
      "        }",
      "",
      "        // Final plan returned",
      "        if resp.Plan != nil {",
      "            return *resp.Plan, nil",
      "        }",
      "    }",
      '    return Plan{}, fmt.Errorf("agent exceeded %d iterations", maxIterations)',
      "}",
    ],
    hl: {
      init: [0, 3],
      llmCall: [6, 14],
      toolCheck: [16, 17],
      toolExec: [18, 25],
      approval: [28, 36],
      finalAnswer: [38, 40],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public Plan travelAgent(UserRequest req) {",
      "    List<Message> history = new ArrayList<>();",
      "    history.add(Message.user(req.prompt()));",
      "",
      "    for (int i = 1; i <= MAX_ITERATIONS; i++) {",
      "        // Activity: call LLM with conversation + scenario",
      "        LLMResponse resp = activities.callLLM(",
      "            new LLMRequest(req.scenario(), i, history));",
      "        history.add(resp.message());",
      "",
      "        // LLM wants to use an MCP tool",
      "        if (resp.toolCall() != null) {",
      "            ToolResult result = activities.executeMCPTool(resp.toolCall());",
      "            history.add(Message.tool(result));",
      "            continue;",
      "        }",
      "",
      "        // LLM needs human approval — block on a signal",
      "        if (resp.needsApproval()) {",
      "            Workflow.await(() -> approval != null);",
      "            ApprovalDecision d = approval;",
      "            approval = null;",
      "            if (!d.approved()) {",
      '                throw ApplicationFailure.newNonRetryableFailure("rejected", "AgentRejected");',
      "            }",
      "            continue;",
      "        }",
      "",
      "        // Final plan returned",
      "        if (resp.plan() != null) {",
      "            return resp.plan();",
      "        }",
      "    }",
      "    throw new MaxIterationsError();",
      "}",
    ],
    hl: {
      init: [0, 3],
      llmCall: [6, 9],
      toolCheck: [11, 12],
      toolExec: [13, 16],
      approval: [18, 27],
      finalAnswer: [29, 31],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { proxyActivities, defineSignal, setHandler, condition } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const a = proxyActivities<typeof activities>({",
      '    startToCloseTimeout: "5 seconds",',
      "});",
      "",
      'const approvalSignal = defineSignal<[{ approved: boolean }]>("approval");',
      "",
      "export async function travelAgentWorkflow(req: UserRequest): Promise<Plan> {",
      '    const history: Message[] = [{ role: "user", content: req.prompt }];',
      "    let approval: { approved: boolean } | null = null;",
      "    setHandler(approvalSignal, (d) => { approval = d; });",
      "",
      "    for (let i = 1; i <= MAX_ITERATIONS; i++) {",
      "        // Activity: call LLM with conversation + scenario",
      "        const resp = await a.callLLM({ scenario: req.scenario, loop: i, history });",
      "        history.push(resp.message);",
      "",
      "        // LLM wants to use an MCP tool",
      "        if (resp.toolCall) {",
      "            const result = await a.executeMCPTool(resp.toolCall);",
      '            history.push({ role: "tool", content: result.output, toolName: result.name });',
      "            continue;",
      "        }",
      "",
      "        // LLM needs human approval — block on a signal",
      "        if (resp.needsApproval) {",
      "            await condition(() => approval !== null);",
      '            if (!approval!.approved) throw new Error("AgentRejected");',
      "            approval = null;",
      "            continue;",
      "        }",
      "",
      "        // Final plan returned",
      "        if (resp.plan) return resp.plan;",
      "    }",
      '    throw new Error("maxIterations exceeded");',
      "}",
    ],
    hl: {
      init: [0, 12],
      llmCall: [15, 17],
      toolCheck: [19, 20],
      toolExec: [21, 24],
      approval: [26, 32],
      finalAnswer: [34, 35],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class TravelAgentWorkflow:",
      "    def __init__(self) -> None:",
      "        self._approval: ApprovalDecision | None = None",
      "",
      '    @workflow.signal(name="approval")',
      "    def approval(self, decision: ApprovalDecision) -> None:",
      "        self._approval = decision",
      "",
      "    @workflow.run",
      "    async def run(self, req: UserRequest) -> Plan:",
      '        history = [Message(role="user", content=req.prompt)]',
      "",
      "        for i in range(1, MAX_ITERATIONS + 1):",
      "            # Activity: call LLM with conversation + scenario",
      "            resp = await workflow.execute_activity(",
      "                call_llm, LLMRequest(req.scenario, i, history),",
      "                start_to_close_timeout=timedelta(seconds=5),",
      "            )",
      "            history.append(resp.message)",
      "",
      "            # LLM wants to use an MCP tool",
      "            if resp.tool_call is not None:",
      "                result = await workflow.execute_activity(",
      "                    execute_mcp_tool, resp.tool_call,",
      "                    start_to_close_timeout=timedelta(seconds=5),",
      "                )",
      "                history.append(Message.tool(result))",
      "                continue",
      "",
      "            # LLM needs human approval — block on a signal",
      "            if resp.needs_approval:",
      "                await workflow.wait_condition(lambda: self._approval is not None)",
      "                if not self._approval.approved:",
      '                    raise ApplicationError("rejected", non_retryable=True)',
      "                self._approval = None",
      "                continue",
      "",
      "            # Final plan returned",
      "            if resp.plan is not None:",
      "                return resp.plan",
      '        raise RuntimeError("maxIterations exceeded")',
    ],
    hl: {
      init: [0, 11],
      llmCall: [14, 19],
      toolCheck: [21, 22],
      toolExec: [23, 28],
      approval: [30, 36],
      finalAnswer: [38, 40],
    },
  },
};

const highlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  const resolve = (key: HighlightKey) => src.hl[key] ?? null;

  for (let i = props.events.length - 1; i >= 0; i--) {
    const env = props.events[i];
    if (!env) continue;
    const data = env.data as Record<string, unknown>;

    switch (env.type) {
      case "progress.workflow.completed":
      case "progress.workflow.failed":
      case "agent.plan.ready":
        return resolve("finalAnswer");
      case "agent.approval.requested":
      case "agent.approval.received":
        return resolve("approval");
      case "agent.tool.started":
      case "agent.tool.completed":
        return resolve("toolExec");
      case "agent.llm.responded": {
        const toolCall = data.toolCall;
        if (toolCall !== null && toolCall !== undefined) return resolve("toolCheck");
        if (data.needsApproval) return resolve("approval");
        if (data.plan) return resolve("finalAnswer");
        return resolve("llmCall");
      }
      case "progress.step.started":
        if (data.step === "call-llm") return resolve("llmCall");
        if (data.step === "execute-mcp-tool") return resolve("toolExec");
        break;
    }
  }
  return null;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="highlight" />
</template>
