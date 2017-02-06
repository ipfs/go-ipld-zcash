package ipldzec

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/ipfs/go-ipld-node"
)

func loadTestBlock() (*Block, []node.Node, []byte, error) {
	fidata, err := ioutil.ReadFile("blk.bin")
	data, err := hex.DecodeString(string(fidata[:len(fidata)-1]))
	if err != nil {
		return nil, nil, nil, err
	}

	out, err := DecodeBlockMessage(data)
	if err != nil {
		return nil, nil, nil, err
	}

	blk := out[0].(*Block)
	return blk, out[1:], data, nil
}

func TestBlockDecoding(t *testing.T) {
	blk, txs, data, err := loadTestBlock()
	if err != nil {
		t.Fatal(err)
	}

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

	if !blk.MerkleRoot.Equals(txs[len(txs)-1].Cid()) {
		t.Fatal("parsed txset didnt match merkle root")
	}
}

func TestJsonMarshaling(t *testing.T) {
	blk, _, _, err := loadTestBlock()
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(blk)
	if err != nil {
		t.Fatal(err)
	}

	var i map[string]interface{}
	err = json.Unmarshal(data, &i)
	if err != nil {
		t.Fatal(err)
	}

	iparent, ok := i["parent"]
	if !ok {
		t.Fatal("should have a parent")
	}

	parentmap, ok := iparent.(map[string]interface{})
	if !ok {
		t.Fatal("parent link should have been a map")
	}

	cidval, ok := parentmap["/"]
	if !ok {
		t.Fatal("parent cid should have had a key of '/'")
	}

	if cidval.(string) != "z4QJh987XTweMqSLmNakYhbReGF55QBJY1P3WqgSYKsfWuwCvHM" {
		t.Fatal("got wrong link")
	}
}
