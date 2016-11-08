package ipldzec

type JSDescription struct {
	OldVal       uint64
	NewVal       uint64
	Anchor       []byte
	Nullifiers   [][]byte
	Commitments  [][]byte
	EphemeralKey []byte
	CipherTexts  [][]byte
	RandomSeed   []byte
	Macs         [][]byte
	Proof        []byte
}
