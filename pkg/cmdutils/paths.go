package cmdutils

import "path/filepath"

func DefaultSeedCorpusDir(fuzzTest string) string {
	return fuzzTest + "_seed_corpus"
}

func GeneratedCorpusDir(projectDir, fuzzTest string) string {
	// Store the generated corpus in a single persistent directory per
	// fuzz test in a hidden subdirectory.
	return filepath.Join(projectDir, ".cifuzz-corpus", fuzzTest)
}
