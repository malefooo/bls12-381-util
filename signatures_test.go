package blsu

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestSecretKey_Deserialize(t *testing.T) {
	// TODO TestSecretKey_Deserialize
}

func TestSecretKey_Serialize(t *testing.T) {
	// TODO TestSecretKey_Serialize
}

func TestPubkey_Deserialize(t *testing.T) {
	// TODO TestPubkey_Deserialize
}

func TestPubkey_Serialize(t *testing.T) {
	// TODO TestPubkey_Serialize
}

func TestSignature_Deserialize(t *testing.T) {
	// TODO TestSignature_Deserialize
}

func TestSignature_Serialize(t *testing.T) {
	// TODO TestSignature_Serialize
}

func hex32(v string) (out [32]byte) {
	s, err := hex.DecodeString(v)
	if err != nil {
		panic(err)
	}
	if len(s) != 32 {
		panic(fmt.Sprintf("not 32 bytes: %x", s))
	}
	copy(out[:], s)
	return
}

func TestSkToPk(t *testing.T) {
	secret := hex32("263dbd792f5b1be47ed85f8938c0f29586af0d3ac7b977f21c278fe1462040e3")
	var sk SecretKey
	sk.Deserialize(&secret)
	pk, err := SkToPk(&sk)
	if err != nil {
		t.Fatal(err)
	}
	out := pk.Serialize()
	got := hex.EncodeToString(out[:])
	expected := "a491d1b0ecd9bb917989f0e74f0dea0422eac4a873e5e2644f368dffb9a6e20fd6e10c1b77654d067c0618f6e5a7f79a"
	if got != expected {
		t.Fatalf("expected %s, got pubkey %s", expected, got)
	}
}

type signTestCase struct {
	Input struct {
		Privkey hexStr32 `yaml:"privkey"`
		Message hexStr   `yaml:"message"`
	}
	Output hexStr `yaml:"output"`
}

func TestSign(t *testing.T) {
	runTestCases(t, "sign/small", func(t *testing.T, getData func(interface{})) {
		var data signTestCase
		getData(&data)
		var sk SecretKey
		if err := sk.Deserialize((*[32]byte)(&data.Input.Privkey)); err != nil {
			if len(data.Output) != 0 {
				t.Fatalf("unexpected failure: %v", err)
			}
			return
		} else {
			if len(data.Output) == 0 {
				t.Fatalf("expected failure, but did not")
			}
		}
		sig := Sign(&sk, data.Input.Message)
		res := sig.Serialize()
		if !bytes.Equal(res[:], data.Output) {
			t.Fatalf("got %x, expected %x", res, data.Output)
		}
	})
}

type aggregateTestCase struct {
	Input  []hexStr96 `yaml:"input"`
	Output *hexStr96  `yaml:"output"`
}

func TestAggregate(t *testing.T) {
	runTestCases(t, "aggregate/small", func(t *testing.T, getData func(interface{})) {
		var data aggregateTestCase
		getData(&data)
		inputs := make([]*Signature, len(data.Input), len(data.Input))
		for i, sig := range data.Input {
			inputs[i] = new(Signature)
			if err := inputs[i].Deserialize((*[96]byte)(&sig)); err != nil {
				if data.Output == nil {
					// expected failure
					return
				} else {
					t.Fatalf("unexpected failure, failed to deserialize signature %d (%x): %v", i, sig[:], err)
				}
			}
		}
		out, err := Aggregate(inputs)
		if err != nil {
			if data.Output == nil {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure, failed to aggregate signatures: %v", err)
			}
		} else {
			res := out.Serialize()
			if !bytes.Equal(res[:], data.Output[:]) {
				t.Fatalf("got %x, expected %x", res[:], data.Output[:])
			}
		}
	})
}

func TestAggregatePubkeys(t *testing.T) {
	// TODO TestAggregatePubkeys
}

type verifyTestCase struct {
	Input struct {
		Pubkey    hexStr48 `yaml:"pubkey"`
		Message   hexStr32 `yaml:"message"`
		Signature hexStr96 `yaml:"signature"` // the signature to verify against pubkey and message
	}
	Output bool `yaml:"output"` // VALID or INVALID
}

func TestVerify(t *testing.T) {
	runTestCases(t, "verify/small", func(t *testing.T, getData func(interface{})) {
		var data verifyTestCase
		getData(&data)
		var pub Pubkey
		if err := pub.Deserialize((*[48]byte)(&data.Input.Pubkey)); err != nil {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure, failed to deserialize pubkey (%x): %v", data.Input.Pubkey[:], err)
			}
		}
		var sig Signature
		if err := sig.Deserialize((*[96]byte)(&data.Input.Signature)); err != nil {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure, failed to deserialize signature (%x): %v", data.Input.Signature[:], err)
			}
		}
		res := Verify(&pub, data.Input.Message[:], &sig)
		if res != data.Output {
			t.Fatalf("expected different output, got %v, expected %v", res, data.Output)
		}
	})
}

type aggregateVerifyTestCase struct {
	Input struct {
		Pubkeys   []hexStr48 `yaml:"pubkeys"`
		Messages  []hexStr32 `yaml:"messages"`
		Signature hexStr     `yaml:"signature"`
	}
	Output bool `yaml:"output"` // VALID or INVALID
}

