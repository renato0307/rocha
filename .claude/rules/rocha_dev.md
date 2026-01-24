# Things to do/know during rocha dev

## Running with Debug Flags

To run rocha with debug output:

```bash
rocha --debug --debug-file <file name>
```

To use the `--dev` flag, you must use the `run` command:

```bash
rocha run --dev
```

## Dev guidelines

- When you add a shortcut, always add to the help screen too.

## When you finish:

1. You need to build the binary for testing with build flags that inject meaningful version information:
   - Use `-ldflags` to set version variables based on the branch name
   - Version should include the branch name for easy identification (e.g., "fix-show-header-on-dialogs-v1")
   - Set commit hash using `git rev-parse HEAD`
   - Set build date using current timestamp
   - Set go version using `go version`
   - Example: `-ldflags="-X rocha/version.Version=branch-name-v1 -X rocha/version.Commit=$(git rev-parse HEAD) -X rocha/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X rocha/version.GoVersion=$(go version | awk '{print $3}')"`
   - Build output goes to `./bin/` directory (ignored by git)

2. When running the built binary for testing, ALWAYS use the --dev flag to see version info in dialog headers and verify you're testing the correct binary.

3. Check if the ARCHITECTURE.md needs update, specially after adding new packages or components; don't forget, you should add or modify mostly diagrams; the amount of text in this file MUST be kept to a minimum.
