import type { AgentContext, PlanStep } from "@homer/shared";
import type { LLMProvider } from "../providers/LLMProvider.js";

export class ExecutorAgent {
  constructor(private readonly provider: LLMProvider) {}

  async execute(step: PlanStep, context: AgentContext): Promise<string> {
    if (step.action === "summarize") {
      return this.provider.generateSummary(context.documents, context.instructions);
    }

    if (!context.text) {
      throw new Error("Rewrite requires text input");
    }

    return this.provider.rewriteText(context.text, context.mode, context.instructions);
  }
}
