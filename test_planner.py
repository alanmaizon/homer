import unittest

from planner import build_plan


class PlannerTests(unittest.TestCase):
    def test_summarize_requires_documents(self):
        result = build_plan(
            {
                "requestId": "r1",
                "task": "summarize",
                "documents": [],
                "text": None,
                "mode": None,
                "instructions": None,
                "style": "bullet",
                "enableCritic": False,
            }
        )
        self.assertEqual(result["plan"][0]["role"], "request_clarification")

    def test_summarize_with_critic(self):
        result = build_plan(
            {
                "requestId": "r2",
                "task": "summarize",
                "documents": [{"id": "d1", "title": "Doc", "content": "secret"}],
                "text": None,
                "mode": None,
                "instructions": "focus",
                "style": "bullet",
                "enableCritic": True,
            }
        )
        self.assertEqual(len(result["plan"]), 2)
        self.assertEqual(result["plan"][0]["role"], "executor")
        self.assertEqual(result["plan"][1]["role"], "critic")
        self.assertEqual(result["plan"][0]["inputs"]["documentRefs"], [{"id": "d1", "title": "Doc"}])

    def test_rewrite_requires_mode(self):
        result = build_plan(
            {
                "requestId": "r3",
                "task": "rewrite",
                "documents": [],
                "text": "hello",
                "mode": None,
                "instructions": None,
                "style": "paragraph",
                "enableCritic": False,
            }
        )
        self.assertEqual(result["plan"][0]["role"], "request_clarification")

    def test_rewrite_textref_fallback_to_selection(self):
        result = build_plan(
            {
                "requestId": "r4",
                "task": "rewrite",
                "documents": [],
                "text": None,
                "mode": "professional",
                "instructions": None,
                "style": "paragraph",
                "enableCritic": False,
            }
        )
        self.assertEqual(result["plan"][0]["inputs"]["textRef"], "selection")


if __name__ == "__main__":
    unittest.main()
