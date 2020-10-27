package rnzml

import (
	"fmt"
	"strings"
	"testing"
)

var r = NewRenderer()

func ExampleRenderer_Render() {
	r := NewRenderer()
	out := &strings.Builder{}
	_ = r.Render(strings.NewReader("*Hello*, `world`"), out)
	fmt.Print(out.String())
	// Output:
	// <p><strong>Hello</strong>, <code>world</code>
	// </p>
	//
}

func TestRender(t *testing.T) {
	t.Run("Should render rnzml to HTML", func(t *testing.T) {
		in := strings.Join([]string{
			"Here is the first line",
			"```",
			"p := *b",
			"",
			"`var`",
			"```",
			"Here `code` is *bold*",
		}, "\n")
		expected := strings.Join([]string{
			"<p>Here is the first line",
			"</p>",
			"<pre><code>p := *b",
			"",
			"`var`",
			"</code></pre>",
			"<p>Here <code>code</code> is <strong>bold</strong>",
			"</p>\n",
		}, "\n")
		out := &strings.Builder{}
		err := r.Render(strings.NewReader(in), out)
		if err != nil {
			t.Error(err)
		} else if expected != out.String() {
			t.Errorf("expected: '%s'(%x) got: '%s'(%x)", expected, []byte(expected), out.String(), []byte(out.String()))
		}
	})
	t.Run("Should encase text blocks in paragraph tags", func(t *testing.T) {
		out := &strings.Builder{}
		expected := "<p>a\n</p>\n"
		err := r.Render(strings.NewReader("a"), out)
		if err != nil {
			t.Error(err)
		} else if expected != out.String() {
			t.Errorf("expected: '%s'(%x) got: '%s'(%x)", expected, []byte(expected), out.String(), []byte(out.String()))
		}
	})
	t.Run("Should encase code blocks in <pre><code>tags", func(t *testing.T) {
		out := &strings.Builder{}
		expected := "<pre><code>a\n</code></pre>\n"
		err := r.Render(strings.NewReader("```\na\n```"), out)
		if err != nil {
			t.Error(err)
		} else if expected != out.String() {
			t.Errorf("expected: '%s'(%x) got: '%s'(%x)", expected, []byte(expected), out.String(), []byte(out.String()))
		}
	})
	t.Run("Should check for closing code blocks", func(t *testing.T) {
		out := &strings.Builder{}
		err := r.Render(strings.NewReader("```"), out)
		if err == nil {
			t.Error("expected error")
		}
	})
	t.Run("Should pass line and char error information in error", func(t *testing.T) {
		out := &strings.Builder{}
		expected := "line 2: unclosed bold text (*) at position: 1"
		err := r.Render(strings.NewReader("a\nb*\nc"), out)
		if err == nil {
			t.Error("expected error")
		}
		if expected != err.Error() {
			t.Errorf("expected: '%s' got: '%s'", expected, err.Error())
		}
	})
}

func TestRenderLine(t *testing.T) {
	t.Run("Should make text encased in '*' bold", func(t *testing.T) {
		out := &strings.Builder{}
		expected := "a <strong>bold</strong> word"
		err := r.renderLine("a *bold* word", out)
		if err != nil {
			t.Error(err)
		} else if expected != out.String() {
			t.Errorf("expected: '%s' got: %s", expected, out.String())
		}
	})
	t.Run("Should check for unclosed '*'", func(t *testing.T) {
		out := &strings.Builder{}
		err := r.renderLine("a *unclosed bold", out)
		if err == nil {
			t.Errorf("expected error")
		}
	})
	t.Run("Should make text incased in '`' code", func(t *testing.T) {
		out := &strings.Builder{}
		expected := "a <code>programmer</code> word"
		err := r.renderLine("a `programmer` word", out)
		if err != nil {
			t.Error(err)
		} else if expected != out.String() {
			t.Errorf("expected: '%s' got: %s", expected, out.String())
		}
	})
	t.Run("Should check for unclosed '`'", func(t *testing.T) {
		out := &strings.Builder{}
		err := r.renderLine("a `unclosed programmer", out)
		if err == nil {
			t.Errorf("expected error")
		}
	})
	t.Run("Should escape basic HTML control characters", func(t *testing.T) {
		out := &strings.Builder{}
		expected := "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;"
		err := r.renderLine("<script>alert('xss')</script>", out)
		if err != nil {
			t.Error(err)
		} else if expected != out.String() {
			t.Errorf("expected: '%s' got: %s", expected, out.String())
		}
	})
}

var escapetests = []struct {
	in  string
	out string
}{
	{`\`, ""},
	{`\\`, `\`},
	{`\\\`, `\`},
	{`\\\\`, `\\`},
	{`\*`, `*`},
	{`\\**`, `\<strong></strong>`},
}

func TestEscapes(t *testing.T) {
	for _, tt := range escapetests {
		t.Run(tt.in, func(t *testing.T) {
			out := &strings.Builder{}
			err := r.renderLine(tt.in, out)
			if err != nil {
				t.Errorf("error: %s", err.Error())
			} else if tt.out != out.String() {
				t.Errorf("expected: '%s' got: '%s'", tt.out, out.String())
			}
		})
	}
}

var linktests = []struct {
	in  string
	out string
	err bool
}{
	{`[1 2]`, `<a href="1">2</a>`, false},
	{`[1 2 3]`, `<a href="1">2 3</a>`, false},
	{`[1*2* 3]`, `<a href="1*2*">3</a>`, false},
	{`[1 *2* 3]`, `<a href="1">*2* 3</a>`, false},
	{`[1 *2*3]`, `<a href="1">*2*3</a>`, false},
	{`[1 \*2*3]`, `<a href="1">*2*3</a>`, false},
	{`[1 \\*2*3]`, `<a href="1">\*2*3</a>`, false},
	{`[1 \]]`, `<a href="1">]</a>`, false},
	{`[1 <]`, `<a href="1">&lt;</a>`, false},
	{`[<a 2]`, `<a href="%3ca">2</a>`, false},
	{`[1 2`, "", true},
	{`[1]`, "", true},
	{`[]`, "", true},
}

func TestLinks(t *testing.T) {
	for _, tt := range linktests {
		t.Run(tt.in, func(t *testing.T) {
			out := &strings.Builder{}
			err := r.renderLine(tt.in, out)
			if tt.err {
				if err == nil {
					t.Errorf("expected error")
				}
			} else if err != nil {
				t.Errorf("error: %s", err.Error())
			} else if tt.out != out.String() {
				t.Errorf("expected: '%s' got: '%s'", tt.out, out.String())
			}
		})
	}
}
