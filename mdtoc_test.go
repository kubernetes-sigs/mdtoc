package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testdata = "testdata"

type testcase struct {
	file          string
	includePrefix bool
	completeTOC   bool
	validTOCTags  bool
	expectedTOC   string
}

var testcases = []testcase{{
	file:          "weird_headings.md",
	includePrefix: false,
	completeTOC:   true,
	validTOCTags:  true,
}, {
	file:          "empty_toc.md",
	includePrefix: false,
	completeTOC:   false,
	validTOCTags:  true,
	expectedTOC:   "- [Only Heading](#only-heading)\n",
}, {
	file:          "include_prefix.md",
	includePrefix: true,
	completeTOC:   true,
	validTOCTags:  true,
}, {
	file:         "invalid_nostart.md",
	validTOCTags: false,
}, {
	file:         "invalid_noend.md",
	validTOCTags: false,
}, {
	file:         "invalid_endbeforestart.md",
	validTOCTags: false,
}}

func TestDryRun(t *testing.T) {
	for _, test := range testcases {
		t.Run(test.file, func(t *testing.T) {
			testFile := filepath.Join("testdata", test.file)
			opts := options{
				dryrun:     true,
				inplace:    true,
				skipPrefix: !test.includePrefix,
			}
			assert.NoError(t, validateArgs(opts, []string{testFile}), test.file)

			_, err := run(testFile, opts)

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
			testFile := filepath.Join("testdata", test.file)
			original, err := ioutil.ReadFile(testFile)
			require.NoError(t, err, test.file)

			// Create a copy of the testFile to modify.
			tmpFile, err := ioutil.TempFile("", test.file)
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
				assert.Equal(t, original, updated, test.file)
			} else {
				assert.NotEqual(t, original, updated, test.file)
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
			testFile := filepath.Join("testdata", test.file)
			opts := options{
				dryrun:     false,
				inplace:    false,
				skipPrefix: !test.includePrefix,
			}
			assert.NoError(t, validateArgs(opts, []string{testFile}), test.file)

			toc, err := run(testFile, opts)
			assert.NoError(t, err, test.file)

			if test.expectedTOC != "" {
				assert.Equal(t, test.expectedTOC, toc, test.file)
			}
		})
	}
}
