package ipldzec

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"

	cid "github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-node"
	mh "github.com/multiformats/go-multihash"
)

type Tx struct {
	Version    uint32
	Inputs     []*txIn
	Outputs    []*txOut
	LockTime   uint32
	JoinSplits []*JSDescription
	JSPubKey   []byte
	JSSig      []byte
}

func (t *Tx) Cid() *cid.Cid {
	h, _ := mh.Sum(t.RawData(), mh.DBL_SHA2_256, -1)
	return cid.NewCidV1(cid.BitcoinTx, h)
}

func (t *Tx) Links() []*node.Link {
	var out []*node.Link
	for i, input := range t.Inputs {
		lnk := txHashToLink(input.PrevTxHash)
		lnk.Name = fmt.Sprintf("inputs/%d/prevTx", i)
		out = append(out, lnk)
	}
	return out
}

func (t *Tx) RawData() []byte {
	buf := new(bytes.Buffer)
	i := make([]byte, 4)
	binary.LittleEndian.PutUint32(i, t.Version)
	buf.Write(i)
	writeVarInt(buf, uint64(len(t.Inputs)))
	for _, inp := range t.Inputs {
		inp.WriteTo(buf)
	}

	writeVarInt(buf, uint64(len(t.Outputs)))
	for _, out := range t.Outputs {
		out.WriteTo(buf)
	}

	binary.LittleEndian.PutUint32(i, t.LockTime)
	buf.Write(i)
	return buf.Bytes()
}

func (t *Tx) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "bitcoin_tx",
	}
}

func (t *Tx) Resolve(path []string) (interface{}, []string, error) {
	switch path[0] {
	case "version":
		return t.Version, path[1:], nil
	case "lockTime":
		return t.LockTime, path[1:], nil
	case "inputs":
		if len(path) == 1 {
			return t.Inputs, nil, nil
		}

		index, err := strconv.Atoi(path[1])
		if err != nil {
			return nil, nil, err
		}

		if index >= len(t.Inputs) || index < 0 {
			return nil, nil, fmt.Errorf("index out of range")
		}

		return t.Inputs[index], path[2:], nil
	case "outputs":
		if len(path) == 1 {
			return t.Outputs, nil, nil
		}

		index, err := strconv.Atoi(path[1])
		if err != nil {
			return nil, nil, err
		}

		if index >= len(t.Outputs) || index < 0 {
			return nil, nil, fmt.Errorf("index out of range")
		}

		return t.Outputs[index], path[2:], nil
	default:
		return nil, nil, fmt.Errorf("no such link")
	}
}

func (t *Tx) ResolveLink(path []string) (*node.Link, []string, error) {
	panic("NYI")
}

func (t *Tx) Size() (uint64, error) {
	return uint64(len(t.RawData())), nil
}

func (t *Tx) Stat() (*node.NodeStat, error) {
	panic("NYI")
}

func (t *Tx) Copy() node.Node {
	nt := *t // cheating shallow copy
	return &nt
}

func (t *Tx) String() string {
	return fmt.Sprintf("zcash transaction")
}

func (t *Tx) Tree(p string, depth int) []string {
	out := []string{"version", "timeLock", "inputs", "outputs"}
	for i, _ := range t.Inputs {
		out = append(out, "inputs/"+fmt.Sprint(i))
	}
	for i, _ := range t.Outputs {
		out = append(out, "outputs/"+fmt.Sprint(i))
	}
	return out
}

func (t *Tx) BTCSha() []byte {
	mh, _ := mh.Sum(t.RawData(), mh.DBL_SHA2_256, -1)
	return []byte(mh[2:])
}

func (t *Tx) HexHash() string {
	return hex.EncodeToString(revString(t.BTCSha()))
}

func txHashToLink(b []byte) *node.Link {
	mhb, _ := mh.Encode(b, mh.DBL_SHA2_256)
	c := cid.NewCidV1(cid.BitcoinTx, mhb)
	return &node.Link{Cid: c}
}

type txIn struct {
	PrevTxHash  []byte
	PrevTxIndex uint32
	Script      []byte
	SeqNo       uint32
}

func (i *txIn) WriteTo(w io.Writer) error {
	buf := make([]byte, 36)
	copy(buf[:32], i.PrevTxHash)
	binary.LittleEndian.PutUint32(buf[32:36], i.PrevTxIndex)
	w.Write(buf)

	writeVarInt(w, uint64(len(i.Script)))
	w.Write(i.Script)
	binary.LittleEndian.PutUint32(buf[:4], i.SeqNo)
	w.Write(buf[:4])
	return nil
}

type txOut struct {
	Value  uint64
	Script []byte
}

func (o *txOut) WriteTo(w io.Writer) error {
	val := make([]byte, 8)
	binary.LittleEndian.PutUint64(val, o.Value)
	w.Write(val)
	writeVarInt(w, uint64(len(o.Script)))
	w.Write(o.Script)
	return nil
}

var _ node.Node = (*Tx)(nil)
