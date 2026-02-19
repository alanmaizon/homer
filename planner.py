import json
import sys


def _non_empty_text(value):
    return isinstance(value, str) and value.strip() != ""


def _document_refs(documents):
    return [{"id": doc.get("id", ""), "title": doc.get("title", "")} for doc in documents]


def _clarification_response(request_id, task, missing_fields):
    return {
        "requestId": request_id,
        "task": task,
        "plan": [
            {
                "id": "step-1",
                "role": "request_clarification",
                "action": "clarify",
                "inputs": {
                    "documentRefs": [],
                    "textRef": None,
                    "mode": None,
                    "style": None,
                    "instructions": f"Missing required field(s): {', '.join(missing_fields)}",
                },
            }
        ],
        "notes": {
            "privacy": "Do not include source document content verbatim; use references only.",
            "assumptions": [],
        },
    }


def build_plan(payload):
    request_id = payload.get("requestId", "")
    task = payload.get("task")
    documents = payload.get("documents") or []
    text = payload.get("text")
    mode = payload.get("mode")
    style = payload.get("style")
    instructions = payload.get("instructions")
    enable_critic = bool(payload.get("enableCritic"))

    if task == "summarize" and len(documents) < 1:
        return _clarification_response(request_id, task, ["documents"])
    if task == "rewrite" and mode is None:
        return _clarification_response(request_id, task, ["mode"])

    if task == "summarize":
        step_inputs = {
            "documentRefs": _document_refs(documents),
            "textRef": None,
            "mode": mode,
            "style": style,
            "instructions": instructions,
        }
        assumptions = ["Summarization uses document references only."]
    else:
        step_inputs = {
            "documentRefs": [],
            "textRef": "provided_text" if _non_empty_text(text) else "selection",
            "mode": mode,
            "style": style,
            "instructions": instructions,
        }
        assumptions = ["If text is not provided, rewrite uses the current selection."]

    plan = [
        {
            "id": "step-1",
            "role": "executor",
            "action": task,
            "inputs": step_inputs,
        }
    ]

    if enable_critic:
        plan.append(
            {
                "id": f"step-{len(plan) + 1}",
                "role": "critic",
                "action": "critique",
                "inputs": step_inputs,
            }
        )
        if task == "summarize":
            assumptions.append("Critic checks clarity, brevity, and sensitive leakage.")
        else:
            assumptions.append("Critic checks meaning preservation and mode adherence.")

    return {
        "requestId": request_id,
        "task": task,
        "plan": plan,
        "notes": {
            "privacy": "Do not include source document content verbatim; use references only.",
            "assumptions": assumptions,
        },
    }


def main():
    payload = json.load(sys.stdin)
    print(json.dumps(build_plan(payload), separators=(",", ":")))


if __name__ == "__main__":
    main()
