# poke

**poke** is a fast, minimal, HTTP request sender for the terminal. It is not designed as a drop in curl replacement, and will not offer nearly the same amount of features as curl. It gives you powerful HTTP debugging with a CLI-first workflow, file-backed requests, 
Vim (or any other editor) based editing, and smart request reuse.

---

## Features

## Features

- `-X`, `-d`, `-H` curl-style flags
- Load payloads from files: `--data-file payload.json`
- Load payloads from stdin: `--data-stdin`
- `--edit` flag to open payloads in `$EDITOR` 
- Save complete requests to disk: `--save myreq.json`
- Send previously saved requests: `send <file|collection>`
- Pretty, colorized output
- Reusable request collections (`collections` command)
- Concurrency with `--workers` and `--repeat`
- Auto-verifies status codes via `--expect-status`

---

## Installation

```bash
git clone https://github.com/jackbarry24/poke.git
cd poke
go build -o poke ./src
```

And to make it global:
`mv poke /usr/local/bin`

---

## Usage

Basic request
```bash
poke -X POST -d '{"hello":"world"}' -H "Content-Type:application/json" https://httpbin.org/post
```

Load payload from file
```bash
poke -X POST --data-file payload.json -H "Content-Type:application/json" https://httpbin.org/post
```

Pipe from stdin
```bash
cat payload.json | poke -X POST --data-stdin -H "Content-Type:application/json" https://httpbin.org/post
```

Use editor
```bash
poke -X POST --edit -H "Content-Type:application/json" https://httpbin.org/post
```

Save request
```bash
poke -X PUT --data-file data.json --save test.json https://httpbin.org/post
```

Re-send saved request
```bash
poke send test.json
```

List collections
```bash
poke collections
```

View a collection
```bash
poke collections my_collection
```

Run a collection
```bash
poke send my_collection
```
