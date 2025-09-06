package utils

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/mr-tron/base58"
)

// KeyPair represents a public-private key pair
type KeyPair struct {
	PublicKey  string
	PrivateKey string
}

// FindKeyPairWithPrefix finds a Solana keypair with the specified prefix
// prefix: the desired prefix for the public key
// concurrency: number of concurrent workers to use
// Returns the found keypair or an error
func FindKeyPairWithPrefix(prefix string, concurrency int) (*KeyPair, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan *KeyPair, 1)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go workerWithPrefix(ctx, &wg, results, prefix)
	}

	// Wait for first result
	select {
	case result := <-results:
		cancel()  // Stop all workers
		wg.Wait() // Wait for all workers to finish
		return result, nil
	case <-time.After(5 * time.Minute): // 5 minute timeout
		cancel()
		wg.Wait()
		return nil, fmt.Errorf("timeout: could not find keypair with prefix %s", prefix)
	}
}

// FindKeyPairWithSuffix finds a Solana keypair with the specified suffix
// suffix: the desired suffix for the public key
// concurrency: number of concurrent workers to use
// Returns the found keypair or an error
func FindKeyPairWithSuffix(suffix string, concurrency int) (*KeyPair, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	results := make(chan *KeyPair, 1)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go workerWithSuffix(ctx, &wg, results, suffix)
	}

	// Wait for first result
	select {
	case result := <-results:
		cancel()  // Stop all workers
		wg.Wait() // Wait for all workers to finish
		return result, nil
	case <-time.After(5 * time.Minute): // 5 minute timeout
		cancel()
		wg.Wait()
		return nil, fmt.Errorf("timeout: could not find keypair with suffix %s", suffix)
	}
}

func generateKeypair() (pubKey string, privKey string, err error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	pub := priv.Public().(ed25519.PublicKey)

	pubStr := base58.Encode(pub)
	privStr := base58.Encode(priv)

	return pubStr, privStr, nil
}

func workerWithPrefix(ctx context.Context, wg *sync.WaitGroup, results chan<- *KeyPair, prefix string) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			pub, priv, err := generateKeypair()
			if err != nil {
				continue
			}
			if len(pub) >= len(prefix) && pub[:len(prefix)] == prefix {
				select {
				case results <- &KeyPair{PublicKey: pub, PrivateKey: priv}:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

func workerWithSuffix(ctx context.Context, wg *sync.WaitGroup, results chan<- *KeyPair, suffix string) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		default:
			pub, priv, err := generateKeypair()
			if err != nil {
				continue
			}
			if len(pub) >= len(suffix) && pub[len(pub)-len(suffix):] == suffix {
				select {
				case results <- &KeyPair{PublicKey: pub, PrivateKey: priv}:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}
}
