package config

type FuzzTestType string

const (
	CPP FuzzTestType = "cpp"
)

type Engine string

const (
	LIBFUZZER Engine = "libfuzzer"
)
