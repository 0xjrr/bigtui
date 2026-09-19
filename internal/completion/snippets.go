package completion

type SnippetMapping struct {
	Prefix      string
	Label       string
	InsertText  string
	Description string
}

var sqlSnippetMappings = []SnippetMapping{
	{Prefix: "SF", Label: "SELECT * FROM `", InsertText: "SELECT * FROM `", Description: "query snippet"},
}
