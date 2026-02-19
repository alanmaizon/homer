import { useMemo, useState } from "react";
import type { DocumentInput, RewriteMode, TaskResponse } from "@homer/shared";
import { getDocumentText, getSelectionText, insertTextAtCursor, replaceSelection } from "./office";
import "./styles.css";

const modes: RewriteMode[] = ["simplify", "professional", "shorter"];

const createSnippet = (): DocumentInput => ({
  id: crypto.randomUUID(),
  title: "Snippet",
  content: ""
});

export default function App() {
  const [documents, setDocuments] = useState<DocumentInput[]>([createSnippet()]);
  const [mode, setMode] = useState<RewriteMode>("professional");
  const [instructions, setInstructions] = useState("");
  const [enableCritic, setEnableCritic] = useState(false);
  const [output, setOutput] = useState("");
  const [busy, setBusy] = useState(false);

  const canSubmit = useMemo(() => !busy, [busy]);

  const updateSnippet = (id: string, key: "title" | "content", value: string) => {
    setDocuments((current) => current.map((doc) => (doc.id === id ? { ...doc, [key]: value } : doc)));
  };

  const runTask = async (payload: Record<string, unknown>) => {
    setBusy(true);
    try {
      const response = await fetch("http://localhost:8080/api/task", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
      });
      const data = (await response.json()) as TaskResponse;
      setOutput(data.result || "");
    } finally {
      setBusy(false);
    }
  };

  const summarizeDocument = async () => {
    const documentText = await getDocumentText();
    const payloadDocs = [{ id: "word-body", title: "Word Document", content: documentText }, ...documents];
    await runTask({ task: "summarize", documents: payloadDocs, instructions, style: "paragraph", enableCritic });
  };

  const rewriteSelection = async () => {
    const text = await getSelectionText();
    await runTask({
      task: "rewrite",
      documents,
      text,
      mode,
      instructions,
      enableCritic
    });
  };

  return (
    <main className="pane">
      <h1>Homer</h1>
      <div className="actions">
        <button onClick={summarizeDocument} disabled={!canSubmit}>Summarize Document</button>
        <button onClick={rewriteSelection} disabled={!canSubmit}>Rewrite Selection</button>
      </div>

      <label>
        Rewrite mode
        <select value={mode} onChange={(event) => setMode(event.target.value as RewriteMode)}>
          {modes.map((item) => (
            <option key={item} value={item}>{item}</option>
          ))}
        </select>
      </label>

      <label>
        Instructions
        <input value={instructions} onChange={(event) => setInstructions(event.target.value)} placeholder="Optional instruction" />
      </label>

      <label className="toggle">
        <input type="checkbox" checked={enableCritic} onChange={(event) => setEnableCritic(event.target.checked)} />
        Enable Critic
      </label>

      <section>
        <h2>Multi-snippet inputs</h2>
        {documents.map((doc) => (
          <div key={doc.id} className="snippet">
            <input value={doc.title} onChange={(event) => updateSnippet(doc.id, "title", event.target.value)} />
            <textarea
              rows={3}
              value={doc.content}
              onChange={(event) => updateSnippet(doc.id, "content", event.target.value)}
            />
          </div>
        ))}
        <button onClick={() => setDocuments((current) => [...current, createSnippet()])}>Add snippet</button>
      </section>

      <section>
        <h2>Output preview</h2>
        <textarea rows={8} value={output} onChange={(event) => setOutput(event.target.value)} />
        <div className="actions">
          <button onClick={() => insertTextAtCursor(output)} disabled={!output}>Insert</button>
          <button onClick={() => replaceSelection(output)} disabled={!output}>Replace</button>
        </div>
      </section>
    </main>
  );
}
