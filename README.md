# Simple Testing Harness

Install:

`go install github.com/encodeous/harn`

This program is a simple testing harness that allows users to compare program output using stdin/out.

```
Usage: harn [options] <program_to_execute> <glob_pattern>
Example: harn -v -t 5s ./myprogram 'testcases/*.in'
  -v               Enable full output when tests fail
  -t               Set timeout for program execution (default: 30s)
  -g               Generate output files if they don't exist
  -h               Use SHA256 to compare with .hash files instead of .out files
```