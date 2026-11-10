# Transactions

A transaction moves value from one owner to another. There are no accounts and no balances stored anywhere. A transaction simply consumes some earlier outputs and creates new ones.

{{image:tx_spending.png|Each input spends a previous output, each output waits as a UTXO (Bitcoin Developer Guide)}}

## Inputs and outputs

```go
type Transaction struct {
    ID   []byte
    Vin  []TXInput
    Vout []TXOutput
    Fee  int
}

type TXInput struct {
    Txid      []byte
    Vout      int
    Signature []byte
    PubKey    []byte
}

type TXOutput struct {
    Value      int
    PubKeyHash []byte
}
```

- An **output** stores an amount and locks it to a public key hash. Only the owner of the matching private key can spend it.
- An **input** references an existing output by transaction ID (`Txid`) and index (`Vout`). It also carries the spender's public key and a signature.
- An output is spent completely. If you own an output of 50 and want to send 12, the transaction has two outputs: 12 to the recipient and the change back to you.
- The **Fee** is the difference that the sender leaves for the miner. Here it is stored explicitly to make it easy to show in the interface.

## Which comes first

Every input spends an output, so outputs must exist before the inputs that spend them. The chicken and egg problem is solved by the coinbase: it has no real input and creates the very first outputs. Every coin in circulation can be traced back through a chain of inputs to some coinbase transaction.

## Coinbase transactions

The first transaction in every block creates new coins and has no real inputs:

```go
func NewCoinbaseTX(to, data string, subsidy int) *Transaction {
    if data == "" {
        randData := make([]byte, 20)
        _, err := rand.Read(randData)
        if err != nil {
            log.Panic(err)
        }
        data = fmt.Sprintf("%x", randData)
    }

    txin := TXInput{[]byte{}, -1, nil, []byte(data)}
    txout := NewTXOutput(subsidy, to)
    tx := Transaction{nil, []TXInput{txin}, []TXOutput{*txout}, 0}
    tx.ID = tx.Hash()

    return &tx
}
```

Its single input points to nothing (empty `Txid`, `Vout` of -1) and the `PubKey` field is used to store arbitrary data. The random data makes two coinbase transactions to the same address have different IDs. When the miner includes transactions from the mempool, the coinbase value is the subsidy plus all their fees.

## Building a transfer

`NewUTXOTransaction` follows these steps:

1. Load the sender wallet from `wallet.dat` and refuse unknown addresses.
2. Refuse the transfer if the sender already has a pending transaction in the mempool.
3. Collect enough unspent outputs to cover `amount + fee`.
4. Create one input per collected output and the outputs for the recipient and the change.
5. Sign every input and add the transaction to the mempool.

```go
acc, validOutputs, err := UTXOSet.FindSpendableOutputs(pubKeyHash, amount+fee)
if err != nil {
    return nil, err
}

if acc < amount+fee {
    return nil, fmt.Errorf("%w: need %d, available %d", ErrNotEnoughFunds, amount+fee, acc)
}

outputs = append(outputs, *NewTXOutput(amount, to))
if acc > amount+fee {
    outputs = append(outputs, *NewTXOutput(acc-amount-fee, from))
}
```

## Signing

{{image:tx_signing.png|The data covered by an input signature (Bitcoin Developer Guide)}}

The signature has to cover the data that matters: which outputs are spent, who they belong to and where the coins go. Every input is signed separately over a **trimmed copy** of the transaction:

```go
for inID, vin := range txCopy.Vin {
    prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
    txCopy.Vin[inID].Signature = nil
    txCopy.Vin[inID].PubKey = prevTx.Vout[vin.Vout].PubKeyHash
    txCopy.ID = txCopy.Hash()
    txCopy.Vin[inID].PubKey = nil

    r, s, err := ecdsa.Sign(rand.Reader, &privKey, txCopy.ID)
    if err != nil {
        return err
    }

    signature := make([]byte, 2*coordinateSize)
    r.FillBytes(signature[:coordinateSize])
    s.FillBytes(signature[coordinateSize:])

    tx.Vin[inID].Signature = signature
}
```

In the copy, every signature and public key is removed. Only the input being signed gets the `PubKeyHash` of the output it spends. The hash of that copy is signed with ECDSA on the P-256 curve.

`r` and `s` are written into fixed 32 byte halves. `big.Int.Bytes` drops leading zero bytes, so without padding roughly one signature in a hundred would be shorter and the verifier would split it in the wrong place.

## Verification

A node never trusts a transaction. `Verify` rebuilds the same trimmed copy and checks two things for every input:

```go
prevTx := prevTXs[hex.EncodeToString(vin.Txid)]
if len(vin.Signature) == 0 || !vin.UsesKey(prevTx.Vout[vin.Vout].PubKeyHash) {
    return false
}
```

1. The public key in the input hashes to the `PubKeyHash` of the output being spent. Without this check anybody could spend someone else's coins with their own valid signature.
2. `ecdsa.Verify` accepts the signature for that public key.

`MineBlock` verifies every transaction before mining, so an invalid transaction can never enter a block.

## Scripts in Bitcoin, fields here

Bitcoin does not store a public key hash and a signature as plain fields. Each output carries a small program, the locking script `scriptPubKey`, and each input carries an unlocking script `scriptSig`. For the common Pay to Public Key Hash type (P2PKH) they look like this:

```text
scriptPubKey: OP_DUP OP_HASH160 <pubKeyHash> OP_EQUALVERIFY OP_CHECKSIG
scriptSig:    <signature> <pubKey>
```

A node runs `scriptSig` and then `scriptPubKey` on one stack. The public key is duplicated, hashed with SHA-256 and RIPEMD-160 and compared with `pubKeyHash`. Then `OP_CHECKSIG` checks the signature against the public key. If the stack ends with true, the output may be spent.

This project keeps the same logic without a script engine. `TXOutput.PubKeyHash` plays the role of the locking script, `TXInput.Signature` and `TXInput.PubKey` play the role of the unlocking script, and `Verify` performs the two checks that `OP_EQUALVERIFY` and `OP_CHECKSIG` do. Other output types, like multisignature outputs, would need real scripts in place of these fields.

## Lifecycle

1. **Input selection.** Spendable outputs are collected from the UTXO set.
2. **Signing.** The private key signs every input.
3. **Mempool.** The transaction waits in `mempool.dat`.
4. **Mining.** A miner selects it on the Mining screen and earns its fee.
5. **Confirmation.** The block is stored and the UTXO set is updated.

## Try it

Send coins on the Transactions screen, then open the Mining screen, tick the pending transaction and mine. Open the new block in the explorer: the transfer has inputs pointing to older transactions and usually two outputs, one of them the change.
