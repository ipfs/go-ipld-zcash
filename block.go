package ipldzec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	cid "github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-node"
	mh "github.com/multiformats/go-multihash"
)

type Block struct {
	rawdata []byte

	Version       uint32 `json:"version"`
	PreviousBlock []byte `json:"parent"`
	MerkleRoot    []byte `json:"merkle_root"`
	Timestamp     uint32 `json:"timestamp"`
	Difficulty    uint32 `json:"difficulty"`
	Nonce         []byte `json:"nonce"`
	Solution      []byte
	ReservedHash  []byte

	cid *cid.Cid
}

type Link struct {
	Target *cid.Cid
}

// assert that Block matches the Node interface for ipld
var _ node.Node = (*Block)(nil)

func (b *Block) Cid() *cid.Cid {
	h, _ := mh.Sum(b.header(), mh.DBL_SHA2_256, -1)
	return cid.NewCidV1(cid.BitcoinBlock, h)
}

func (b *Block) RawData() []byte {
	return b.header()
}

func (b *Block) Links() []*node.Link {
	mrhash, _ := mh.Encode(b.MerkleRoot, mh.DBL_SHA2_256)
	c := cid.NewCidV1(cid.BitcoinTx, mrhash)

	return []*node.Link{
		{
			Name: "merkleRoot",
			Cid:  c,
		},
	}
}

func (b *Block) Loggable() map[string]interface{} {
	// TODO: more helpful info here
	return map[string]interface{}{
		"type": "zcash_block",
	}
}

// Resolve attempts to traverse a path through this block.
func (b *Block) Resolve(path []string) (interface{}, []string, error) {
	if len(path) == 0 {
		return nil, nil, fmt.Errorf("zero length path")
	}
	switch path[0] {
	case "version":
		return b.Version, path[1:], nil
	case "timestamp":
		return b.Timestamp, path[1:], nil
	case "difficulty":
		return b.Difficulty, path[1:], nil
	case "nonce":
		return b.Nonce, path[1:], nil
	case "parent":
		blkhash, _ := mh.Encode(b.PreviousBlock, mh.DBL_SHA2_256)
		c := cid.NewCidV1(cid.BitcoinBlock, blkhash)

		return &node.Link{Cid: c}, path[1:], nil
	case "tx":
		txroothash, _ := mh.Encode(b.MerkleRoot, mh.DBL_SHA2_256)
		c := cid.NewCidV1(cid.BitcoinTx, txroothash)

		return &node.Link{Cid: c}, path[1:], nil

	default:
		return nil, nil, fmt.Errorf("no such link")
	}
}

// ResolveLink is a helper function that allows easier traversal of links through blocks
func (b *Block) ResolveLink(path []string) (*node.Link, []string, error) {
	out, rest, err := b.Resolve(path)
	if err != nil {
		return nil, nil, err
	}

	lnk, ok := out.(*node.Link)
	if !ok {
		return nil, nil, fmt.Errorf("object at path was not a link")
	}

	return lnk, rest, nil
}

// header generates a serialized block header for this block
func (b *Block) header() []byte {
	buf := bytes.NewBuffer(make([]byte, 112))

	i := make([]byte, 4)
	binary.LittleEndian.PutUint32(i, b.Version)
	buf.Write(i)

	buf.Write(b.PreviousBlock)
	buf.Write(b.MerkleRoot)
	buf.Write(b.ReservedHash)

	binary.LittleEndian.PutUint32(i, b.Timestamp)
	buf.Write(i)

	binary.LittleEndian.PutUint32(i, b.Difficulty)
	buf.Write(i)

	buf.Write(b.Nonce)

	writeVarInt(buf, uint64(len(b.Solution)))
	buf.Write(b.Solution)

	return buf.Bytes()
}

func (b *Block) Size() (uint64, error) {
	return uint64(len(b.rawdata)), nil
}

func (b *Block) Stat() (*node.NodeStat, error) {
	panic("NYI")
}

func (b *Block) String() string {
	return fmt.Sprintf("zcash block")
}

func (b *Block) Tree(p string, depth int) []string {
	// TODO: this isnt a correct implementation yet
	return []string{"difficulty", "nonce", "version", "timestamp", "tx"}
}

func (b *Block) BTCSha() []byte {
	blkmh, _ := mh.Sum(b.header(), mh.DBL_SHA2_256, -1)
	return blkmh[2:]
}

func (b *Block) HexHash() string {
	return hex.EncodeToString(revString(b.BTCSha()))
}

func (b *Block) Copy() node.Node {
	nb := *b // cheating shallow copy
	return &nb
}

func revString(s []byte) []byte {
	b := make([]byte, len(s))
	for i, v := range []byte(s) {
		b[len(b)-(i+1)] = v
	}
	return b
}
