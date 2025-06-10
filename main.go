package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var Reset = "\033[0m"
var Red = "\033[31m"
var Green = "\033[32m"
var Yellow = "\033[33m"
var Blue = "\033[34m"
var Magenta = "\033[35m"
var Cyan = "\033[36m"
var Gray = "\033[37m"
var White = "\033[97m"

func main() {
	// Define command line flags
	verbose := flag.Bool("v", false, "Enable full verbose output when tests fail")
	silent := flag.Bool("s", false, "Enable silent output when tests fail")
	timeout := flag.Duration("t", 30*time.Second, "Timeout for program execution (e.g., 5s, 1m, 500ms)")
	generate := flag.Bool("g", false, "Generate output files if they don't exist")
	forceGen := flag.Bool("f", false, "Overwrite the output file even if it exists")
	useHash := flag.Bool("h", false, "Use SHA256 hash comparison with .hash files instead of .out files")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		fmt.Println("Usage: harn [options] <program_to_execute> <glob_pattern>")
		fmt.Println("Example: harn -v -t 5s ./myprogram 'testcases/*.in'")
		fmt.Println("  -v               Enable full output when tests fail")
		fmt.Println("  -t               Set timeout for program execution (default: 30s)")
		fmt.Println("  -g               Generate output files if they don't exist")
		fmt.Println("  -f               (when -g is passed in) Overwrite the output file even if it exists")
		fmt.Println("  -h               Use SHA256 to compare with .hash files instead of .out files")
		os.Exit(1)
	}

	programPath := args[0]
	globPattern := args[1]

	expectedExt := ".out"
	if *useHash {
		expectedExt = ".hash"
	}

	// Find all .in files matching the glob pattern
	inputFiles, err := filepath.Glob(globPattern)
	if err != nil {
		log.Fatalf("Error matching glob pattern: %v", err)
	}

	if len(inputFiles) == 0 {
		fmt.Printf("No files found matching pattern: %s\n", globPattern)
		return
	}

	fmt.Printf("Found %d input files matching pattern \"%s\" (timeout: %v)\n", len(inputFiles), globPattern, *timeout)

	passedTests := 0
	totalTests := len(inputFiles)
	generatedFiles := 0
	var totalExecutionTime time.Duration

	for _, inputFile := range inputFiles {
		fmt.Printf("%s%s%s - ", Yellow, inputFile, Reset)

		// Generate corresponding .out/.hash file name
		outputFile := strings.TrimSuffix(inputFile, ".in") + expectedExt

		// Check if the expected output file exists
		if *generate {
			if _, err := os.Stat(outputFile); os.IsNotExist(err) || *forceGen {
				actualOutput, executionTime, err := executeProgram(programPath, inputFile, *timeout, *useHash)
				totalExecutionTime += executionTime
				execTimeStr := executionTime.Round(time.Millisecond).String()

				if err != nil {
					if err == context.DeadlineExceeded {
						fmt.Printf("%sTLE%s [%s]: Program exceeded %v timeout\n", Gray, Reset, execTimeStr, *timeout)
					} else {
						fmt.Printf("%sERR%s [%s]: executing program: %v\n", Red, Reset, execTimeStr, err)
					}
					continue
				}
				err = writeFile(outputFile, actualOutput)
				if err != nil {
					fmt.Printf("%sERR%s [%s]: failed while writing output: %v\n", Red, Reset, execTimeStr, err)
				} else {
					fmt.Printf("%sGEN%s [%s]: Wrote output file %s\n", Green, Reset, execTimeStr, outputFile)
					generatedFiles++
				}
			} else {
				fmt.Printf("%sSKIP%s: Output file %s found, skipping\n", Gray, Reset, outputFile)
				passedTests++
			}
		} else {
			actualOutput, executionTime, err := executeProgram(programPath, inputFile, *timeout, *useHash)
			totalExecutionTime += executionTime
			execTimeStr := executionTime.Round(time.Millisecond).String()

			if err != nil {
				if err == context.DeadlineExceeded {
					fmt.Printf("%sTLE%s [%s]: Program exceeded %v timeout\n", Gray, Reset, execTimeStr, *timeout)
				} else {
					fmt.Printf("%sERR%s [%s]: executing program: %v\n", Red, Reset, execTimeStr, err)
				}
				continue
			}

			// Read expected output
			expectedOutput, err := readFile(outputFile)
			if err != nil {
				fmt.Printf("%sERR%s: reading expected output file: %v\n", Red, Reset, err)
				continue
			}

			// Compare outputs
			if strings.TrimSpace(actualOutput) == strings.TrimSpace(expectedOutput) {
				fmt.Printf("%sAC%s [%s]: Output matches expected result\n", Green, Reset, execTimeStr)
				passedTests++
				if *verbose {
					fmt.Printf(" === Expected:\n%s\n", expectedOutput)
					fmt.Printf(" === End Expected:\n")
					fmt.Printf(" === Actual:\n%s\n", actualOutput)
					fmt.Printf(" === End Actual:\n")
				}
			} else {
				fmt.Printf("%sWA%s [%s]: Output doesn't match\n", Red, Reset, execTimeStr)
				if *verbose {
					fmt.Printf(" === Expected:\n%s\n", expectedOutput)
					fmt.Printf(" === End Expected:\n")
					fmt.Printf(" === Actual:\n%s\n", actualOutput)
					fmt.Printf(" === End Actual:\n")
				} else if !*silent {
					dmp := diffmatchpatch.New()

					diffs := dmp.DiffMain(expectedOutput, actualOutput, false)

					fmt.Printf(" === Diff:\n")
					fmt.Println(dmp.DiffPrettyText(diffs))
					fmt.Printf(" === End Diff (💡 Use -v flag for full output)\n")
				}
			}
		}
	}

	// Print summary
	fmt.Printf("\n" + strings.Repeat("=", 50) + "\n")
	if *generate {
		fmt.Printf("Generated %d/%d new test files\n", generatedFiles, totalTests)
		fmt.Printf("    - %d/%d tests already exist\n", passedTests, totalTests)
	} else {
		fmt.Printf("Test Results: %d/%d passed\n", passedTests, totalTests)
		fmt.Printf("Total execution time: %v\n", totalExecutionTime)
		if totalTests > 0 {
			fmt.Printf("Average execution time: %v\n", totalExecutionTime/time.Duration(totalTests))
		}

		if passedTests == totalTests {
			fmt.Printf("🎉 All tests passed!\n")
		} else {
			fmt.Printf("💥 %d test(s) failed\n", totalTests-passedTests)
		}
	}
}

