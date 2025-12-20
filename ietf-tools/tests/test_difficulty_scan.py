import importlib.util
import sys
from pathlib import Path
import unittest


def load_module(name, rel_path):
    base_dir = Path(__file__).resolve().parents[1]
    path = base_dir / rel_path
    spec = importlib.util.spec_from_file_location(name, path)
    module = importlib.util.module_from_spec(spec)
    sys.modules[name] = module
    spec.loader.exec_module(module)
    return module


scan = load_module("difficulty_scan", "ietf-draft-difficulty-scan.py")


class TestRfc2119Counting(unittest.TestCase):
    def test_counts_without_double_counting(self):
        text = "MUST NOT MUST SHOULD NOT SHOULD"
        counts = scan.count_rfc2119(text)
        self.assertEqual(counts["MUST NOT"], 1)
        self.assertEqual(counts["MUST"], 1)
        self.assertEqual(counts["SHOULD NOT"], 1)
        self.assertEqual(counts["SHOULD"], 1)
        self.assertEqual(counts["TOTAL"], 4)


class TestGrammarDetection(unittest.TestCase):
    def test_detects_json(self):
        lines = ['{ "key": true, "x": 1 }']
        text = "\n".join(lines)
        kinds = scan.detect_grammar_kind(text, lines)
        self.assertIn("json", kinds)


class TestSeverityScoring(unittest.TestCase):
    def test_single_grammar_signal_scoring(self):
        metrics = {"grammar_kinds": ["abnf"]}
        score = scan.compute_severity(metrics, [])
        self.assertEqual(score, 18)


if __name__ == "__main__":
    unittest.main()
