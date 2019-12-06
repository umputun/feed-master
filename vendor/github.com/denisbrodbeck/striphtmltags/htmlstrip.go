/*
Package striphtmltags strips HTML tags from strings.


*/
package striphtmltags // import "github.com/denisbrodbeck/striphtmltags"

import "github.com/denisbrodbeck/striphtmltags/html/template"

// StripTags takes a snippet of HTML and returns only the text content.
//
// For example, `<b>&iexcl;Hi!</b> <script>...</script>` -> `&iexcl;Hi! `.
func StripTags(html string) string {
	return template.StripTags(html)
}
