package templateutil

import (
	"bytes"
	"text/template"
)

func ResolveMap(m map[string]any, data map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		switch val := v.(type) {
		case string:
			t, err := template.New("").Option("missingkey=zero").Parse(val)
			if err != nil {
				out[k] = val
				continue
			}
			var buf bytes.Buffer
			if err := t.Execute(&buf, data); err != nil {
				out[k] = val
				continue
			}
			out[k] = buf.String()
		default:
			out[k] = v
		}
	}
	return out
}
