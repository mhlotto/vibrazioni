Simple scripts

- `80col`: Wrap a text file to 80 columns in place with `fold -s -w 80 ... | sponge ...`.
- `ask`: Send a prompt from args or stdin to an OpenAI-compatible chat endpoint.
- `pickcols`: Read stdin and print selected whitespace fields joined by a separator.
- `twask`: Do two chat turns against the same endpoint, keeping the first turn as context for the second.
- `zipxgit`: Run `zip` with your normal args and skip `./.git/` if it exists.
