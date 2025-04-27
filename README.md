# poke

**poke** is a fast, minimal, HTTP request sender for the terminal. It is not designed as a drop in curl replacement, and will not offer nearly the same amount of features as curl. It let's you save requests as .json file and resend them easily from the command line. 

---

## Features

- `-X`, `-d`, `-H` curl-style flags
- Load payloads from files: `--data-file payload.json`
- Load payloads from stdin: `--data-stdin`
- `--edit` flag to open payloads in `$EDITOR` 
- Save complete requests to disk: `--save myreq.json`
- Send previously saved requests: `send <file|directory>`
- Pretty, colorized output
- Reusable request collections (`collections` command)
- Concurrency with `--workers` and `--repeat`
- Retry and backoff with `--retry` and `backoff`
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
poke -X GET --save myreq.json https://httpbin.org/get
```

Send request
```bash
poke send myreq.json
```
**Note** if you have multiple json request files in one directory you can run them all at once using `poke send path/`

Other features of note:
- Env templating: In any request json file you can use `{{ env.VAR }}` to fill in a secret/variable from env or a .env file.
- History templating: In any request json file you can do stuff like the following:  

`{{ history.body }}` to put the body of the response from the last request in your current request.  
`{{ index history.headers "Content-Length" 0 }}`   

. This allows you to chain responses together when running a collection. If you need to force the files to run in a certain order for the chaining to work simply name them `1_getuser.json`, `2_updateuser.json` and so on.
- Assertions: In each request json you can manually add status, body, and header assertions (status will be populated from `--expect-status`). This will cause the request to fail if the status doesn't match, etc.



