import { WorkflowNotFoundError } from "@temporalio/client";
import type { SagaProgress } from "~~/shared/types";

interface ProgressQueryResult {
  currentStep: string;
  completed: string[];
  failed?: string;
}

export default defineEventHandler(async (event): Promise<SagaProgress> => {
  const workflowId = getRouterParam(event, "id");
  if (!workflowId) {
    throw createError({ statusCode: 400, statusMessage: "workflow id is required" });
  }

  const client = await getTemporalClient();
  const handle = client.workflow.getHandle(workflowId);

  try {
    const [description, progress] = await Promise.all([
      handle.describe(),
      handle.query<ProgressQueryResult>("getProgress"),
    ]);

    return {
      workflowId,
      status: mapStatus(description.status.name),
      currentStep: progress.currentStep,
      completed: progress.completed,
      failed: progress.failed,
    };
  } catch (error) {
    if (error instanceof WorkflowNotFoundError) {
      throw createError({ statusCode: 404, statusMessage: "workflow not found" });
    }
    throw error;
  }
});

function mapStatus(status: string): SagaProgress["status"] {
  if (status === "COMPLETED") return "completed";
  if (status === "RUNNING") return "running";
  return "failed";
}
