package template

// StripTags takes a snippet of HTML and returns only the text content.
//
// For example, `<b>&iexcl;Hi!</b> <script>...</script>` -> `&iexcl;Hi! `.
//
// This function exports the private html/template/stripTags function.
func StripTags(html string) string {
	return stripTags(html)
}
