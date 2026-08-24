package redact

import (
	"strings"
	"testing"
)

func TestTemplate(t *testing.T) {
	in := `# Database connection strings.

export NVIM_DB_DEV="postgres://u:p@h/db"
export H=mqtt.example.com # public
PW='hunter2'
typeset -g FOO=bar
db() { echo "x=1"; }
`
	got := string(Template([]byte(in)))
	for _, want := range []string{
		"# Database connection strings.\n",
		"export NVIM_DB_DEV=\n",
		"export H=mqtt.example.com # public\n",
		"PW=\n",
		"typeset -g FOO=\n",
		"db() { echo \"x=1\"; }\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, leak := range []string{"hunter2", "postgres://", "bar"} {
		if strings.Contains(got, leak) {
			t.Errorf("leaked %q", leak)
		}
	}
}
