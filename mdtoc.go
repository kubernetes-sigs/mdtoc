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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"sigs.k8s.io/release-utils/version"

	"sigs.k8s.io/mdtoc/pkg/mdtoc"
)

var cmd = &cobra.Command{
	Use: os.Args[0] + " [FILE]...",
	Long: "Generate a table of contents for a markdown file (GitHub flavor).\n\n" +
		"TOC may be wrapped in a pair of tags to allow in-place updates:\n" +
		"<!-- toc --><!-- /toc -->\n\n" +
		"Use - as the file argument to read from stdin.",
	RunE: run,
}

type utilityOptions struct {
	mdtoc.Options
	Inplace bool
	Output  string
}

var defaultOptions utilityOptions

func init() {
	cmd.PersistentFlags().BoolVarP(&defaultOptions.Dryrun, "dryrun", "d", false, "Whether to check for changes to TOC, rather than overwriting. Requires --inplace flag.")
	cmd.PersistentFlags().BoolVarP(&defaultOptions.Inplace, "inplace", "i", false, "Whether to edit the file in-place, or output to STDOUT. Requires toc tags to be present.")
	cmd.PersistentFlags().BoolVarP(&defaultOptions.SkipPrefix, "skip-prefix", "s", true, "Whether to ignore any headers before the opening toc tag.")
	cmd.PersistentFlags().IntVarP(&defaultOptions.MaxDepth, "max-depth", "m", mdtoc.MaxHeaderDepth, "Limit the depth of headers that will be included in the TOC.")
	cmd.PersistentFlags().BoolVarP(&defaultOptions.Version, "version", "v", false, "Show MDTOC version.")
	cmd.PersistentFlags().StringVarP(&defaultOptions.Output, "output", "o", "", "Write the TOC to the specified file instead of STDOUT.")
}

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

// expandGlobs expands any glob patterns in the file arguments.
func expandGlobs(args []string) ([]string, error) {
	var expanded []string

	for _, arg := range args {
		if arg == "-" {
			expanded = append(expanded, arg)

			continue
		}

		matches, err := filepath.Glob(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid glob pattern %q: %w", arg, err)
		}

		if len(matches) == 0 {
			// No matches: keep the original argument so the downstream
			// error message references the actual path the user provided.
			expanded = append(expanded, arg)
		} else {
			expanded = append(expanded, matches...)
		}
	}

	return expanded, nil
}

func run(_ *cobra.Command, args []string) error {
	if defaultOptions.Version {
		v := version.GetVersionInfo()
		v.Name = "mdtoc"
		v.Description = "is a utility for generating a table-of-contents for markdown files"
		v.ASCIIName = "true"
		v.FontName = "banner"
		fmt.Fprintln(os.Stdout, v.String())

		return nil
	}

	expandedArgs, err := expandGlobs(args)
	if err != nil {
		return err
	}

	if err := validateArgs(defaultOptions, expandedArgs); err != nil {
		return fmt.Errorf("validate args: %w", err)
	}

	if defaultOptions.Inplace {
		var retErr error

		for _, file := range expandedArgs {
			if err := mdtoc.WriteTOC(file, defaultOptions.Options); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("%s: %w", file, err))
			}
		}

		return retErr
	}

	return generateAndOutput(expandedArgs[0], defaultOptions)
}

func generateAndOutput(file string, opts utilityOptions) error {
	var toc string

	if file == "-" {
		doc, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}

		toc, err = mdtoc.GetTOCFromBytes(doc, opts.Options)
		if err != nil {
			return fmt.Errorf("get toc: %w", err)
		}
	} else {
		var err error

		toc, err = mdtoc.GetTOC(file, opts.Options)
		if err != nil {
			return fmt.Errorf("get toc: %w", err)
		}
	}

	if opts.Output != "" {
		const perms = 0o644

		if err := os.WriteFile(opts.Output, []byte(toc), perms); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		return nil
	}

	fmt.Println(toc)

	return nil
}

func validateArgs(opts utilityOptions, args []string) error {
	if len(args) < 1 {
		return errors.New("must specify at least 1 file")
	}

	if !opts.Inplace && len(args) > 1 {
		return errors.New("non-inplace updates require exactly 1 file")
	}

	if opts.Dryrun && !opts.Inplace {
		return errors.New("--dryrun requires --inplace")
	}

	for _, arg := range args {
		if arg == "-" && opts.Inplace {
			return errors.New("stdin (-) cannot be used with --inplace")
		}
	}

	if opts.Output != "" && opts.Inplace {
		return errors.New("--output cannot be used with --inplace")
	}

	return nil
}
