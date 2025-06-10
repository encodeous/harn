# Simple Testing Harness

Install:

`go install github.com/encodeous/harn@latest`

This program is a simple testing harness that allows users to compare program output using stdin/out.

```
Usage: harn [options] <program_to_execute> <glob_pattern>
Example: harn -v -t 5s ./myprogram 'testcases/*.in'
  -v               Enable full output when tests fail
  -t               Set timeout for program execution (default: 30s)
  -g               Generate output files if they don't exist
  -f               (when -g is passed in) Overwrite the output file even if it exists
  -h               Use SHA256 to compare with .hash files instead of .out files
```