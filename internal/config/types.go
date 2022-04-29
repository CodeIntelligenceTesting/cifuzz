package config

type TargetType string

const (
	CPP TargetType = "cpp"
	//JAVA string = "java"
	//GO   string = "go"
)

// map of supported types -> label:value
var SupportedTypes = map[string]string{
	"C/C++": string(CPP),
	//"Java":  JAVA,
	//"Go":    GO,
}
