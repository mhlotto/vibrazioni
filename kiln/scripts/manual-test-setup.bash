#!/bin/bash

rm -rf .venv

python3 -m venv .venv
. .venv/bin/activate

python -m ensurepip --upgrade
python -m pip install --upgrade pip setuptools wheel

python -m pip install -e ".[test]"
pytest

kiln --help


