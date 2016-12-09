package ipldzec

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"testing"
)

func TestBlockDecoding(t *testing.T) {
	fidata, err := ioutil.ReadFile("blk.bin")
	data, err := hex.DecodeString(string(fidata[:len(fidata)-1]))
	if err != nil {
		t.Fatal(err)
	}

	out, err := DecodeBlockMessage(data)
	if err != nil {
		t.Fatal(err)
	}

	blk := out[0].(*Block)
	outputdata := blk.header()
	if blk.Version != 4 {
		t.Fatal("block version should be 4")
	}

	if blk.HexHash() != "000000002c67a4a2351da58b0822193018e95abc94f243d4d9fdcefed81f45e1" {
		t.Fatal("output hash didnt match")
	}
	if !bytes.Equal(data[:len(outputdata)], outputdata) {
		t.Fatal("couldnt reconstruct object correctly")
	}

	txt := out[len(out)-1].(*TxTree)
	t.Log(txt.ZECSha())
	t.Log(blk.MerkleRoot)
	t.Log(cidToHash(blk.MerkleRoot))
}
