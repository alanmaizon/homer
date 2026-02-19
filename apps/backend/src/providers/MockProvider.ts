import type { AgentContext, RewriteMode } from "@homer/shared";
import type { LLMProvider } from "./LLMProvider.js";

export class MockProvider implements LLMProvider {
  readonly name = "mock";

  async generateSummary(documents: AgentContext["documents"], instructions?: string): Promise<string> {
    const joined = documents.map((doc) => `${doc.title}: ${doc.content}`).join(" ");
    const compact = joined.replace(/\s+/g, " ").trim();
    const baseSummary = compact.length > 240 ? `${compact.slice(0, 237)}...` : compact;
    return instructions ? `${baseSummary}\n\nInstruction note: ${instructions}` : baseSummary;
  }

  async rewriteText(text: string, mode: RewriteMode | undefined, instructions?: string): Promise<string> {
    const normalized = text.trim();
    let rewritten = normalized;
    if (mode === "shorter") rewritten = normalized.slice(0, Math.max(1, Math.floor(normalized.length * 0.7)));
    if (mode === "simplify") rewritten = normalized.replace(/\butilize\b/gi, "use");
    if (mode === "professional") rewritten = `Professional rewrite: ${normalized}`;
    return instructions ? `${rewritten}\n\nInstruction note: ${instructions}` : rewritten;
  }
}
