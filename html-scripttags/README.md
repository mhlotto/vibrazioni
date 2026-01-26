
python3 tool for extract `<script>...</script>` from HTML files.

use `python3 html-scripttags-extract.py --type "application/json"` to only
extract `<script type="application/json" ...>...` tags. That is, those with
attribute `type` as `application/json`. Similarly, we should add options for
filtering for other script tag attributes, such as:

- id
- src

the tool should allow for piping in html and to read from a file via `--file <path>`

the tool should have an option `--out <file>` to write the extracted stuffs to it

the tool should have an option `--clean` to remove the `<script...>` and `</script>` tags from the results

# tests

The directory `testdata` has test inputs and output. I describe them here:

| test id | description | input file | expected output file |
|---------|-------------|------------|----------------------|
| 001 | extract script with `--id "airgap-GPP"` | in001.html | out001.txt |
| 002 | extract script with `--id "airgap-GPP" --clean` | in002.html | out002.txt|
| 003 | extract script with `--type "application/json"` | in003.html | out003.txt|
| 004 | extract script with `--type "application/json" --clean` | in004.html | out004.txt|
| 005 | extract scripts with `--type "foobar"` | in005.html | out005.txt|
| 006 | extract scripts with `--type "foobar" --clean` | in006.html | out006.txt|