func executeProgram(programPath, inputFile string, timeout time.Duration, hash bool) (string, time.Duration, error) {
	// Read input file content
	inputContent, err := readFile(inputFile)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read input file: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, programPath)
	cmd.Stdin = strings.NewReader(inputContent)

	start := time.Now()

	var output []byte
	if hash {
		hasher := sha256.New()
		var pipe io.ReadCloser
		pipe, err = cmd.StdoutPipe()
		if err != nil {
			goto errHandle
		}
		err = cmd.Start()
		if err != nil {
			goto errHandle
		}

		hashReader := io.TeeReader(pipe, hasher)

		if _, err = io.Copy(io.Discard, hashReader); err == nil {
			err = cmd.Wait()
			output = []byte(hex.EncodeToString(hasher.Sum(nil)))
		}
	} else {
		output, err = cmd.Output()
	}

errHandle:
	executionTime := time.Since(start)
	if err != nil {
		// Check if it was a timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", executionTime, context.DeadlineExceeded
		}
		return "", executionTime, fmt.Errorf("program execution failed: %v", err)
	}

	return string(output), executionTime, nil
}

// writeFile writes content to a file
func writeFile(filename, content string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}

// readFile reads the entire content of a file and returns it as a string
func readFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var content strings.Builder
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteString("\n")
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.TrimSuffix(content.String(), "\n"), nil
}
