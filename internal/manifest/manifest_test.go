package manifest

import "testing"

func TestMatch(t *testing.T) {
	cases := []struct {
		pats []string
		rel  string
		want bool
	}{
		{[]string{"*.bak*"}, "config.kdl.bak2", true},
		{[]string{".claude/"}, ".claude/settings.json", true},
		{[]string{".git/"}, "plugins/x/.git/HEAD", true},
		{[]string{"plugins/zsh-*/"}, "plugins/zsh-autosuggestions/a/b.zsh", true},
		{[]string{"plugins/zsh-*/"}, "plugins/example/x", false},
		{[]string{"lua/plugins/*.lua"}, "lua/plugins/lsp.lua", true},
		{[]string{"lua/plugins/*.lua"}, "lua/config/lsp.lua", false},
		{[]string{"databases.zsh"}, "databases.zsh", true},
	}
	for _, c := range cases {
		if got := Match(c.pats, c.rel); got != c.want {
			t.Errorf("Match(%v, %q) = %v, want %v", c.pats, c.rel, got, c.want)
		}
	}
}
