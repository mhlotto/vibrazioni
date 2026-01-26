
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

the tool should have an option `--count` which outputs a non-negative integer representing the number of found entries (based on --type or other options used).

the tool should have an option `--only M[,N...]` which will only output the Nth entry found (or M, N, and ... entries). It indexes starting at 0, so if `--count` outputs `1`, then `--only 0` is the only thing that works.
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
| 007 | extract scripts with `--type "foobar" --id "zhang"` | in007.html | out007.txt|
| 008 | extract count of scripts with `--type "foobar" --count` | in008.html | out008.txt|
| 009 | extract 2nd fooba script  with `--type "foobar" --only 1` | in009.html | out009.txt|


# example kinda use

```
curl https://foobasite | python3 html-scripttags-extract.py --type "application/json" --clean | jq .
```
