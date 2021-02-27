package util

import (
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"
)

type DiffieHellman struct {
	base       *big.Int
	prime      *big.Int
	privateKey *big.Int
	publicKey  *big.Int
}

func NewDiffieHellman(base *big.Int, prime *big.Int) DiffieHellman {
	return DiffieHellman{base: base, prime: prime}
}

func (this *DiffieHellman) GenerateKeys(max *big.Int) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	this.privateKey = big.NewInt(0).Add(big.NewInt(0).Rand(rnd, max), big.NewInt(100))
	this.publicKey = big.NewInt(0).Exp(this.base, this.privateKey, this.prime)
}

func (this *DiffieHellman) GenerateKeysWithPrivateKey(privateKey string) bool {
	var ok bool
	pKey := big.NewInt(0)
	if pKey, ok = pKey.SetString(privateKey, 0); !ok {
		return false
	}
	this.privateKey = pKey
	this.publicKey = big.NewInt(0).Exp(this.base, this.privateKey, this.prime)
	return true
}

func (this *DiffieHellman) GetPublicKey() string {
	return this.publicKey.Text(10)
}

func (this *DiffieHellman) ComputeSecret(peerPublicKey string) (string, bool) {
	var ok bool
	peerKey := big.NewInt(0)
	if peerKey, ok = peerKey.SetString(peerPublicKey, 0); !ok {
		return "", false
	}
	return big.NewInt(0).Exp(peerKey, this.privateKey, this.prime).Text(10), true
}

func DiffieHellmanTest() {
	base := big.NewInt(5)
	prime := big.NewInt(0)
	prime, _ = prime.SetString("a0b2c2cdaccec0401f58c597", 16)
	fmt.Println("prime:", prime.String())
	max := big.NewInt(math.MaxInt64)

	dh := NewDiffieHellman(base, prime)
	dh.GenerateKeys(max)

	dh2 := NewDiffieHellman(base, prime)
	dh2.GenerateKeys(max)

	fmt.Println(dh.ComputeSecret(dh2.GetPublicKey()))
	fmt.Println(dh2.ComputeSecret(dh.GetPublicKey()))
}
