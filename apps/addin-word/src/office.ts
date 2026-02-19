declare const Word: any;

function hasWordRuntime(): boolean {
  return typeof Word !== "undefined" && typeof Word.run === "function";
}

export async function getDocumentText(): Promise<string> {
  if (!hasWordRuntime()) {
    return "Demo document text from browser mode.";
  }

  return Word.run(async (context: any) => {
    const body = context.document.body;
    body.load("text");
    await context.sync();
    return body.text as string;
  });
}

export async function getSelectionText(): Promise<string> {
  if (!hasWordRuntime()) {
    return "Demo selected text.";
  }

  return Word.run(async (context: any) => {
    const selection = context.document.getSelection();
    selection.load("text");
    await context.sync();
    return selection.text as string;
  });
}

export async function insertTextAtCursor(text: string): Promise<void> {
  if (!hasWordRuntime()) {
    return;
  }

  await Word.run(async (context: any) => {
    const selection = context.document.getSelection();
    selection.insertText(text, "Start");
    await context.sync();
  });
}

export async function replaceSelection(text: string): Promise<void> {
  if (!hasWordRuntime()) {
    return;
  }

  await Word.run(async (context: any) => {
    const selection = context.document.getSelection();
    selection.insertText(text, "Replace");
    await context.sync();
  });
}
