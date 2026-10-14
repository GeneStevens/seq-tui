package record

import (
	"bytes"
	"encoding/json"
)

func compact(dst *bytes.Buffer, src []byte) error { return json.Compact(dst, src) }
