package ipldzec

import (
	"encoding/binary"
	"io"
)

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

func (js *JSDescription) WriteTo(w io.Writer) (int, error) {
	buf := make([]byte, 16)

	binary.LittleEndian.PutUint64(buf, js.OldVal)
	binary.LittleEndian.PutUint64(buf[8:], js.NewVal)

	return writeMany(w, buf, js.Anchor, js.Nullifiers[0], js.Nullifiers[1], js.Commitments[0], js.Commitments[1], js.EphemeralKey, js.RandomSeed, js.Macs[0], js.Macs[1], js.Proof, js.CipherTexts[0], js.CipherTexts[1])
}

func writeMany(w io.Writer, bs ...[]byte) (int, error) {
	var total int
	for _, b := range bs {
		n, err := w.Write(b)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}
