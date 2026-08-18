/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mdtoc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gomarkdown/markdown/ast"
	"github.com/gomarkdown/markdown/html"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripFrontMatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no front matter",
			input:    "# Hello\nWorld\n",
			expected: "# Hello\nWorld\n",
		},
		{
			name:     "valid YAML front matter",
			input:    "---\ntitle: test\n---\n# Hello\n",
			expected: "# Hello\n",
		},
		{
			name:     "front matter with CRLF",
			input:    "---\r\ntitle: test\r\n---\r\n# Hello\r\n",
			expected: "# Hello\r\n",
		},
		{
			name:     "unclosed front matter",
			input:    "---\ntitle: test\n# Hello\n",
			expected: "---\ntitle: test\n# Hello\n",
		},
		{
			name:     "does not start at byte 0",
			input:    " ---\ntitle: test\n---\n# Hello\n",
			expected: " ---\ntitle: test\n---\n# Hello\n",
		},
		{
			name:     "empty content after front matter",
			input:    "---\ntitle: test\n---\n",
			expected: "",
		},
		{
			name:     "front matter with multiple fields",
			input:    "---\ntitle: test\ndesc: foo\ntags:\n- a\n- b\n---\n# Content\n",
			expected: "# Content\n",
		},
		{
			name:     "triple dash with trailing text is not a closing delimiter",
			input:    "---\ntitle: test\n---not-on-own-line\n# Real content\n",
			expected: "---\ntitle: test\n---not-on-own-line\n# Real content\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := stripFrontMatter([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestAnchorGen(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "basic heading",
			input:    "Hello World",
			expected: "hello-world",
		},
		{
			name:     "punctuation removal",
			input:    "What's up?",
			expected: "whats-up",
		},
		{
			name:     "mixed case",
			input:    "CamelCase UPPER lower",
			expected: "camelcase-upper-lower",
		},
		{
			name:     "special characters",
			input:    "!@#$&*^ headers!",
			expected: "-headers",
		},
		{
			name:     "preserves hyphens",
			input:    "-and-then-we-said_go",
			expected: "-and-then-we-said_go",
		},
		{
			name:     "all dashes",
			input:    "-------",
			expected: "-------",
		},
		{
			name:     "complex punctuation",
			input:    "bring it all out: !@#$%^&*(){}/=?+;:\"'`,.<> ok",
			expected: "bring-it-all-out--ok",
		},
		{
			name:     "unicode characters are stripped",
			input:    "Über cool",
			expected: "ber-cool",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			a := make(anchorGen)
			result := a.mkAnchor(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAnchorGenDuplicates(t *testing.T) {
	t.Parallel()

	a := make(anchorGen)

	assert.Equal(t, "duplicate", a.mkAnchor("Duplicate"))
	assert.Equal(t, "duplicate-1", a.mkAnchor("Duplicate"))
	assert.Equal(t, "duplicate-2", a.mkAnchor("Duplicate"))
	assert.Equal(t, "duplicate-3", a.mkAnchor("duplicate"))
}

func TestAsText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "plain text heading",
			markdown: "# Hello World\n",
			expected: "Hello World",
		},
		{
			name:     "heading with inline code",
			markdown: "# Hello `code` World\n",
			expected: "Hello code World",
		},
		{
			name:     "heading with bold",
			markdown: "# **bold** text\n",
			expected: "bold text",
		},
		{
			name:     "heading with italic",
			markdown: "# _italic_ text\n",
			expected: "italic text",
		},
		{
			name:     "heading with bold and italic",
			markdown: "# **be _bold_**\n",
			expected: "be bold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := parse([]byte(tt.markdown))

			var result string

			walkHeadings(doc, func(heading *ast.Heading) {
				result = asText(heading)
			})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindTOCTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		input         string
		expectedStart int
		expectedEnd   int
	}{
		{
			name:          "valid tag pair",
			input:         "# Title\n<!-- toc -->\ntoc content\n<!-- /toc -->\n",
			expectedStart: 8,
			expectedEnd:   33,
		},
		{
			name:          "missing start tag",
			input:         "# Title\n<!-- /toc -->\n",
			expectedStart: -1,
			expectedEnd:   8,
		},
		{
			name:          "missing end tag",
			input:         "# Title\n<!-- toc -->\n",
			expectedStart: 8,
			expectedEnd:   -1,
		},
		{
			name:          "no tags",
			input:         "# Title\nSome content\n",
			expectedStart: -1,
			expectedEnd:   -1,
		},
		{
			name:          "case insensitive",
			input:         "<!-- TOC -->\ncontent\n<!-- /TOC -->\n",
			expectedStart: 0,
			expectedEnd:   21,
		},
		{
			name:          "mixed case",
			input:         "<!-- Toc -->\ncontent\n<!-- /toC -->\n",
			expectedStart: 0,
			expectedEnd:   21,
		},
		{
			name:          "tags in wrong order",
			input:         "<!-- /toc -->\n<!-- toc -->\n",
			expectedStart: 14,
			expectedEnd:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			start, end := findTOCTags([]byte(tt.input))
			assert.Equal(t, tt.expectedStart, start, "start")
			assert.Equal(t, tt.expectedEnd, end, "end")
		})
	}
}

