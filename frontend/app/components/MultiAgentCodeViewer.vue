<script setup lang="ts">
import { computed } from "vue";
import type { EventEnvelope } from "~~/shared/events";
import type { CodeLang } from "~/composables/useCodeLang";
import type { CodeSource } from "~/types/code-viewer";

type StepKey = "plan" | "queries" | "fanout" | "fanin" | "synth";

interface MultiAgentSource extends CodeSource {
  stepLines: Record<StepKey, [number, number]>;
}

const props = defineProps<{
  events: EventEnvelope[];
}>();

const SOURCES: Record<CodeLang, MultiAgentSource> = {
  go: {
    label: "Go",
    lines: [
      "func DeepResearchWorkflow(ctx workflow.Context, req Request) (Report, error) {",
      "    var a *Activities",
      "",
      "    // Phase 1 — Plan: LLM decomposes the prompt into subtopics.",
      "    var plan ResearchPlan",
      "    if err := workflow.ExecuteActivity(ctx, a.PlanResearch, req).",
      "        Get(ctx, &plan); err != nil {",
      "        return Report{}, err",
      "    }",
      "",
      "    // Phase 2 — Query gen: one LLM call per subtopic.",
      "    var queries ResearchQueries",
      "    if err := workflow.ExecuteActivity(ctx, a.GenerateQueries, plan).",
      "        Get(ctx, &queries); err != nil {",
      "        return Report{}, err",
      "    }",
      "",
      "    // Phase 3 — Fan-out: one child workflow per subtopic.",
      "    futures := make([]workflow.ChildWorkflowFuture, len(queries.Topics))",
      "    for i, tq := range queries.Topics {",
      "        futures[i] = workflow.ExecuteChildWorkflow(ctx,",
      "            ResearchAgentWorkflow,",
      "            AgentInput{TopicIndex: i, TopicName: tq.TopicName,",
      "                Queries: tq.Queries})",
      "    }",
      "",
      "    // Fan-in: tolerate per-child failures.",
      "    var ok []ResearchResult",
      "    for _, f := range futures {",
      "        var r ResearchResult",
      "        if err := f.Get(ctx, &r); err == nil {",
      "            ok = append(ok, r)",
      "        }",
      "    }",
      "",
      "    // Phase 4 — Synthesis: LLM merges the surviving results.",
      "    var report Report",
      "    if err := workflow.ExecuteActivity(ctx, a.SynthesizeReport,",
      "        SynthesisInput{Prompt: req.Prompt, Results: ok}).",
      "        Get(ctx, &report); err != nil {",
      "        return Report{}, err",
      "    }",
      "    return report, nil",
      "}",
      "",
      "func ResearchAgentWorkflow(ctx workflow.Context, in AgentInput) (ResearchResult, error) {",
      "    var a *Activities",
      "    result := ResearchResult{TopicIndex: in.TopicIndex, TopicName: in.TopicName}",
      "    failed := 0",
      "    for qi, q := range in.Queries {",
      "        var sr SearchResult",
      "        if err := workflow.ExecuteActivity(ctx, a.WebSearch,",
      "            SearchInput{TopicIndex: in.TopicIndex, QueryIndex: qi, Query: q}).",
      "            Get(ctx, &sr); err != nil {",
      "            failed++",
      "            continue",
      "        }",
      "        result.Sources = append(result.Sources, sr.Sources...)",
      "    }",
      "    if failed == len(in.Queries) {",
      '        return ResearchResult{}, fmt.Errorf("all searches failed")',
      "    }",
      "    result.Partial = failed > 0",
      "    return result, nil",
      "}",
    ],
    stepLines: {
      plan: [3, 8],
      queries: [10, 15],
      fanout: [17, 24],
      fanin: [26, 33],
      synth: [35, 41],
    },
  },
  java: {
    label: "Java",
    lines: [
      "@WorkflowMethod",
      "public Report deepResearch(Request req) {",
      "    // Phase 1 — Plan: LLM decomposes the prompt into subtopics.",
      "    ResearchPlan plan = activities.planResearch(req);",
      "",
      "    // Phase 2 — Query gen: one LLM call per subtopic.",
      "    ResearchQueries queries = activities.generateQueries(plan);",
      "",
      "    // Phase 3 — Fan-out: one child workflow per subtopic.",
      "    var futures = new ArrayList<Promise<ResearchResult>>();",
      "    for (var tq : queries.topics()) {",
      "        var child = Workflow.newChildWorkflowStub(ResearchAgentWorkflow.class);",
      "        futures.add(Async.function(child::researchAgent,",
      "            new AgentInput(tq.topicIndex(), tq.topicName(), tq.queries())));",
      "    }",
      "",
      "    // Fan-in: tolerate per-child failures.",
      "    var ok = new ArrayList<ResearchResult>();",
      "    for (var f : futures) {",
      "        try { ok.add(f.get()); }",
      "        catch (ChildWorkflowFailure e) { /* skip failed child */ }",
      "    }",
      "",
      "    // Phase 4 — Synthesis: LLM merges the surviving results.",
      "    return activities.synthesizeReport(",
      "        new SynthesisInput(req.prompt(), ok));",
      "}",
      "",
      "@WorkflowMethod",
      "public ResearchResult researchAgent(AgentInput in) {",
      "    var result = new ResearchResult(in.topicIndex(), in.topicName());",
      "    int failed = 0;",
      "    for (int qi = 0; qi < in.queries().size(); qi++) {",
      "        try {",
      "            var sr = activities.webSearch(",
      "                new SearchInput(in.topicIndex(), qi, in.queries().get(qi)));",
      "            result.addSources(sr.sources());",
      "        } catch (ActivityFailure e) {",
      "            failed++;",
      "        }",
      "    }",
      "    if (failed == in.queries().size()) {",
      '        throw ApplicationFailure.newFailure("all searches failed", "AllFailed");',
      "    }",
      "    result.setPartial(failed > 0);",
      "    return result;",
      "}",
    ],
    stepLines: {
      plan: [2, 3],
      queries: [5, 6],
      fanout: [8, 14],
      fanin: [16, 21],
      synth: [23, 25],
    },
  },
  typescript: {
    label: "TypeScript",
    lines: [
      'import { executeChild, proxyActivities } from "@temporalio/workflow";',
      'import type * as activities from "./activities";',
      "",
      "const a = proxyActivities<typeof activities>({",
      '    startToCloseTimeout: "10 seconds",',
      "});",
      "",
      "export async function deepResearchWorkflow(req: Request): Promise<Report> {",
      "    // Phase 1 — Plan: LLM decomposes the prompt into subtopics.",
      "    const plan = await a.planResearch(req);",
      "",
      "    // Phase 2 — Query gen: one LLM call per subtopic.",
      "    const queries = await a.generateQueries(plan);",
      "",
      "    // Phase 3 — Fan-out: one child workflow per subtopic.",
      "    const futures = queries.topics.map((tq) =>",
      "        executeChild(researchAgentWorkflow, {",
      "            args: [{ topicIndex: tq.topicIndex,",
      "                topicName: tq.topicName, queries: tq.queries }],",
      "        }),",
      "    );",
      "",
      "    // Fan-in: tolerate per-child failures.",
      "    const settled = await Promise.allSettled(futures);",
      "    const ok = settled",
      '        .filter((s): s is PromiseFulfilledResult<ResearchResult> => s.status === "fulfilled")',
      "        .map((s) => s.value);",
      "",
      "    // Phase 4 — Synthesis: LLM merges the surviving results.",
      "    return a.synthesizeReport({ prompt: req.prompt, results: ok });",
      "}",
      "",
      "export async function researchAgentWorkflow(in_: AgentInput): Promise<ResearchResult> {",
      "    const result: ResearchResult = {",
      "        topicIndex: in_.topicIndex, topicName: in_.topicName,",
      "        sources: [], partial: false,",
      "    };",
      "    let failed = 0;",
      "    for (let qi = 0; qi < in_.queries.length; qi++) {",
      "        try {",
      "            const sr = await a.webSearch({",
      "                topicIndex: in_.topicIndex, queryIndex: qi,",
      "                query: in_.queries[qi]!,",
      "            });",
      "            result.sources.push(...sr.sources);",
      "        } catch {",
      "            failed++;",
      "        }",
      "    }",
      "    if (failed === in_.queries.length) {",
      '        throw new Error("all searches failed");',
      "    }",
      "    result.partial = failed > 0;",
      "    return result;",
      "}",
    ],
    stepLines: {
      plan: [8, 9],
      queries: [11, 12],
      fanout: [14, 20],
      fanin: [22, 26],
      synth: [28, 29],
    },
  },
  python: {
    label: "Python",
    lines: [
      "@workflow.defn",
      "class DeepResearchWorkflow:",
      "    @workflow.run",
      "    async def run(self, req: Request) -> Report:",
      "        # Phase 1 — Plan: LLM decomposes the prompt into subtopics.",
      "        plan = await workflow.execute_activity(",
      "            plan_research, req,",
      "            start_to_close_timeout=timedelta(seconds=10))",
      "",
      "        # Phase 2 — Query gen: one LLM call per subtopic.",
      "        queries = await workflow.execute_activity(",
      "            generate_queries, plan,",
      "            start_to_close_timeout=timedelta(seconds=10))",
      "",
      "        # Phase 3 — Fan-out: one child workflow per subtopic.",
      "        handles = [",
      "            await workflow.start_child_workflow(",
      "                ResearchAgentWorkflow.run,",
      "                AgentInput(tq.topic_index, tq.topic_name, tq.queries))",
      "            for tq in queries.topics",
      "        ]",
      "",
      "        # Fan-in: tolerate per-child failures.",
      "        settled = await asyncio.gather(*handles, return_exceptions=True)",
      "        ok = [r for r in settled if not isinstance(r, Exception)]",
      "",
      "        # Phase 4 — Synthesis: LLM merges the surviving results.",
      "        return await workflow.execute_activity(",
      "            synthesize_report, SynthesisInput(req.prompt, ok),",
      "            start_to_close_timeout=timedelta(seconds=10))",
      "",
      "@workflow.defn",
      "class ResearchAgentWorkflow:",
      "    @workflow.run",
      "    async def run(self, in_: AgentInput) -> ResearchResult:",
      "        result = ResearchResult(in_.topic_index, in_.topic_name)",
      "        failed = 0",
      "        for qi, q in enumerate(in_.queries):",
      "            try:",
      "                sr = await workflow.execute_activity(",
      "                    web_search,",
      "                    SearchInput(in_.topic_index, qi, q),",
      "                    start_to_close_timeout=timedelta(seconds=10))",
      "                result.sources.extend(sr.sources)",
      "            except ActivityError:",
      "                failed += 1",
      "        if failed == len(in_.queries):",
      '            raise ApplicationError("all searches failed", non_retryable=True)',
      "        result.partial = failed > 0",
      "        return result",
    ],
    stepLines: {
      plan: [4, 7],
      queries: [9, 12],
      fanout: [14, 20],
      fanin: [22, 24],
      synth: [26, 29],
    },
  },
};

