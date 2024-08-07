package githubfoundations

import "github.com/zclconf/go-cty/cty"

func toCtyValueSlice(values []string) cty.Value {
	if len(values) == 0 {
		return cty.ListValEmpty(cty.String)
	}

	ctyValues := make([]cty.Value, len(values))
	for _, v := range values {
		ctyValues = append(ctyValues, cty.StringVal(v))
	}
	return cty.ListVal(ctyValues)
}
