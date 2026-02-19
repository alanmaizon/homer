import type { AgentContext, RewriteMode } from "@homer/shared";
import type { LLMProvider } from "./LLMProvider.js";

const OPENAI_URL = "https://api.openai.com/v1/chat/completions";

export class OpenAIProvider implements LLMProvider {
  readonly name = "openai";

  constructor(private readonly apiKey: string, private readonly model: string) {}

  private async call(prompt: string): Promise<string> {
    const response = await fetch(OPENAI_URL, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.apiKey}`
      },
      body: JSON.stringify({
        model: this.model,
        messages: [{ role: "user", content: prompt }]
      })
    });

    if (!response.ok) {
      throw new Error(`OpenAI request failed (${response.status})`);
    }

    const payload = (await response.json()) as {
      choices?: Array<{ message?: { content?: string } }>;
    };

    return payload.choices?.[0]?.message?.content?.trim() || "";
  }

  async generateSummary(documents: AgentContext["documents"], instructions?: string): Promise<string> {
    const prompt = `Summarize the following documents for a Word add-in user.\n\n${documents
      .map((doc) => `# ${doc.title}\n${doc.content}`)
      .join("\n\n")}\n\nInstructions: ${instructions || "none"}`;
    return this.call(prompt);
  }

  async rewriteText(text: string, mode: RewriteMode | undefined, instructions?: string): Promise<string> {
    const prompt = `Rewrite the following text in mode '${mode || "professional"}'.\nText:\n${text}\nInstructions: ${instructions || "none"}`;
    return this.call(prompt);
  }
}
