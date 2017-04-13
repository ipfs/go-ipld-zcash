package ipldzec

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	cid "github.com/ipfs/go-cid"
	node "github.com/ipfs/go-ipld-format"
	mh "github.com/multiformats/go-multihash"
)

func DecodeBlockMessage(b []byte) ([]node.Node, error) {
	r := bytes.NewReader(b)
	blk, err := ReadBlock(r)
	if err != nil {
		return nil, err
	}

	nTx, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	var txs []node.Node
	for i := 0; i < nTx; i++ {
		tx, err := readTx(r)
		if err != nil {
			return nil, err
		}

		txs = append(txs, tx)
	}

	txtrees, err := mkMerkleTree(txs)
	if err != nil {
		return nil, err
	}

	out := []node.Node{blk}
	for _, tx := range txs {
		out = append(out, tx)
	}

	for _, txtree := range txtrees {
		out = append(out, txtree)
	}

	return out, nil
}

func mkMerkleTree(txs []node.Node) ([]*TxTree, error) {
	var out []*TxTree
	var next []node.Node
	layer := txs
	for len(layer) > 1 {
		if len(layer)%2 != 0 {
			layer = append(layer, layer[len(layer)-1])
		}
		for i := 0; i < len(layer)/2; i++ {
			var left, right node.Node
			left = layer[i*2]
			right = layer[(i*2)+1]

			t := &TxTree{
				Left:  &node.Link{Cid: left.Cid()},
				Right: &node.Link{Cid: right.Cid()},
			}

			out = append(out, t)
			next = append(next, t)
		}

		layer = next
		next = nil
	}

	return out, nil
}

func DecodeBlock(b []byte) (*Block, error) {
	return ReadBlock(bytes.NewReader(b))
}

func ReadBlock(r *bytes.Reader) (*Block, error) {
	var blk Block

	version := make([]byte, 4)
	_, err := io.ReadFull(r, version)
	if err != nil {
		return nil, err
	}
	blk.Version = binary.LittleEndian.Uint32(version)

	prevBlock := make([]byte, 32)
	_, err = io.ReadFull(r, prevBlock)
	if err != nil {
		return nil, err
	}

	blkhash, _ := mh.Encode(prevBlock, mh.DBL_SHA2_256)
	blk.Parent = cid.NewCidV1(cid.ZcashBlock, blkhash)

	merkleRoot := make([]byte, 32)
	_, err = io.ReadFull(r, merkleRoot)
	if err != nil {
		return nil, err
	}
	txroothash, _ := mh.Encode(merkleRoot, mh.DBL_SHA2_256)
	blk.MerkleRoot = cid.NewCidV1(cid.ZcashTx, txroothash)

	reserved := make([]byte, 32)
	_, err = io.ReadFull(r, reserved)
	if err != nil {
		return nil, err
	}
	blk.ReservedHash = reserved

	timestamp := make([]byte, 4)
	_, err = io.ReadFull(r, timestamp)
	if err != nil {
		return nil, err
	}
	blk.Timestamp = binary.LittleEndian.Uint32(timestamp)

	diff := make([]byte, 4)
	_, err = io.ReadFull(r, diff)
	if err != nil {
		return nil, err
	}
	blk.Difficulty = binary.LittleEndian.Uint32(diff)

	nonce := make([]byte, 32)
	_, err = io.ReadFull(r, nonce)
	if err != nil {
		return nil, err
	}
	blk.Nonce = nonce

	sollen, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	solution := make([]byte, sollen)
	_, err = io.ReadFull(r, solution)
	if err != nil {
		return nil, err
	}
	blk.Solution = solution

	return &blk, nil
}

func DecodeMaybeTx(b []byte) (node.Node, error) {
	if len(b) == 64 {
		return DecodeTxTree(b)
	}
	return DecodeTx(b)
}

func DecodeTx(b []byte) (*Tx, error) {
	r := bytes.NewReader(b)
	return readTx(r)
}

func DecodeTxTree(b []byte) (*TxTree, error) {
	if len(b) != 64 {
		return nil, fmt.Errorf("invalid tx tree data")
	}

	lnkL := txHashToLink(b[:32])
	lnkR := txHashToLink(b[32:])
	return &TxTree{
		Left:  lnkL,
		Right: lnkR,
	}, nil
}

func readTx(r *bytes.Reader) (*Tx, error) {
	var out Tx

	version := make([]byte, 4)
	_, err := io.ReadFull(r, version)
	if err != nil {
		return nil, err
	}
	out.Version = binary.LittleEndian.Uint32(version)

	inCtr, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	for i := 0; i < inCtr; i++ {
		txin, err := parseTxIn(r)
		if err != nil {
			return nil, err
		}

		out.Inputs = append(out.Inputs, txin)
	}

	outCtr, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	for i := 0; i < outCtr; i++ {
		txout, err := parseTxOut(r)
		if err != nil {
			return nil, err
		}

		out.Outputs = append(out.Outputs, txout)
	}

	lock_time := make([]byte, 4)
	_, err = io.ReadFull(r, lock_time)
	if err != nil {
		return nil, err
	}

	out.LockTime = binary.LittleEndian.Uint32(lock_time)

	if out.Version == 1 {
		return &out, nil
	}

	nJoinSplit, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	var joinsplits []*JSDescription
	for i := 0; i < nJoinSplit; i++ {
		js, err := readJoinSplit(r)
		if err != nil {
			return nil, err
		}

		joinsplits = append(joinsplits, js)
	}
	out.JoinSplits = joinsplits

	jsPubK := make([]byte, 32)
	_, err = io.ReadFull(r, jsPubK)
	if err != nil {
		return nil, err
	}
	out.JSPubKey = jsPubK

	jsSig := make([]byte, 64)
	_, err = io.ReadFull(r, jsSig)
	if err != nil {
		return nil, err
	}
	out.JSSig = jsSig

	return &out, nil
}

