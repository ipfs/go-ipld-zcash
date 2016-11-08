package ipldzec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"

	node "gx/ipfs/QmVtyW4wZg6Aic31zSX9cHCjj6Lyt1jY68S4uXF61ZaWLX/go-ipld-node"
	mh "gx/ipfs/QmYDds3421prZgqKbLpEK7T9Aa2eVdQ7o3YarX1LVLdP2J/go-multihash"
	cid "gx/ipfs/QmbTGYCo96Z9hiG37D9zeErFo5GjrEPcqdh7PJX1HTM73E/go-cid"
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
	return map[string]interface{}{
		"type": "zcash_block",
	}
}

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

func (b *Block) Tree() []string {
	return []string{"difficulty", "nonce", "version", "timestamp", "tx"}
}

func (b *Block) BTCSha() []byte {
	blkmh, _ := mh.Sum(b.header(), mh.DBL_SHA2_256, -1)
	return blkmh[2:]
}

func (b *Block) HexHash() string {
	return hex.EncodeToString(revString(b.BTCSha()))
}

func revString(s []byte) []byte {
	b := make([]byte, len(s))
	for i, v := range []byte(s) {
		b[len(b)-(i+1)] = v
	}
	return b
}
