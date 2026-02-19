import type { LLMProvider } from "./LLMProvider.js";
import { MockProvider } from "./MockProvider.js";
import { OpenAIProvider } from "./OpenAIProvider.js";

export function createProvider(): LLMProvider {
  if (process.env.LLM_PROVIDER === "openai" && process.env.OPENAI_API_KEY) {
    return new OpenAIProvider(process.env.OPENAI_API_KEY, process.env.OPENAI_MODEL || "gpt-4o-mini");
  }
  return new MockProvider();
}
