package utils

import (
	"strconv"
	"strings"
)

func EmbeddingToVectorLiteral(vec []float32) string {
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = strconv.FormatFloat(float64(v), 'f', -1, 32)
	}
	// pgvector expects: [0.1,0.2,0.3]
	return "[" + strings.Join(parts, ",") + "]"
}
