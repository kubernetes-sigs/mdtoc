# Markdown Table of Contents Generator

`mdtoc` is a utility for generating a table-of-contents for markdown files.

Only github-flavored markdown is currently supported, but I am open to accepting patches to add
other formats.

# Table of Contents

Generated with `mdtoc --inplace --skip-prefix README.md`

<!-- toc -->
- [Usage](#usage)
- [Installation](#installation)
<!-- /toc -->

## Usage

```
Usage: mdtoc [OPTIONS] [FILE]...
Generate a table of contents for a markdown file (github flavor).
TOC is wrapped in a pair of tags to allow in-place updates: <!-- toc --><!-- /toc -->
  -dryrun
    	Whether to check for changes to TOC, rather than overwriting. Requires --inplace flag.
  -inplace
    	Whether to edit the file in-place, or output to STDOUT. Requires toc tags to be present.
  -skip-prefix
    	Whether to ignore any headers before the opening toc tag.
```

## Installation

```
go get github.com/tallclair/mdtoc
```
