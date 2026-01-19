# pretty-claude-stream

A CLI tool that pretty-prints Claude AI API streaming responses with ANSI color formatting.

## Installation

```bash
git clone <repo-url>
cd pretty-claude-stream
make build
```

Or install directly once published:

```bash
go install github.com/<username>/pretty-claude-stream@latest
```

## Usage

Pipe JSON streaming output from the Claude API to pretty-print it:

```bash
your-claude-client | pretty-claude-stream
```

The tool reads newline-delimited JSON events from stdin and displays:
- Assistant text responses with proper formatting
- Tool calls with highlighted names and pretty-printed parameters
- Error messages in red

## Example

Input (JSON stream):
```json
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":"Hello"}}}
{"type":"stream_event","event":{"type":"content_block_delta","delta":{"type":"text_delta","text":" world!"}}}
```

Output:
```
Hello world!
```

Tool calls are displayed with syntax highlighting:
```
[Tool: read_file]
  path: /home/user/example.txt
```

## License

MIT
