package util

import (
	"encoding/json"
	"fmt"
)

func Dump(o interface{}) {
	b, _ := json.MarshalIndent(o, "", "  ")
	fmt.Printf("--- DUMP ---\n\n%s\n\n", string(b))
}
