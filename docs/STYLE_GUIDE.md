# Go Project Style Guide

This style guide outlines the best practices to ensure consistency and readability across the codebase.

## General Guidelines

- **File Naming**: Use lowercase letters and underscores for file names (e.g., `my_file.go`).
- **Package Names**: Use short, concise, and lowercase names for packages. Avoid underscores and mixed caps.
- **Error Handling**: Always check for errors and handle them appropriately. Return errors to the caller when necessary.
- **Testing**: Write tests for your code. Use descriptive names for test functions and include edge cases.
- **Line Length**: Limit lines to 99 characters when reasonable to improve readability.

## Licensing and File Headers

To maintain consistency and compliance with the project’s licensing policy, all source files must include an SPDX license identifier.

```go
// SPDX-License-Identifier: Apache-2.0
```

## Code Formatting

- **Imports**: Use [`goimports`](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) to format and organize imports. Separate standard library, third-party, and project-specific imports with blank lines. Also ensure only necessary imports.
- **Indentation and Spacing**: Use the [`go fmt`](https://go.dev/blog/gofmt) tool to automatically format your code. This ensures consistent indentation with tabs and proper spacing after commas, colons, and semicolons.
- **Braces**: Place opening braces on the same line as the statement (e.g., `if`, `for`, `func`).

## Additional Guidelines

- **Empty Line at End of File**: Ensure that all files include an empty line at the end. This helps with version control diffs and adheres to POSIX standards.
- Other [Go checks](https://github.com/complytime/complytime/blob/main/.golangci.yml) are present in CI/CD and therefore it may be useful to also run them locally before submitting a PR.
- The pre-commit and pre-push hooks can be configured by installing [pre-commit](https://pre-commit.com/) and running `make dev-setup`
- ComplyTime leverages the [charmbracelet/log](https://github.com/charmbracelet/log) library for logging all command and plugin activity. By default, this output is printed to stdout.