func parseTxIn(r *bytes.Reader) (*TxIn, error) {
	prevTxHash := make([]byte, 32)
	_, err := io.ReadFull(r, prevTxHash)
	if err != nil {
		return nil, err
	}

	prevTxIndex := make([]byte, 4)
	_, err = io.ReadFull(r, prevTxIndex)
	if err != nil {
		return nil, err
	}

	scriptLen, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	// Read Script
	script := make([]byte, scriptLen)
	_, err = io.ReadFull(r, script)
	if err != nil {
		return nil, err
	}

	seqNo := make([]byte, 4)
	_, err = io.ReadFull(r, seqNo)
	if err != nil {
		return nil, err
	}

	var ptxl *cid.Cid
	if !isBlank(prevTxHash) {
		ptxl = hashToCid(prevTxHash, cid.ZcashTx)
	}
	return &TxIn{
		PrevTx:      ptxl,
		PrevTxIndex: binary.LittleEndian.Uint32(prevTxIndex),
		Script:      script,
		SeqNo:       binary.LittleEndian.Uint32(seqNo),
	}, nil
}

func isBlank(b []byte) bool {
	for _, v := range b {
		if v != 0 {
			return false
		}
	}
	return true
}

func parseTxOut(r *bytes.Reader) (*TxOut, error) {
	value := make([]byte, 8)
	_, err := io.ReadFull(r, value)
	if err != nil {
		return nil, err
	}

	scriptLen, err := readVarint(r)
	if err != nil {
		return nil, err
	}

	script := make([]byte, scriptLen)
	_, err = io.ReadFull(r, script)
	if err != nil {
		return nil, err
	}

	// read script
	return &TxOut{
		Value:  binary.LittleEndian.Uint64(value),
		Script: script,
	}, nil
}

func readJoinSplit(r *bytes.Reader) (*JSDescription, error) {
	val := make([]byte, 8)
	_, err := io.ReadFull(r, val)
	if err != nil {
		return nil, err
	}
	vpub_old := binary.LittleEndian.Uint64(val)

	_, err = io.ReadFull(r, val)
	if err != nil {
		return nil, err
	}
	vpub_new := binary.LittleEndian.Uint64(val)

	anchor, err := readBuf(r, 32)
	if err != nil {
		return nil, err
	}

	nullifiers, err := readBuf(r, 64)
	if err != nil {
		return nil, err
	}

	commitments, err := readBuf(r, 64)
	if err != nil {
		return nil, err
	}

	ephKey, err := readBuf(r, 32)
	if err != nil {
		return nil, err
	}

	randSeed, err := readBuf(r, 32)
	if err != nil {
		return nil, err
	}

	vmacs, err := readBuf(r, 64)
	if err != nil {
		return nil, err
	}

	zkproof, err := readBuf(r, 296)
	if err != nil {
		return nil, err
	}

	encCiphertexts, err := readBuf(r, 1202)
	if err != nil {
		return nil, err
	}

	return &JSDescription{
		NewVal:       vpub_new,
		OldVal:       vpub_old,
		Anchor:       anchor,
		CipherTexts:  [][]byte{encCiphertexts[:601], encCiphertexts[601:]}, // TODO: thisones not in the right format yet
		Commitments:  [][]byte{commitments[:32], commitments[32:]},         // SAME
		Macs:         [][]byte{vmacs[:32], vmacs[32:]},                     // SAME
		EphemeralKey: ephKey,
		RandomSeed:   randSeed,
		Proof:        zkproof,
		Nullifiers:   [][]byte{nullifiers[:32], nullifiers[32:]},
	}, nil
}

func readBuf(r io.Reader, size int) ([]byte, error) {
	out := make([]byte, size)
	_, err := io.ReadFull(r, out)
	return out, err
}

func readVarint(r *bytes.Reader) (int, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}

	switch b {
	case 0xfd:
		buf := make([]byte, 2)
		_, err := r.Read(buf)
		if err != nil {
			return 0, err
		}
		return int(binary.LittleEndian.Uint16(buf)), nil
	case 0xfe:
		buf := make([]byte, 4)
		_, err := r.Read(buf)
		if err != nil {
			return 0, err
		}

		return int(binary.LittleEndian.Uint32(buf)), nil
	case 0xff:
		buf := make([]byte, 8)
		_, err := r.Read(buf)
		if err != nil {
			return 0, err
		}

		return int(binary.LittleEndian.Uint64(buf)), nil
	default:
		return int(b), nil
	}
}

func writeVarInt(w io.Writer, n uint64) error {
	var d []byte
	if n < 0xFD {
		d = []byte{byte(n)}
	} else if n <= 0xFFFF {
		d = make([]byte, 3)
		binary.LittleEndian.PutUint16(d[1:], uint16(n))
		d[0] = 0xFD
	} else if n <= 0xFFFFFFF {
		d = make([]byte, 5)
		binary.LittleEndian.PutUint32(d[1:], uint32(n))
		d[0] = 0xFE
	} else {
		d = make([]byte, 9)
		binary.LittleEndian.PutUint64(d[1:], n)
		d[0] = 0xFE
	}
	_, err := w.Write(d)
	return err
}
