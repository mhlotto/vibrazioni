#!/usr/bin/env python3
import difflib
import pathlib
import subprocess
import sys


ROOT = pathlib.Path(__file__).resolve().parent
TESTDATA = ROOT / "testdata"
SCRIPT = ROOT / "html-scripttags-extract.py"

CASES = [
    {
        "args": ["--id", "airgap-GPP"],
        "input": "in001.html",
        "expected": "out001.txt",
    },
    {
        "args": ["--id", "airgap-GPP", "--clean"],
        "input": "in002.html",
        "expected": "out002.txt",
    },
    {
        "args": ["--type", "application/json"],
        "input": "in003.html",
        "expected": "out003.txt",
    },
    {
        "args": ["--type", "application/json", "--clean"],
        "input": "in004.html",
        "expected": "out004.txt",
    },
    {
        "args": ["--type", "foobar"],
        "input": "in005.html",
        "expected": "out005.txt",
    },
    {
        "args": ["--type", "foobar", "--clean"],
        "input": "in006.html",
        "expected": "out006.txt",
    },
]


def run_case(case):
    cmd = [
        sys.executable,
        str(SCRIPT),
        *case["args"],
        "--file",
        str(TESTDATA / case["input"]),
    ]
    proc = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if proc.returncode != 0:
        raise RuntimeError(
            f"command failed: {' '.join(cmd)}\n{proc.stderr.strip()}"
        )
    expected = (TESTDATA / case["expected"]).read_text(encoding="utf-8")
    actual = proc.stdout
    return expected, actual, cmd


def main():
    failed = False
    for idx, case in enumerate(CASES, start=1):
        expected, actual, cmd = run_case(case)
        if expected != actual:
            failed = True
            sys.stderr.write(f"FAILED case {idx}: {' '.join(cmd)}\n")
            diff = difflib.unified_diff(
                expected.splitlines(keepends=True),
                actual.splitlines(keepends=True),
                fromfile=case["expected"],
                tofile="actual",
            )
            sys.stderr.writelines(diff)
            sys.stderr.write("\n")
        else:
            sys.stdout.write(f"ok case {idx}: {case['input']}\n")
    if failed:
        sys.exit(1)


if __name__ == "__main__":
    main()
