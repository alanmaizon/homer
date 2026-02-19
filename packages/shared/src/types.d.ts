export type AgentRole = "planner" | "executor" | "critic";
export type TaskType = "summarize" | "rewrite";
export type RewriteMode = "simplify" | "professional" | "shorter";
export interface DocumentInput {
    id: string;
    title: string;
    content: string;
}
export interface TaskRequest {
    task: TaskType;
    documents: DocumentInput[];
    text?: string;
    mode?: RewriteMode;
    instructions?: string;
    enableCritic?: boolean;
}
export interface PlanStep {
    id: string;
    role: AgentRole;
    action: TaskType;
    inputRef: "documents" | "text";
}
export interface AgentContext {
    requestId: string;
    task: TaskType;
    documents: DocumentInput[];
    text?: string;
    mode?: RewriteMode;
    instructions?: string;
}
export interface TaskResponse {
    result: string;
    plan: PlanStep[];
    metadata: {
        provider: string;
        executionTimeMs: number;
        requestId: string;
        criticEnabled: boolean;
    };
}
