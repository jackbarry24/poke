# poke

**poke** is a fast, minimal, curl replacement for the terminal. It gives you powerful HTTP debugging with a CLI-first workflow, file-backed requests, 
Vim (or any other editor) based editing, and smart request reuse.

---

## Features

- `-X`, `-d`, `-H` curl-style flags
- Load payloads from files: `-d @payload.json`
- Load payloads from stdin: `-d @-`
- `--edit` flag to open payloas in `$EDITOR` (prefilled if using `-d`)
- Save complete requests to disk: `--save myreq.json`
- Send previously saved requests: `--send myreq.json`
- Pretty, colorized output

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
poke -X POST -d @payload.json -H "Content-Type:application/json" https://api.example.com/data
```

Pipe from stdin
```bash
cat payload.json | poke -X POST -d @- https://api.example.com
```

Use editor
```bash
poke -X POST --edit -H "Content-Type:application/json" https://api.example.com
```

Save request
```bash
poke -X PUT -d @data.json --save update-user.json https://api.example.com/users/123
```

Re-send saved request
```bash
poke --send update-user.json
```

Re-send and override
```bash
poke --send update-user.json -X PATCH -d @patch.json
```