func parsePubkeys(input []hexStr48) (pubkeys []*Pubkey, err error) {
	pubkeys = make([]*Pubkey, len(input), len(input))
	for i, pub := range input {
		pubkeys[i] = new(Pubkey)
		if err := pubkeys[i].Deserialize((*[48]byte)(&pub)); err != nil {
			return nil, fmt.Errorf("failed to deserialize pubkey %d (%x): %v", i, pub[:], err)
		}
	}
	return pubkeys, nil
}

func TestAggregateVerify(t *testing.T) {
	runTestCases(t, "aggregate_verify/small", func(t *testing.T, getData func(interface{})) {
		var data aggregateVerifyTestCase
		getData(&data)
		pubkeys, err := parsePubkeys(data.Input.Pubkeys)
		if err != nil {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure: %v", err)
			}
		}
		messages := make([][]byte, len(data.Input.Messages), len(data.Input.Messages))
		for i := range data.Input.Messages {
			messages[i] = data.Input.Messages[i][:]
		}
		// Our signature Deserialization cannot deserialize anything else than 96 bytes, yay typing.
		// But the tests have invalid-signature cases for non-96 bytes, catch those.
		var sigData [96]byte
		if len(data.Input.Signature) != 96 {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("expected 96 byte signature, got %d bytes: %x", len(data.Input.Signature), data.Input.Signature[:])
			}
		} else {
			copy(sigData[:], data.Input.Signature)
		}
		var sig Signature
		if err := sig.Deserialize(&sigData); err != nil {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure, failed to deserialize signature (%x): %v", data.Input.Signature[:], err)
			}
		}
		res := AggregateVerify(pubkeys, messages, &sig)
		if res != data.Output {
			t.Fatalf("expected different output, got %v, expected %v", res, data.Output)
		}
	})
}

type fastAggregateVerifyTestCase struct {
	Input struct {
		Pubkeys   []hexStr48 `yaml:"pubkeys"`
		Message   hexStr32   `yaml:"message"`
		Signature hexStr96   `yaml:"signature"`
	}
	Output bool `yaml:"output"` // VALID or INVALID
}

func TestFastAggregateVerify(t *testing.T) {
	runTestCases(t, "fast_aggregate_verify/small", func(t *testing.T, getData func(interface{})) {
		var data fastAggregateVerifyTestCase
		getData(&data)
		pubkeys, err := parsePubkeys(data.Input.Pubkeys)
		if err != nil {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure: %v", err)
			}
		}
		message := data.Input.Message[:]
		var sig Signature
		if err := sig.Deserialize((*[96]byte)(&data.Input.Signature)); err != nil {
			if !data.Output {
				// expected failure
				return
			} else {
				t.Fatalf("unexpected failure, failed to deserialize signature (%x): %v", data.Input.Signature[:], err)
			}
		}
		res := FastAggregateVerify(pubkeys, message, &sig)
		if res != data.Output {
			t.Fatalf("expected different output, got %v, expected %v", res, data.Output)
		}
	})
}

func TestEth2FastAggregateVerify(t *testing.T) {
	// TODO TestEth2FastAggregateVerify
}

type hexStr []byte

func (v *hexStr) UnmarshalText(text []byte) error {
	if len(text) >= 2 && text[0] == '0' && text[1] == 'x' {
		text = text[2:]
	}
	l := hex.DecodedLen(len(text))
	dat := make([]byte, l, l)
	n, err := hex.Decode(dat, text)
	*v = dat
	fmt.Println("n", n)
	return err
}

func unmarshalHex(dst []byte, text []byte) error {
	if len(text) >= 2 && text[0] == '0' && text[1] == 'x' {
		text = text[2:]
	}
	l := hex.DecodedLen(len(text))
	if l != len(dst) {
		return fmt.Errorf("unexpected length, not %d bytes: %d", len(dst), l)
	}
	_, err := hex.Decode(dst, text)
	return err
}

type hexStr32 [32]byte

func (v *hexStr32) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	return unmarshalHex(v[:], text)
}

type hexStr48 [48]byte

func (v *hexStr48) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	return unmarshalHex(v[:], text)
}

type hexStr96 [96]byte

func (v *hexStr96) UnmarshalText(text []byte) error {
	if len(text) == 0 {
		return nil
	}
	return unmarshalHex(v[:], text)
}

var testDir = "../bls-tests/bls"

func runTestCases(t *testing.T, path string, runCase func(t *testing.T, getData func(interface{}))) {
	t.Run(path, func(t *testing.T) {
		casesPath := filepath.Join(testDir, path)
		casesDir := os.DirFS(casesPath)
		fs.WalkDir(casesDir, ".", func(path string, d fs.DirEntry, err error) error {
			// recurse into main dir
			if path == "." {
				return nil
			}
			// not a dir? skip it
			if !d.IsDir() {
				return nil
			}
			// can't open the file/dir? skip it
			if err != nil {
				return fs.SkipDir
			}
			// each sub-directory is a test-case
			t.Run(path, func(t *testing.T) {
				// run call-back to process test-case
				runCase(t, func(dst interface{}) {
					p := filepath.Join(d.Name(), "data.yaml")
					data, err := fs.ReadFile(casesDir, p)
					if err != nil {
						t.Fatalf("failed to read %q: %v", p, err)
						return
					}
					if err := yaml.Unmarshal(data, dst); err != nil {
						t.Fatalf("failed to decode %q: %v", p, err)
						return
					}
				})
			})
			// don't recurse into the directory further
			return fs.SkipDir
		})
	})
}