function latestRelevant(events: EventEnvelope[]): string | null {
  for (let i = events.length - 1; i >= 0; i--) {
    const env = events[i];
    if (!env) continue;
    const data = env.data as Record<string, unknown>;
    const step = typeof data.step === "string" ? data.step : "";
    const t = env.type;

    if (t === "progress.workflow.completed" || t === "progress.workflow.failed") return t;
    if (t === "multi-agent.plan.ready") return t;
    if (t === "multi-agent.queries.ready") return t;
    if (t === "multi-agent.fanout.started") return t;
    if (t === "multi-agent.search.started") return t;
    if (t === "multi-agent.search.completed") return t;
    if (t === "multi-agent.search.failed") return t;
    if (t === "multi-agent.child.completed") return t;
    if (t === "multi-agent.child.failed") return t;
    if (t === "multi-agent.report.ready") return t;
    if (t === "progress.step.started" || t === "progress.step.completed") {
      if (step === "plan-research") return "step.plan";
      if (step === "generate-queries") return "step.queries";
      if (step === "synthesize-report") return "step.synth";
    }
  }
  return null;
}

const lang = useCodeLang();

const currentHighlight = computed<[number, number] | null>(() => {
  const src = SOURCES[lang.value];
  const latest = latestRelevant(props.events);
  if (!latest) return null;

  switch (latest) {
    case "progress.workflow.completed":
    case "progress.workflow.failed":
      return null;
    case "step.plan":
    case "multi-agent.plan.ready":
      return src.stepLines.plan;
    case "step.queries":
    case "multi-agent.queries.ready":
      return src.stepLines.queries;
    case "multi-agent.fanout.started":
    case "multi-agent.search.started":
    case "multi-agent.search.completed":
    case "multi-agent.search.failed":
      return src.stepLines.fanout;
    case "multi-agent.child.completed":
    case "multi-agent.child.failed":
      return src.stepLines.fanin;
    case "step.synth":
    case "multi-agent.report.ready":
      return src.stepLines.synth;
  }
  return null;
});
</script>

<template>
  <CodeViewer :sources="SOURCES" :highlight="currentHighlight" />
</template>
