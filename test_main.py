import json
import subprocess
import sys
import unittest

from main import review


class HomerCriticTests(unittest.TestCase):
    def test_pass_when_style_and_mode_match(self):
        result = review(
            {
                "requestId": "r1",
                "task": "summarize",
                "output": "- First point.\n- Second point.",
                "mode": "professional",
                "style": "bullet",
            }
        )
        self.assertTrue(result["pass"])
        self.assertEqual(result["issues"], [])
        self.assertIsNone(result["suggestedEdit"])

    def test_fail_on_wrong_style_with_suggested_edit(self):
        result = review(
            {
                "requestId": "r2",
                "task": "rewrite",
                "output": "This is one sentence. This is another sentence.",
                "mode": None,
                "style": "bullet",
            }
        )
        self.assertFalse(result["pass"])
        self.assertIn("Style mismatch", result["issues"][0])
        self.assertIsNotNone(result["suggestedEdit"])
        self.assertIn("- ", result["suggestedEdit"])

    def test_fail_on_shorter_mode_when_too_long(self):
        long_text = "word " * 100
        result = review(
            {
                "requestId": "r3",
                "task": "summarize",
                "output": long_text,
                "mode": "shorter",
                "style": "paragraph",
            }
        )
        self.assertFalse(result["pass"])
        self.assertIn("Too long", result["issues"][0])
        self.assertIsNotNone(result["suggestedEdit"])

    def test_cli_returns_only_json(self):
        payload = {
            "requestId": "r4",
            "task": "summarize",
            "output": "Clean professional paragraph.",
            "mode": "professional",
            "style": "paragraph",
        }
        completed = subprocess.run(
            [sys.executable, "main.py"],
            input=json.dumps(payload),
            text=True,
            capture_output=True,
            check=True,
        )
        parsed = json.loads(completed.stdout)
        self.assertEqual(parsed["requestId"], "r4")
        self.assertIn("pass", parsed)


if __name__ == "__main__":
    unittest.main()
