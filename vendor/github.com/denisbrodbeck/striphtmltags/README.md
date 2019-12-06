# DEPRECATED

This project and its [related](https://github.com/grokify/html-strip-tags-go) [projects](https://gist.github.com/christopherhesse/d422447a086d373a967f) sound like a good idea, but really aren't.

Using the `stripTags` function could be dangerous. From https://golang.org/pkg/html/template/#hdr-Security_Model:

> This package assumes that template authors are trusted

`stripTags` resides within `html/template` and works according to those guaranties. Which might mean, that certain XSS attacks might go through undetected.

A fast, reliable and already battle-worn library to strip HTML tags is [bluemonday](https://github.com/microcosm-cc/bluemonday).

They've got the `bluemonday.StrictPolicy()` mode:

> `bluemonday.StrictPolicy()`is a mode which can be thought of as equivalent to stripping all HTML elements and their attributes as it has nothing on it's whitelist. An example usage scenario would be blog post titles where HTML tags are not expected at all and if they are then the elements and the content of the elements should be stripped. This is a very strict policy.

Example:

```golang
stripped := bluemonday.StrictPolicy().SanitizeBytes(`<a onblur="alert(secret)" href="http://www.google.com">Google</a>`)
// Output: Google
```

That is exactly what you want when stripping arbitrary HTML content. A library, which understands XSS attacks and knows how to defuse these attacks. Even to the point of stripping *all* tags, leaving only plain text.

## Strip HTML tags from strings

[![GoDoc](https://godoc.org/github.com/denisbrodbeck/striphtmltags?status.svg)](https://godoc.org/github.com/denisbrodbeck/striphtmltags) [![Go Report Card](https://goreportcard.com/badge/github.com/denisbrodbeck/striphtmltags)](https://goreportcard.com/report/github.com/denisbrodbeck/striphtmltags)

This Go package strips HTML tags from strings. No heavy lifting is done in this package. The unexported `stripTags` fuction from `html/template/html.go` is better suited for this task. All this package does is providing an exported function to access `stripTags`.

## Background

* The `stripTags` function in [html/template/html.go](https://golang.org/src/html/template/html.go) could be really useful, however, it is not exported.
* Requests to export `stripTags` were made [on Github](https://github.com/golang/go/issues/5884) without success.
* Several attempts exist to *un-unexport* the function ([1](https://github.com/grokify/html-strip-tags-go), [2](https://gist.github.com/christopherhesse/d422447a086d373a967f), [3](https://storage.googleapis.com/go-attachment/5884/7/strip.go)), but all solutions lacked an easy upgrade path for changes made from upstream.
* Most solutions take the content of all `html/template` files and put the content into *one single file*.
* This solution does not modify the original `html/template` source files. Instead, it copies all `html/template` files from go source into this package and adds one `export.go` file, which adds a `StripTags` function (see [Versioning](#versioning) for the whole workflow).

## Use Cases

* Strip HTML from html strings (you don't say :smile:)
* Convert HTML emails to plain text
* Display HTML strings in cli app context
* Convert HTML content into plain text for RSS feeds

## Installation

Import the library with

```golang
import "github.com/denisbrodbeck/striphtmltags"
```

## Usage

```golang
package main

import (
	"fmt"
	"github.com/denisbrodbeck/striphtmltags"
)

func main() {
	html := `<script>...</script> <b>&iexcl;Hi!</b>`
	got := striphtmltags.StripTags(html)
	fmt.Println(got)
	// Output:  &iexcl;Hi!
}
```

## Versioning

This package follows the go release cycle.

On each new go release we:

* download the new go source files
* copy all files from `$GOSRC/src/html/template/` into `html/template`
* add one function `StripTags` which calls `stripTags`
* run all unit tests
* commit all changes
* create new tag matching go version (e.g. v1.9.2)

Build script:

```bash
#!/usr/bin/env bash
set -eru -o pipefail
# exit on error
# exit on uninitialized variables
# enter restricted shell https://www.gnu.org/s/bash/manual/html_node/The-Restricted-Shell.html

URL='https://redirector.gvt1.com/edgedl/go/go1.9.2.src.tar.gz'
curl -L --silent "$URL" -o "go.tar.gz"
tar -zxf "go.tar.gz"

rm -rf "html/template/*"
cp "go/LICENSE" "./"
cp "go/PATENTS" "./"
cp "go/VERSION" "./"
cp -a "go/src/html/template/" "html/template/"
cp "export.go.tpl" "html/template/export.go"

rm -f "go.tar.gz"
rm -rf "./go/"
```

## Security

This package uses the unexported `stripTags` function from `html/template`. That works for most normal use cases, when you want to completely strip HTML tags.

If you need to sanitize potentially unsafe user input, while preserving some valid html tags, consider using HTML sanitizer libraries such as [Bluemonday](https://github.com/microcosm-cc/bluemonday).

## License

The original [go license](https://github.com/golang/go/blob/master/LICENSE). Please have a look at the [LICENSE](LICENSE) for more details.
