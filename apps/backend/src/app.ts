import cors from "cors";
import express from "express";
import type { Express } from "express";
import { v4 as uuidv4 } from "uuid";
import type { TaskRequest } from "@homer/shared";
import { createProvider } from "./providers/index.js";
import { Orchestrator } from "./orchestrator.js";

export function createApp(): Express {
  const app = express();
  app.use(cors());
  app.use(express.json({ limit: "1mb" }));

  app.use((req, res, next) => {
    const requestId = req.header("x-request-id") || uuidv4();
    res.setHeader("x-request-id", requestId);
    res.locals.requestId = requestId;
    next();
  });

  app.get("/api/health", (_req, res) => {
    res.json({ ok: true });
  });

  app.post("/api/task", async (req, res) => {
    const body = req.body as TaskRequest;

    if (!body?.task || !Array.isArray(body.documents)) {
      return res.status(400).json({ error: "Invalid task payload" });
    }

    try {
      const orchestrator = new Orchestrator(createProvider());
      const response = await orchestrator.run(res.locals.requestId as string, body);
      return res.json(response);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Unknown error";
      return res.status(500).json({ error: message, requestId: res.locals.requestId });
    }
  });

  return app;
}
