package cmdutils

import "path/filepath"

func GeneratedCorpusDir(projectDir, fuzzTest string) string {
	// Store the generated corpus in a single persistent directory per
	// fuzz test in a hidden subdirectory.
	return filepath.Join(projectDir, ".cifuzz-corpus", fuzzTest)
}
