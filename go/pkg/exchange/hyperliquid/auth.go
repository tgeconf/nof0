package hyperliquid

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	mathhex "github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/vmihailenco/msgpack/v5"
)

// Signer encapsulates signing behaviour for exchange actions.
type Signer interface {
	Sign(message []byte) (*Signature, error)
	GetAddress() string
}

// PrivateKeySigner signs payloads using an ECDSA private key.
type PrivateKeySigner struct {
	privateKey *ecdsa.PrivateKey
	address    string
}

// NewPrivateKeySigner constructs a signer from a hex-encoded private key string.
func NewPrivateKeySigner(privateKeyHex string) (*PrivateKeySigner, error) {
	keyHex := strings.TrimPrefix(strings.TrimSpace(privateKeyHex), "0x")
	if keyHex == "" {
		return nil, fmt.Errorf("hyperliquid: empty private key")
	}

	key, err := crypto.HexToECDSA(keyHex)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: decode private key: %w", err)
	}
	address := strings.ToLower(crypto.PubkeyToAddress(key.PublicKey).Hex())
	return &PrivateKeySigner{
		privateKey: key,
		address:    address,
	}, nil
}

// Sign produces an ECDSA signature for the provided digest.
func (s *PrivateKeySigner) Sign(message []byte) (*Signature, error) {
	if s == nil || s.privateKey == nil {
		return nil, errors.New("hyperliquid: signer not initialised")
	}
	if len(message) == 0 {
		return nil, errors.New("hyperliquid: empty message for signing")
	}
	if len(message) != 32 {
		return nil, fmt.Errorf("hyperliquid: expected 32-byte message hash, got %d bytes", len(message))
	}

	sigBytes, err := crypto.Sign(message, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: sign message: %w", err)
	}
	r := hex.EncodeToString(sigBytes[:32])
	sComponent := hex.EncodeToString(sigBytes[32:64])
	v := int(sigBytes[64]) + 27

	return &Signature{
		R: "0x" + r,
		S: "0x" + sComponent,
		V: v,
	}, nil
}

// GetAddress returns the signer wallet address.
func (s *PrivateKeySigner) GetAddress() string {
	if s == nil {
		return ""
	}
	return s.address
}

// signAction attaches signature metadata to an action.
func signAction(action interface{}, signer Signer, nonce int64, mainAddress string, vaultAddress string, isMainnet bool) (*ExchangeRequest, error) {
	if signer == nil {
		return nil, errors.New("hyperliquid: signer required")
	}
	if nonce <= 0 {
		nonce = time.Now().UnixMilli()
	}

	digest, err := buildEIP712Message(action, nonce, vaultAddress, isMainnet)
	if err != nil {
		return nil, err
	}
	sig, err := signer.Sign(digest)
	if err != nil {
		return nil, err
	}
	return &ExchangeRequest{
		Action:       action,
		Nonce:        nonce,
		Signature:    *sig,
		VaultAddress: vaultAddress,
	}, nil
}

// buildEIP712Message constructs the EIP-712 hash for the action.
func buildEIP712Message(action interface{}, nonce int64, vaultAddress string, isMainnet bool) ([]byte, error) {
	if nonce <= 0 {
		return nil, fmt.Errorf("hyperliquid: nonce must be positive")
	}
	var buf bytes.Buffer
	encoder := msgpack.NewEncoder(&buf)
	encoder.UseCompactInts(true)
	if err := encoder.Encode(action); err != nil {
		return nil, fmt.Errorf("hyperliquid: msgpack encode action: %w", err)
	}
	msgpackBytes := convertStr16ToStr8(buf.Bytes())

	var nonceBytes [8]byte
	binary.BigEndian.PutUint64(nonceBytes[:], uint64(nonce))

	payload := make([]byte, 0, len(msgpackBytes)+1+common.AddressLength+len(nonceBytes))
	payload = append(payload, msgpackBytes...)
	payload = append(payload, nonceBytes[:]...)

	if vaultAddress == "" {
		payload = append(payload, 0x00)
	} else {
		if !common.IsHexAddress(vaultAddress) {
			return nil, fmt.Errorf("hyperliquid: invalid vault address %q", vaultAddress)
		}
		payload = append(payload, 0x01)
		payload = append(payload, common.HexToAddress(vaultAddress).Bytes()...)
	}

	connectionID := crypto.Keccak256(payload)

	source := "a"
	if !isMainnet {
		source = "b"
	}

	chainID := int64(1337)
	domain := apitypes.TypedDataDomain{
		Name:              "Exchange",
		Version:           "1",
		ChainId:           mathhex.NewHexOrDecimal256(chainID),
		VerifyingContract: verifyingContractHex,
	}
	message := map[string]interface{}{
		"source":       source,
		"connectionId": connectionID,
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Agent": {
				{Name: "source", Type: "string"},
				{Name: "connectionId", Type: "bytes32"},
			},
		},
		PrimaryType: "Agent",
		Domain:      domain,
		Message:     message,
	}

	return typedDataHash(typedData)
}

const verifyingContractHex = "0x0000000000000000000000000000000000000000"

func convertStr16ToStr8(data []byte) []byte {
	result := make([]byte, 0, len(data))
	for i := 0; i < len(data); {
		if data[i] == 0xda && i+2 < len(data) {
			length := int(data[i+1])<<8 | int(data[i+2])
			if length < 256 && i+3+length <= len(data) {
				result = append(result, 0xd9, byte(length))
				result = append(result, data[i+3:i+3+length]...)
				i += 3 + length
				continue
			}
		}
		result = append(result, data[i])
		i++
	}
	return result
}

func typedDataHash(td apitypes.TypedData) ([]byte, error) {
	domainSeparator, err := td.HashStruct("EIP712Domain", td.Domain.Map())
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: hash domain: %w", err)
	}
	messageHash, err := td.HashStruct(td.PrimaryType, td.Message)
	if err != nil {
		return nil, fmt.Errorf("hyperliquid: hash primary type: %w", err)
	}
	raw := make([]byte, 0, 2+len(domainSeparator)+len(messageHash))
	raw = append(raw, 0x19, 0x01)
	raw = append(raw, domainSeparator...)
	raw = append(raw, messageHash...)
	return crypto.Keccak256(raw), nil
}
