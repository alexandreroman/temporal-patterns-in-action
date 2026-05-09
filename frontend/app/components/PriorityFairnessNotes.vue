<script setup lang="ts">
/**
 * Static informational panel — explains how the simulation maps onto
 * Temporal's `Priority` struct so a presenter can connect what's on screen
 * to the underlying SDK feature.
 */
</script>

<template>
  <div class="rounded-xl border border-slate-700 bg-slate-900/40 p-4">
    <h2 class="text-xs font-medium text-slate-300">how this maps to Temporal</h2>
    <p class="mt-2 text-[12px] leading-relaxed text-slate-400">
      In Temporal, every workflow start and every activity invocation can carry a
      <code class="font-mono text-slate-300">Priority</code>:
    </p>
    <ul class="mt-2 list-disc space-y-1 pl-5 text-[12px] leading-relaxed text-slate-400">
      <li>
        <code class="font-mono text-slate-300">PriorityKey</code> &mdash; an integer (1 = highest).
        Lower keys jump ahead of higher keys at the head of the task queue.
      </li>
      <li>
        <code class="font-mono text-slate-300">FairnessKey</code> &mdash; a partition string. Tasks
        are grouped by this key so no single partition starves others.
      </li>
      <li>
        <code class="font-mono text-slate-300">FairnessWeight</code> &mdash; a positive float;
        partitions with higher weight get a proportionally larger share of the worker pool.
      </li>
    </ul>
    <p class="mt-3 text-[12px] leading-relaxed text-slate-400">
      In this demo, <code class="font-mono text-slate-300">PriorityKey</code> is the
      <span class="text-slate-300">P0..P3</span> chip on every ticket;
      <code class="font-mono text-slate-300">FairnessKey</code> is the
      <span class="text-slate-300">tenant ID</span>;
      <code class="font-mono text-slate-300">FairnessWeight</code> mirrors the
      <span class="text-slate-300">contract tier</span> (Enterprise=10, Pro=3, Free=1). Toggling
      fairness off makes selection FIFO within each priority tier, so a flood from Acme can starve
      smaller tenants. Toggling fairness on makes the dispatcher pick the tenant minimising
      <code class="font-mono text-slate-300">served / weight</code>.
    </p>
    <pre
      class="mt-3 overflow-x-auto rounded-md border border-slate-800 bg-slate-950 p-3 font-mono text-[11px] leading-relaxed text-slate-300"
    ><code>priority := temporal.Priority{
    PriorityKey:    int(ticket.Priority),     // 1..4
    FairnessKey:    string(ticket.Tenant),    // "acme" | "brick" | "solo"
    FairnessWeight: tenantWeight[ticket.Tenant],
}</code></pre>
  </div>
</template>
