package names

import (
	"encoding/binary"
	"math/rand"
	_ "unsafe"

	_ "github.com/moby/moby/pkg/namesgenerator"
)

// we have to work with fixed array sizes here as they are also
// used in the original package and there is no way to have a
// dynamically sized array
var (
	//go:linkname left github.com/moby/moby/pkg/namesgenerator.left
	left [108]string

	//go:linkname right github.com/moby/moby/pkg/namesgenerator.right
	right [237]string
)

// GetDeterministicName will return the same name when given the same
// seed value. It also uses only the first 8 bytes of the seed value
// so we recommend inserting a hash value
func GetDeterministicName(seedValue []byte) string {
	// make sure the seed is at least 8 bytes long
	if len(seedValue) < 8 {
		for i := len(seedValue); i < 8; i++ {
			seedValue = append(seedValue, 0)
		}
	}
	source := rand.NewSource(int64(binary.BigEndian.Uint64(seedValue)))
	r := rand.New(source)
	return left[r.Intn(len(left))] + "_" + right[r.Intn(len(right))]
}
