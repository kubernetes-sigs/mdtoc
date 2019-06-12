package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testcase struct {
	file          string
	includePrefix bool
	completeTOC   bool
	validTOCTags  bool
	expectedTOC   string
}

var testcases = []testcase{{
	file:          testdata("weird_headings.md"),
	includePrefix: false,
	completeTOC:   true,
	validTOCTags:  true,
}, {
	file:          testdata("empty_toc.md"),
	includePrefix: false,
	completeTOC:   false,
	validTOCTags:  true,
	expectedTOC:   "- [Only Heading](#only-heading)\n",
}, {
	file:          testdata("include_prefix.md"),
	includePrefix: true,
	completeTOC:   true,
	validTOCTags:  true,
}, {
	file:         testdata("invalid_nostart.md"),
	validTOCTags: false,
}, {
	file:         testdata("invalid_noend.md"),
	validTOCTags: false,
}, {
	file:         testdata("invalid_endbeforestart.md"),
	validTOCTags: false,
}, {
	file:          "README.md",
	includePrefix: false,
	completeTOC:   true,
	validTOCTags:  true,
}}

func testdata(subpath string) string {
	return filepath.Join("testdata", subpath)
}

func TestDryRun(t *testing.T) {
	for _, test := range testcases {
		t.Run(test.file, func(t *testing.T) {
			opts := options{
				dryrun:     true,
				inplace:    true,
				skipPrefix: !test.includePrefix,
			}
			assert.NoError(t, validateArgs(opts, []string{test.file}), test.file)

			_, err := run(test.file, opts)

			if test.completeTOC {
				assert.NoError(t, err, test.file)
			} else {
				assert.Error(t, err, test.file)
			}
		})
	}
}

func TestInplace(t *testing.T) {
	for _, test := range testcases {
		t.Run(test.file, func(t *testing.T) {
			original, err := ioutil.ReadFile(test.file)
			require.NoError(t, err, test.file)

			// Create a copy of the test.file to modify.
			escapedFile := strings.ReplaceAll(test.file, string(filepath.Separator), "_")
			tmpFile, err := ioutil.TempFile("", escapedFile)
			require.NoError(t, err, test.file)
			defer os.Remove(tmpFile.Name())
			_, err = tmpFile.Write(original)
			require.NoError(t, err, test.file)
			require.NoError(t, tmpFile.Close(), test.file)

			opts := options{
				inplace:    true,
				skipPrefix: !test.includePrefix,
			}
			assert.NoError(t, validateArgs(opts, []string{tmpFile.Name()}), test.file)

			_, err = run(tmpFile.Name(), opts)
			if test.validTOCTags {
				require.NoError(t, err, test.file)
			} else {
				assert.Error(t, err, test.file)
			}

			updated, err := ioutil.ReadFile(tmpFile.Name())
			require.NoError(t, err, test.file)

			if test.completeTOC || !test.validTOCTags { // Invalid tags should not modify contents.
				assert.Equal(t, string(original), string(updated), test.file)
			} else {
				assert.NotEqual(t, string(original), string(updated), test.file)
			}
		})
	}
}

func TestOutput(t *testing.T) {
	for _, test := range testcases {
		// Ignore the invalid cases, they're only for inplace tests.
		if !test.validTOCTags {
			continue
		}

		t.Run(test.file, func(t *testing.T) {
			opts := options{
				dryrun:     false,
				inplace:    false,
				skipPrefix: !test.includePrefix,
			}
			assert.NoError(t, validateArgs(opts, []string{test.file}), test.file)

			toc, err := run(test.file, opts)
			assert.NoError(t, err, test.file)

			if test.expectedTOC != "" {
				assert.Equal(t, test.expectedTOC, toc, test.file)
			}
		})
	}
}
