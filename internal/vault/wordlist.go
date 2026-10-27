package vault

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"math/big"
	"strings"
	"sync"
)

// wordlistEN is the BIP-39 English word list, 2048 words.
//
//go:embed wordlist_en.txt
var wordlistEN []byte

var words = sync.OnceValue(func() []string {
	w := strings.Split(strings.TrimSuffix(string(wordlistEN), "\n"), "\n")
	if len(w) != 2048 {
		panic(fmt.Sprintf("vault: embedded word list has %d words", len(w)))
	}
	return w
})

const recoveryWords = 24

// recoveryCode draws 24 words uniformly from the list (264 bits).
func recoveryCode() ([]byte, error) {
	w := words()
	n := big.NewInt(int64(len(w)))
	var b strings.Builder
	for i := range recoveryWords {
		x, err := rand.Int(rand.Reader, n)
		if err != nil {
			return nil, fmt.Errorf("vault: %w", err)
		}
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w[x.Int64()])
	}
	return []byte(b.String()), nil
}
