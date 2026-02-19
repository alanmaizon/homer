import type { AgentContext, RewriteMode } from "@homer/shared";

export interface LLMProvider {
  readonly name: string;
  generateSummary(documents: AgentContext["documents"], instructions?: string): Promise<string>;
  rewriteText(text: string, mode: RewriteMode | undefined, instructions?: string): Promise<string>;
}
