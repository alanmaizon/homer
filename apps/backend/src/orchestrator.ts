import type { AgentContext, TaskRequest, TaskResponse } from "@homer/shared";
import { PlannerAgent } from "./agents/plannerAgent.js";
import { ExecutorAgent } from "./agents/executorAgent.js";
import { CriticAgent } from "./agents/criticAgent.js";
import type { LLMProvider } from "./providers/LLMProvider.js";

export class Orchestrator {
  private readonly planner = new PlannerAgent();
  private readonly critic = new CriticAgent();

  constructor(private readonly provider: LLMProvider) {}

  async run(requestId: string, request: TaskRequest): Promise<TaskResponse> {
    const startedAt = Date.now();
    const context: AgentContext = {
      requestId,
      task: request.task,
      documents: request.documents,
      text: request.text,
      mode: request.mode,
      instructions: request.instructions
    };

    const plan = this.planner.plan(context);
    const executor = new ExecutorAgent(this.provider);

    let result = "";
    for (const step of plan) {
      result = await executor.execute(step, context);
    }

    if (request.enableCritic) {
      result = this.critic.review(result);
    }

    return {
      result,
      plan,
      metadata: {
        provider: this.provider.name,
        executionTimeMs: Date.now() - startedAt,
        requestId,
        criticEnabled: Boolean(request.enableCritic)
      }
    };
  }
}
