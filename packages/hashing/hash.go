package hashing

import (
	"encoding/hex"
	"github.com/mr-tron/base58"
	"hash"
	"io"

	// github.com/mr-tron/base58
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
	"math/rand"
)

const HashSize = 32

type HashValue [HashSize]byte

var (
	nilHash  HashValue
	NilHash  = &nilHash
	allFHash HashValue
	AllFHash = &allFHash
)

func init() {
	if getHash().Size() != HashSize {
		panic("hash size != 32")
	}
	for i := range allFHash {
		allFHash[i] = 0xFF
	}
}

func getHash() hash.Hash {
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(err)
	}
	return h
}

func (h HashValue) String() string {
	return base58.Encode(h[:])
}

func (h HashValue) Short() string {
	return base58.Encode((h)[:6]) + ".."
}

func (h HashValue) Shortest() string {
	return hex.EncodeToString((h)[:4])
}

func (h *HashValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.String())
}

func (h *HashValue) UnmarshalJSON(buf []byte) error {
	var s string
	err := json.Unmarshal(buf, &s)
	if err != nil {
		return err
	}
	ret, err := HashValueFromBase58(s)
	if err != nil {
		return err
	}
	copy(h[:], ret[:])
	return nil
}

func HashValueFromBytes(b []byte) (HashValue, error) {
	if len(b) != HashSize {
		return nilHash, errors.New("wrong HashValue bytes length")
	}
	var ret HashValue
	copy(ret[:], b)
	return ret, nil
}

func HashValueFromBase58(s string) (HashValue, error) {
	b, err := base58.Decode(s)
	if err != nil {
		return nilHash, err
	}
	return HashValueFromBytes(b)
}

// HashData Blake2b
func HashData(data ...[]byte) (ret HashValue) {
	h := getHash()
	for _, d := range data {
		h.Write(d)
	}
	copy(ret[:], h.Sum(nil))
	return
}

func HashStrings(str ...string) HashValue {
	tarr := make([][]byte, len(str))
	for i, s := range str {
		tarr[i] = []byte(s)
	}
	return HashData(tarr...)
}

func RandomHash(rnd *rand.Rand) *HashValue {
	s := ""
	if rnd == nil {
		s = fmt.Sprintf("%d", rand.Int())
	} else {
		s = fmt.Sprintf("%d", rnd.Int())
	}
	ret := HashStrings(s, s, s)
	return &ret
}

func (h *HashValue) Write(w io.Writer) error {
	_, err := w.Write(h[:])
	return err
}

func (h *HashValue) Read(r io.Reader) error {
	n, err := r.Read(h[:])
	if err != nil {
		return err
	}
	if n != HashSize {
		return errors.New("not enough bytes for HashValue")
	}
	return nil
}
