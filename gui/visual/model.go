package visual

import (
	"Blockchain/core"
	"Blockchain/gui/state"
	"Blockchain/gui/ui"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type refKind int

const (
	refNone refKind = iota
	refBlock
	refTx
	refWallet
)

type ref struct {
	kind   refKind
	id     string
	height int
}

func blockRef(height int) ref {
	return ref{kind: refBlock, id: strconv.Itoa(height), height: height}
}

func txRef(id string) ref {
	return ref{kind: refTx, id: id}
}

func walletRef(address string) ref {
	return ref{kind: refWallet, id: address}
}

type output struct {
	value   int
	owner   string
	spentBy string
}

type input struct {
	txid  string
	vout  int
	owner string
	value int
}

type txn struct {
	id       string
	block    int
	coinbase bool
	fee      int
	inputs   []input
	outputs  []output
}

func (t *txn) pending() bool {
	return t.block < 0
}

func (t *txn) sender() string {
	if len(t.inputs) == 0 {
		return ""
	}
	return t.inputs[0].owner
}

func (t *txn) transfer() (int, string) {
	sender := t.sender()
	amount, to, total := 0, "", 0
	for _, out := range t.outputs {
		total += out.value
		if out.owner != sender || t.coinbase {
			amount += out.value
			if to == "" {
				to = out.owner
			}
		}
	}
	if to == "" {
		return total, sender
	}
	return amount, to
}

type blockInfo struct {
	height int
	hash   string
	nonce  int
	bits   int
	txs    []*txn
}

type walletInfo struct {
	address string
	name    string
	named   bool
	balance int
}

type model struct {
	blocks  []*blockInfo
	mempool []*txn
	wallets []*walletInfo
	txs     map[string]*txn
	maxPool int
}

func newModel() *model {
	return &model{txs: map[string]*txn{}}
}

func (m *model) wallet(address string) *walletInfo {
	for _, w := range m.wallets {
		if w.address == address {
			return w
		}
	}
	return nil
}

func (m *model) walletName(address string) string {
	if w := m.wallet(address); w != nil {
		return w.name
	}
	return ui.ShortAddress(address)
}

func (m *model) block(height int) *blockInfo {
	if height < 0 || height >= len(m.blocks) {
		return nil
	}
	return m.blocks[height]
}

func (m *model) add(t *txn) {
	m.txs[t.id] = t
}

func (m *model) link() {
	for _, t := range m.txs {
		for i := range t.outputs {
			t.outputs[i].spentBy = ""
		}
	}
	for _, t := range m.txs {
		for i, in := range t.inputs {
			source, ok := m.txs[in.txid]
			if !ok || in.vout < 0 || in.vout >= len(source.outputs) {
				continue
			}
			source.outputs[in.vout].spentBy = t.id
			t.inputs[i].value = source.outputs[in.vout].value
			if t.inputs[i].owner == "" {
				t.inputs[i].owner = source.outputs[in.vout].owner
			}
		}
	}
}

func fromSnapshot(snap *state.Snapshot) *model {
	m := newModel()
	if snap == nil {
		return m
	}
	m.maxPool = snap.MempoolMax

	convert := func(tx *core.Transaction, height int) *txn {
		t := &txn{id: hex.EncodeToString(tx.ID), block: height, coinbase: tx.IsCoinbase(), fee: tx.Fee}
		if !t.coinbase {
			for _, in := range tx.Vin {
				t.inputs = append(t.inputs, input{txid: hex.EncodeToString(in.Txid), vout: in.Vout, owner: core.PubKeyToAddress(in.PubKey)})
			}
		}
		for _, out := range tx.Vout {
			owner := ""
			if len(out.PubKeyHash) > 0 {
				owner = core.PubKeyHashToAddress(out.PubKeyHash)
			}
			t.outputs = append(t.outputs, output{value: out.Value, owner: owner})
		}
		return t
	}

	count := len(snap.Blocks)
	for height := 0; height < count; height++ {
		b := snap.Blocks[count-1-height]
		info := &blockInfo{height: height, hash: hex.EncodeToString(b.Hash), nonce: b.Nonce, bits: b.Bits}
		for _, tx := range b.Transactions {
			t := convert(tx, height)
			info.txs = append(info.txs, t)
			m.add(t)
		}
		m.blocks = append(m.blocks, info)
	}

	for _, tx := range snap.Mempool {
		t := convert(tx, -1)
		m.mempool = append(m.mempool, t)
		m.add(t)
	}

	for _, w := range snap.Wallets {
		m.wallets = append(m.wallets, &walletInfo{address: w.Address, name: ui.ShortString(w.Address, 5, 4), balance: w.Balance})
	}

	m.link()
	return m
}

func (m *model) related(r ref) map[ref]bool {
	set := map[ref]bool{r: true}

	addTx := func(t *txn) {
		set[txRef(t.id)] = true
		for _, in := range t.inputs {
			set[txRef(in.txid)] = true
			if in.owner != "" {
				set[walletRef(in.owner)] = true
			}
		}
		for _, out := range t.outputs {
			if out.owner != "" {
				set[walletRef(out.owner)] = true
			}
			if out.spentBy != "" {
				set[txRef(out.spentBy)] = true
			}
		}
	}

	switch r.kind {
	case refNone:
		return nil
	case refBlock:
		height := r.height
		b := m.block(height)
		if b == nil {
			return set
		}
		if height > 0 {
			set[blockRef(height-1)] = true
		}
		if height+1 < len(m.blocks) {
			set[blockRef(height+1)] = true
		}
		for _, t := range b.txs {
			addTx(t)
		}
	case refTx:
		t, ok := m.txs[r.id]
		if !ok {
			return set
		}
		addTx(t)
		if !t.pending() {
			set[blockRef(t.block)] = true
		}
	case refWallet:
		for _, t := range m.txs {
			if t.sender() == r.id {
				set[txRef(t.id)] = true
				continue
			}
			for _, out := range t.outputs {
				if out.owner == r.id {
					set[txRef(t.id)] = true
					break
				}
			}
		}
	}

	return set
}

func (m *model) unspent(address string) (count, total int) {
	for _, t := range m.txs {
		if t.pending() {
			continue
		}
		for _, out := range t.outputs {
			if out.owner == address && out.spentBy == "" {
				count++
				total += out.value
			}
		}
	}
	return count, total
}

func (m *model) explain(r ref) (string, []string) {
	switch r.kind {
	case refNone:
		return "", nil
	case refBlock:
		height := r.height
		b := m.block(height)
		if b == nil {
			break
		}
		title := fmt.Sprintf("Block #%d", height)
		var text []string
		if height == 0 {
			title += ", genesis"
			text = append(text, "The genesis block starts the chain. It has no parent, so its previous hash field is empty.")
		} else {
			text = append(text, fmt.Sprintf("Its header stores the hash of block #%d, that is the arrow pointing back. Change anything inside #%d and that stored hash stops matching, so the link breaks.", height-1, height-1))
		}

		minted, transfers, moved := 0, 0, 0
		miner := ""
		for _, t := range b.txs {
			amount, to := t.transfer()
			if t.coinbase {
				minted, miner = amount, to
				continue
			}
			transfers++
			moved += amount
		}
		line := fmt.Sprintf("Inside: a coinbase that minted %d new coins for %s", minted, m.walletName(miner))
		if transfers > 0 {
			line += fmt.Sprintf(" and %s moving %d coins", plural(transfers, "transfer"), moved)
		}
		text = append(text, line+".")
		text = append(text, fmt.Sprintf("Hash %s starts with %d zero bits. The miner found it after trying nonce %s, that search is the proof of work.", ui.ShortString(b.hash, 16, 0), b.bits, ui.Number(int64(b.nonce))))
		if on := len(m.blocks) - 1 - height; on > 0 {
			text = append(text, fmt.Sprintf("%s were mined on top of it. Rewriting this block would mean redoing all of their work too.", plural(on, "block")))
		} else {
			title += ", tip"
			text = append(text, "This is the tip, the newest block. The next mined block will point to it.")
		}
		return title, text

	case refTx:
		t, ok := m.txs[r.id]
		if !ok {
			break
		}
		if t.coinbase {
			amount, to := t.transfer()
			text := []string{
				fmt.Sprintf("The first transaction of block #%d. It has no inputs: it creates %d new coins as the reward for mining, the subsidy plus the fees of the block.", t.block, amount),
				fmt.Sprintf("Output: %d coins locked to %s. %s", amount, m.walletName(to), m.status(t, 0)),
			}
			return "Coinbase transaction", text
		}

		title := "Transfer"
		if t.pending() {
			title = "Pending transfer"
		}
		in := 0
		var sources []string
		for _, i := range t.inputs {
			in += i.value
			where := "an older block"
			if source, ok := m.txs[i.txid]; ok && !source.pending() {
				where = fmt.Sprintf("block #%d", source.block)
			}
			sources = append(sources, fmt.Sprintf("%d from %s", i.value, where))
		}
		text := []string{fmt.Sprintf("A transfer has no balance to edit. It spends %s owned by %s (%s), %d coins in total. Each blue arc starts at an output it consumes.", plural(len(t.inputs), "earlier output"), m.walletName(t.sender()), strings.Join(sources, ", "), in)}

		out := 0
		var parts []string
		for _, o := range t.outputs {
			out += o.value
			part := fmt.Sprintf("%d to %s", o.value, m.walletName(o.owner))
			if o.owner == t.sender() {
				part += " as change"
			}
			if !t.pending() && o.spentBy != "" {
				part += " (spent later)"
			}
			parts = append(parts, part)
		}
		text = append(text, "New outputs: "+strings.Join(parts, ", ")+".")
		text = append(text, fmt.Sprintf("Fee: inputs minus outputs, %d - %d = %d coins, paid to whoever mines it.", in, out, in-out))
		if t.pending() {
			text = append(text, "It waits in the mempool until a miner puts it into a block. The outputs it spends are reserved, so the same coins cannot be sent twice.")
		} else {
			text = append(text, fmt.Sprintf("Confirmed in block #%d.", t.block))
		}
		return title, text

	case refWallet:
		w := m.wallet(r.id)
		if w == nil {
			break
		}
		count, total := m.unspent(w.address)
		reserved, received, sent := 0, 0, 0
		for _, t := range m.txs {
			if t.sender() == w.address {
				sent++
			}
			got := false
			for _, o := range t.outputs {
				if o.owner != w.address {
					continue
				}
				got = true
				if spender, ok := m.txs[o.spentBy]; ok && spender.pending() && !t.pending() {
					reserved += o.value
				}
			}
			if got && t.sender() != w.address {
				received++
			}
		}
		text := []string{
			fmt.Sprintf("Balance %d coins. No balance is stored anywhere: the node adds up the unspent outputs locked to this wallet's key.", w.balance),
			fmt.Sprintf("The green lines point at %s worth %d coins that are free to spend.", plural(count, "unspent output"), total),
		}
		if reserved > 0 {
			text = append(text, fmt.Sprintf("Another %d coins sit in outputs reserved by a pending transfer. They leave the wallet once that transfer is mined.", reserved))
		}
		text = append(text, fmt.Sprintf("Received coins in %s, sent coins in %s.", plural(received, "transaction"), plural(sent, "transaction")))
		return "Wallet " + w.name, text
	}
	return "", nil
}

func (m *model) status(t *txn, index int) string {
	if index >= len(t.outputs) {
		return ""
	}
	spender, ok := m.txs[t.outputs[index].spentBy]
	switch {
	case !ok:
		return "Still unspent, so it counts toward the balance."
	case spender.pending():
		return "A pending transfer is about to spend it."
	}
	return fmt.Sprintf("Spent later by a transfer in block #%d.", spender.block)
}
