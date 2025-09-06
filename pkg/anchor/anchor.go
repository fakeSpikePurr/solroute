package anchor

import (
	"crypto/sha256"
	"fmt"
)

func GetDiscriminator(namespace string, name string) []byte {
	preimage := fmt.Sprintf("%s:%s", namespace, name)
	hash := sha256.Sum256([]byte(preimage))
	return hash[:8]
}
