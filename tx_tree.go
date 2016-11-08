package ipldzec

import (
	"encoding/json"
	"fmt"

	node "gx/ipfs/QmVtyW4wZg6Aic31zSX9cHCjj6Lyt1jY68S4uXF61ZaWLX/go-ipld-node"
	mh "gx/ipfs/QmYDds3421prZgqKbLpEK7T9Aa2eVdQ7o3YarX1LVLdP2J/go-multihash"
	cid "gx/ipfs/QmbTGYCo96Z9hiG37D9zeErFo5GjrEPcqdh7PJX1HTM73E/go-cid"
)

type TxTree struct {
	Left  *node.Link
	Right *node.Link
}

func (t *TxTree) ZECSha() []byte {
	mh, _ := mh.Sum(t.RawData(), mh.DBL_SHA2_256, -1)
	return []byte(mh[2:])
}

func (t *TxTree) Cid() *cid.Cid {
	h, _ := mh.Sum(t.RawData(), mh.DBL_SHA2_256, -1)
	return cid.NewCidV1(cid.ZcashTx, h)
}

func (t *TxTree) Links() []*node.Link {
	return []*node.Link{t.Left, t.Right}
}

func (t *TxTree) RawData() []byte {
	out := make([]byte, 64)
	lbytes := t.Left.Cid.Hash()[2:]
	copy(out[:32], lbytes)

	rbytes := t.Right.Cid.Hash()[2:]
	copy(out[32:], rbytes)

	return out
}

func (t *TxTree) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"type": "zcash_tx_tree",
	}
}

func (t *TxTree) Resolve(path []string) (interface{}, []string, error) {
	if len(path) == 0 {
		return nil, nil, fmt.Errorf("zero length path")
	}

	switch path[0] {
	case "0":
		return t.Left, path[1:], nil
	case "1":
		return t.Right, path[1:], nil
	default:
		return nil, nil, fmt.Errorf("no such link")

	}
}

func (t *TxTree) MarshalJSON() ([]byte, error) {
	return json.Marshal([]*Link{{t.Left.Cid}, {t.Right.Cid}})
}

func (t *TxTree) ResolveLink(path []string) (*node.Link, []string, error) {
	out, rest, err := t.Resolve(path)
	if err != nil {
		return nil, nil, err
	}

	lnk, ok := out.(*node.Link)
	if ok {
		return lnk, rest, nil
	}

	return nil, nil, fmt.Errorf("path did not lead to link")
}

func (t *TxTree) Size() (uint64, error) {
	return uint64(len(t.RawData())), nil
}

func (t *TxTree) Stat() (*node.NodeStat, error) {
	panic("NYI")
}

func (t *TxTree) String() string {
	return fmt.Sprintf("[zcash transaction tree]")
}

func (t *TxTree) Tree() []string {
	return []string{"0", "1"}
}
