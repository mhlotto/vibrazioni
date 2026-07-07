#!/bin/bash

python3 -m venv kilnenv
source kilnenv/bin/activate
kilnenv/bin/python -m ensurepip --upgrade
kilnenv/bin/python -m  pip  install --upgrade pip setuptools wheel
kilnenv/bin/python -m  pip  install --no-build-isolation -e '.[test]'
kilnenv/bin/kiln -h

