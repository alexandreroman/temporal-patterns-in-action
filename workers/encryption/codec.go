// Package encryption — codec.go holds the AES-256-GCM PayloadCodec used by
// the encrypted scenario. Client and worker both register this codec so
// Temporal sees only ciphertext while application code sees cleartext.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"

	commonpb "go.temporal.io/api/common/v1"
	"google.golang.org/protobuf/proto"
)

// DemoKey is a fixed 32-byte key used by the demo. Real deployments would
// fetch this from a KMS per workflow/tenant — never ship a hard-coded key.
var DemoKey = []byte("temporal-demo-encryption-key-32b")

// Metadata markers stamped on encrypted payloads so Decode can recognise its
// own output and pass unrelated payloads through unchanged.
const (
	metaEncoding = "binary/encrypted"
	cipherName   = "AES-256-GCM"
	keyID        = "v1-demo"
)

// EncryptionCodec implements converter.PayloadCodec using AES-256-GCM.
// Package-level note: the codec wraps each Temporal payload (itself a
// protobuf message) into a fresh payload whose Data is nonce||ciphertext;
// the original payload — metadata and all — is recovered on Decode.
type EncryptionCodec struct {
	Key []byte
}

// Encode seals each incoming payload with AES-256-GCM. The original payload
// is marshalled as protobuf first so both its metadata and data are covered
// by the ciphertext.
func (c *EncryptionCodec) Encode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	gcm, err := newGCM(c.Key)
	if err != nil {
		return payloads, err
	}
	out := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		plaintext, err := proto.Marshal(p)
		if err != nil {
			return payloads, fmt.Errorf("marshal payload: %w", err)
		}
		nonce := make([]byte, gcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			return payloads, fmt.Errorf("read nonce: %w", err)
		}
		sealed := gcm.Seal(nil, nonce, plaintext, nil)
		out[i] = &commonpb.Payload{
			Metadata: map[string][]byte{
				"encoding":          []byte(metaEncoding),
				"encryption-cipher": []byte(cipherName),
				"encryption-key-id": []byte(keyID),
			},
			Data: append(nonce, sealed...),
		}
	}
	return out, nil
}

// Decode opens payloads produced by Encode. Payloads with a different
// encoding are passed through untouched so the codec composes cleanly with
// whatever else the server returns.
func (c *EncryptionCodec) Decode(payloads []*commonpb.Payload) ([]*commonpb.Payload, error) {
	gcm, err := newGCM(c.Key)
	if err != nil {
		return payloads, err
	}
	out := make([]*commonpb.Payload, len(payloads))
	for i, p := range payloads {
		if string(p.Metadata["encoding"]) != metaEncoding {
			out[i] = p
			continue
		}
		ns := gcm.NonceSize()
		if len(p.Data) < ns {
			return payloads, fmt.Errorf("payload too short for nonce")
		}
		nonce, ciphertext := p.Data[:ns], p.Data[ns:]
		plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			return payloads, fmt.Errorf("gcm open: %w", err)
		}
		var orig commonpb.Payload
		if err := proto.Unmarshal(plaintext, &orig); err != nil {
			return payloads, fmt.Errorf("unmarshal payload: %w", err)
		}
		out[i] = &orig
	}
	return out, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("new aes cipher: %w", err)
	}
	return cipher.NewGCM(block)
}
