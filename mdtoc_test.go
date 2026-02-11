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

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"sigs.k8s.io/mdtoc/pkg/mdtoc"
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
	file:          testdata("capital_toc.md"),
	includePrefix: false,
	completeTOC:   true,
	validTOCTags:  true,
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
}, {
	file:          testdata("code.md"),
	includePrefix: true,
	completeTOC:   true,
	validTOCTags:  true,
}}

func testdata(subpath string) string {
	return filepath.Join("testdata", subpath)
}

func TestDryRun(t *testing.T) {
	t.Parallel()

	for _, test := range testcases {
		t.Run(test.file, func(t *testing.T) {
			t.Parallel()

			opts := utilityOptions{
				Options: mdtoc.Options{
					Dryrun:     true,
					SkipPrefix: !test.includePrefix,
					MaxDepth:   mdtoc.MaxHeaderDepth,
				},
				Inplace: true,
			}
			require.NoError(t, validateArgs(opts, []string{test.file}), test.file)

			err := mdtoc.WriteTOC(test.file, opts.Options)

			if test.completeTOC {
				assert.NoError(t, err, test.file)
			} else {
				assert.Error(t, err, test.file)
			}
		})
	}
}

func TestInplace(t *testing.T) {
	t.Parallel()

	for _, test := range testcases {
		t.Run(test.file, func(t *testing.T) {
			t.Parallel()

			original, err := os.ReadFile(test.file)
			require.NoError(t, err, test.file)

			// Create a copy of the test.file to modify.
			escapedFile := strings.ReplaceAll(test.file, string(filepath.Separator), "_")
			tmpFile, err := os.CreateTemp(t.TempDir(), escapedFile)
			require.NoError(t, err, test.file)

			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write(original)
			require.NoError(t, err, test.file)
			require.NoError(t, tmpFile.Close(), test.file)

			opts := utilityOptions{
				Options: mdtoc.Options{
					SkipPrefix: !test.includePrefix,
					Dryrun:     false,
					MaxDepth:   mdtoc.MaxHeaderDepth,
				},
			}
			assert.NoError(t, validateArgs(opts, []string{tmpFile.Name()}), test.file)

			err = mdtoc.WriteTOC(tmpFile.Name(), opts.Options)
			if test.validTOCTags {
				require.NoError(t, err, test.file)
			} else {
				require.Error(t, err, test.file)
			}

			updated, err := os.ReadFile(tmpFile.Name())
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
	t.Parallel()

	for _, test := range testcases {
		// Ignore the invalid cases, they're only for inplace tests.
		if !test.validTOCTags {
			continue
		}

		t.Run(test.file, func(t *testing.T) {
			t.Parallel()

			opts := utilityOptions{
				Options: mdtoc.Options{
					Dryrun:     false,
					SkipPrefix: !test.includePrefix,
					MaxDepth:   mdtoc.MaxHeaderDepth,
				},
			}
			require.NoError(t, validateArgs(opts, []string{test.file}), test.file)

			toc, err := mdtoc.GetTOC(test.file, opts.Options)
			require.NoError(t, err, test.file)

			if test.expectedTOC != "" {
				assert.Equal(t, test.expectedTOC, toc, test.file)
			}
		})
	}
}
