package mocks

import (
	"bytes"
	"io"
)

var BpowWorkGenerateResponse = io.NopCloser(bytes.NewReader([]byte("{\n  \"data\": {\n    \"workGenerate\": \"00000001cce3db6c\"\n  }\n}")))
