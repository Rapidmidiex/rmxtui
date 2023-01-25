# RMX TUI Client

A terminal user interface client for the Realtime MIDI eXchange server.

## Run Locally

Clone the project

```bash
$  git clone https://github.com/Rapidmidiex/rmxtui
```

Go to the project directory

```bash
$  cd rmxtui
```

Install dependencies

```bash
$  go mod tidy
```

Run the TUI

```bash
$  go run ./cmd --server http://localhost:9003
```

### Flags

| Flag     | Description                           | Default                     |
| -------- | ------------------------------------- | --------------------------- |
| --server | RMX server URL                        | https://api.rapidmidiex.com |
| --debug  | Debug Mode. Logs write to `debug.log` | false                       |

#### Example

```
$  go run ./cmd --debug --server http://localhost:9003
```

View logs in another terminal:

```
$  tail -f debug.log

debug 2023/01/21 06:06:34 LISTEN
```
