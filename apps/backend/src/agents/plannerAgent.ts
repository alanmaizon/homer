import type { AgentContext, PlanStep } from "@homer/shared";

export class PlannerAgent {
  plan(context: AgentContext): PlanStep[] {
    return [
      {
        id: `${context.task}-step-1`,
        role: "executor",
        action: context.task,
        inputRef: context.task === "summarize" ? "documents" : "text"
      }
    ];
  }
}