func TestGenerateTOC(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		opts     Options
		expected string
	}{
		{
			name:     "basic headers",
			markdown: "# H1\n## H2\n### H3\n",
			opts:     Options{},
			expected: "- [H1](#h1)\n  - [H2](#h2)\n    - [H3](#h3)\n",
		},
		{
			name:     "max depth filtering",
			markdown: "# H1\n## H2\n### H3\n#### H4\n",
			opts:     Options{MaxDepth: 2},
			expected: "- [H1](#h1)\n  - [H2](#h2)\n",
		},
		{
			name:     "max depth zero includes all",
			markdown: "# H1\n## H2\n### H3\n",
			opts:     Options{MaxDepth: 0},
			expected: "- [H1](#h1)\n  - [H2](#h2)\n    - [H3](#h3)\n",
		},
		{
			name:     "duplicate heading anchors",
			markdown: "## Foo\n## Foo\n## Foo\n",
			opts:     Options{},
			expected: "- [Foo](#foo)\n- [Foo](#foo-1)\n- [Foo](#foo-2)\n",
		},
		{
			name:     "empty document",
			markdown: "",
			opts:     Options{},
			expected: "",
		},
		{
			name:     "no headings",
			markdown: "Just some text\nwith no headings\n",
			opts:     Options{},
			expected: "",
		},
		{
			name:     "heading with inline code",
			markdown: "# Hello `code` World\n",
			opts:     Options{},
			expected: "- [Hello <code>code</code> World](#hello-code-world)\n",
		},
		{
			name:     "headings inside fenced code blocks are ignored",
			markdown: "# Real\n```\n# Not A Heading\n```\n## Also Real\n",
			opts:     Options{},
			expected: "- [Real](#real)\n  - [Also Real](#also-real)\n",
		},
		{
			name:     "heading with punctuation",
			markdown: "## H2: with punctuation!\n",
			opts:     Options{},
			expected: "- [H2: with punctuation!](#h2-with-punctuation)\n",
		},
		{
			name:     "front matter is stripped",
			markdown: "---\ntitle: test\n---\n# Heading\n",
			opts:     Options{},
			expected: "- [Heading](#heading)\n",
		},
		{
			name:     "base level normalization",
			markdown: "## L2\n### L3\n#### L4\n",
			opts:     Options{},
			expected: "- [L2](#l2)\n  - [L3](#l3)\n    - [L4](#l4)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			toc, err := GenerateTOC([]byte(tt.markdown), tt.opts)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, toc)
		})
	}
}

func TestHeadingBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		expected string
	}{
		{
			name:     "plain text",
			markdown: "# Hello\n",
			expected: "Hello",
		},
		{
			name:     "inline code",
			markdown: "# Hello `code`\n",
			expected: "Hello <code>code</code>",
		},
		{
			name:     "bold text",
			markdown: "# **bold** text\n",
			expected: "<strong>bold</strong> text",
		},
		{
			name:     "italic text",
			markdown: "# _italic_ text\n",
			expected: "<em>italic</em> text",
		},
		{
			name:     "bold and italic nested",
			markdown: "# **be _bold_**\n",
			expected: "<strong>be <em>bold</em></strong>",
		},
		{
			name:     "link in heading",
			markdown: "# Check [this](http://example.com)\n",
			expected: "Check <a href=\"http://example.com\">this</a>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := parse([]byte(tt.markdown))
			renderer := html.NewRenderer(html.RendererOptions{})

			var result string

			walkHeadings(doc, func(heading *ast.Heading) {
				result = headingBody(renderer, heading)
			})
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAtomicWrite(t *testing.T) {
	t.Parallel()

	t.Run("writes content atomically", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		require.NoError(t, os.WriteFile(path, []byte("original"), 0o644))

		err := atomicWrite(path, "chunk1", "chunk2", "chunk3")
		require.NoError(t, err)

		content, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "chunk1chunk2chunk3", string(content))
	})

	t.Run("preserves file permissions", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		require.NoError(t, os.WriteFile(path, []byte("original"), 0o755))

		err := atomicWrite(path, "new content")
		require.NoError(t, err)

		fi, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), fi.Mode().Perm())
	})

	t.Run("errors on nonexistent file", func(t *testing.T) {
		t.Parallel()

		err := atomicWrite("/nonexistent/path/file.md", "content")
		assert.Error(t, err)
	})
}

func TestGetTOCFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("with skip prefix and toc tags", func(t *testing.T) {
		t.Parallel()

		doc := []byte("# Title\n<!-- toc -->\n<!-- /toc -->\n## Heading\n")
		toc, err := GetTOCFromBytes(doc, Options{SkipPrefix: true})
		require.NoError(t, err)
		assert.Equal(t, "- [Heading](#heading)\n", toc)
	})

	t.Run("without skip prefix", func(t *testing.T) {
		t.Parallel()

		doc := []byte("# Title\n<!-- toc -->\n<!-- /toc -->\n## Heading\n")
		toc, err := GetTOCFromBytes(doc, Options{SkipPrefix: false})
		require.NoError(t, err)
		assert.Contains(t, toc, "- [Title](#title)")
		assert.Contains(t, toc, "- [Heading](#heading)")
	})

	t.Run("without toc tags", func(t *testing.T) {
		t.Parallel()

		doc := []byte("# Title\n## Heading\n")
		toc, err := GetTOCFromBytes(doc, Options{SkipPrefix: true})
		require.NoError(t, err)
		assert.Contains(t, toc, "- [Title](#title)")
		assert.Contains(t, toc, "- [Heading](#heading)")
	})
}

func TestWriteTOC(t *testing.T) {
	t.Parallel()

	t.Run("missing start tag", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		require.NoError(t, os.WriteFile(path, []byte("# Hello\n<!-- /toc -->\n"), 0o644))

		err := WriteTOC(path, Options{})
		assert.ErrorContains(t, err, "missing opening TOC tag")
	})

	t.Run("missing end tag", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		require.NoError(t, os.WriteFile(path, []byte("# Hello\n<!-- toc -->\n"), 0o644))

		err := WriteTOC(path, Options{})
		assert.ErrorContains(t, err, "missing closing TOC tag")
	})

	t.Run("closing before start tag", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		require.NoError(t, os.WriteFile(path, []byte("<!-- /toc -->\n<!-- toc -->\n"), 0o644))

		err := WriteTOC(path, Options{})
		assert.ErrorContains(t, err, "TOC closing tag before start tag")
	})

	t.Run("dryrun returns error when changes needed", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		content := "<!-- toc -->\n<!-- /toc -->\n## Heading\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		err := WriteTOC(path, Options{Dryrun: true})
		assert.ErrorContains(t, err, "changes found")
	})

	t.Run("dryrun no error when up to date", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		content := "<!-- toc -->\n- [Heading](#heading)\n<!-- /toc -->\n## Heading\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		err := WriteTOC(path, Options{Dryrun: true})
		assert.NoError(t, err)
	})

	t.Run("writes TOC in place", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		content := "<!-- toc -->\n<!-- /toc -->\n## Heading\n"
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		err := WriteTOC(path, Options{})
		require.NoError(t, err)

		result, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Contains(t, string(result), "- [Heading](#heading)")
	})

	t.Run("nonexistent file", func(t *testing.T) {
		t.Parallel()

		err := WriteTOC("/nonexistent/file.md", Options{})
		assert.Error(t, err)
	})
}

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("strips front matter before parsing", func(t *testing.T) {
		t.Parallel()

		doc := parse([]byte("---\ntitle: test\n---\n# Heading\n"))

		var found bool

		walkHeadings(doc, func(heading *ast.Heading) {
			found = true

			assert.Equal(t, "Heading", asText(heading))
		})
		assert.True(t, found)
	})

	t.Run("handles empty input", func(t *testing.T) {
		t.Parallel()

		doc := parse([]byte(""))

		var count int

		walkHeadings(doc, func(_ *ast.Heading) {
			count++
		})
		assert.Equal(t, 0, count)
	})
}

func TestHeadingBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		expected int
	}{
		{
			name:     "starts at h1",
			markdown: "# H1\n## H2\n",
			expected: 1,
		},
		{
			name:     "starts at h2",
			markdown: "## H2\n### H3\n",
			expected: 2,
		},
		{
			name:     "starts at h3",
			markdown: "### H3\n#### H4\n",
			expected: 3,
		},
		{
			name:     "mixed order picks minimum",
			markdown: "### H3\n## H2\n#### H4\n",
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			doc := parse([]byte(tt.markdown))
			assert.Equal(t, tt.expected, headingBase(doc))
		})
	}
}
