package state

import (
	"Blockchain/core"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
)

type WalletInfo struct {
	Address    string
	PubKeyHash []byte
	Balance    int
	Available  int
	Outputs    int
}

type Snapshot struct {
	Loaded     bool
	HasChain   bool
	Blocks     []*core.Block
	Wallets    []WalletInfo
	Mempool    []*core.Transaction
	MempoolMax int
	UTXOCount  int
	Supply     int
	TxCount    int
	Err        error
}

func (s *Snapshot) Height() int {
	return len(s.Blocks)
}

func (s *Snapshot) Tip() *core.Block {
	if len(s.Blocks) == 0 {
		return nil
	}
	return s.Blocks[0]
}

func (s *Snapshot) Wallet(address string) (WalletInfo, bool) {
	for _, w := range s.Wallets {
		if w.Address == address {
			return w, true
		}
	}
	return WalletInfo{}, false
}

func (s *Snapshot) Addresses() []string {
	addresses := make([]string, len(s.Wallets))
	for i, w := range s.Wallets {
		addresses[i] = w.Address
	}
	return addresses
}

type AppState struct {
	Subsidy     int
	TargetBits  int
	MempoolSize int

	mu        sync.Mutex
	bc        *core.Blockchain
	snapshot  *Snapshot
	listeners []func(*Snapshot)
	loading   atomic.Bool
	pending   atomic.Bool

	mining   atomic.Bool
	cancel   context.CancelFunc
	Progress core.MiningProgress

	miningListeners []func(bool)
	Navigate        func(screen string, arg string)
	Notify          func(title, message string, level Level)
}

type Level int

const (
	LevelInfo Level = iota
	LevelSuccess
	LevelWarning
	LevelError
)

func NewAppState() *AppState {
	return &AppState{
		Subsidy:     10,
		TargetBits:  16,
		MempoolSize: 10,
		snapshot:    &Snapshot{},
	}
}

func (s *AppState) Open() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bc != nil || !core.BlockchainExists() {
		return nil
	}

	bc, err := core.NewBlockchain()
	if err != nil {
		return err
	}
	bc.Mempool.SetMaxSize(s.MempoolSize)
	if _, err := (core.UTXOSet{Blockchain: bc}).PruneMempool(); err != nil {
		return errors.Join(err, bc.Close())
	}

	s.bc = bc
	return nil
}

func (s *AppState) Close() error {
	s.CancelMining()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bc == nil {
		return nil
	}
	err := s.bc.Close()
	s.bc = nil
	return err
}

func (s *AppState) Blockchain() *core.Blockchain {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.bc
}

func (s *AppState) CreateBlockchain(address string) error {
	if s.Blockchain() != nil {
		return errors.New("blockchain already exists")
	}

	bc, err := core.CreateBlockchain(address, s.Subsidy, s.TargetBits)
	if err != nil {
		return err
	}
	bc.Mempool.SetMaxSize(s.MempoolSize)

	if err := (core.UTXOSet{Blockchain: bc}).Reindex(); err != nil {
		return errors.Join(err, bc.Close())
	}

	s.mu.Lock()
	s.bc = bc
	s.mu.Unlock()

	return nil
}

func (s *AppState) ResetBlockchain() error {
	if s.IsMining() {
		return errors.New("stop mining before resetting the blockchain")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bc != nil {
		if err := s.bc.Close(); err != nil {
			return err
		}
		s.bc = nil
	}

	return core.RemoveBlockchain()
}

func (s *AppState) SetMempoolSize(size int) {
	s.MempoolSize = size
	if bc := s.Blockchain(); bc != nil {
		bc.Mempool.SetMaxSize(size)
	}
}

func (s *AppState) IsMining() bool {
	return s.mining.Load()
}

func (s *AppState) Busy() bool {
	return s.loading.Load() || s.mining.Load()
}

func (s *AppState) Mine(miner string, txs []*core.Transaction, done func(*core.Block, error)) {
	bc := s.Blockchain()
	if bc == nil {
		done(nil, core.ErrNoBlockchain)
		return
	}
	if !s.mining.CompareAndSwap(false, true) {
		done(nil, errors.New("mining is already running"))
		return
	}

	fees := 0
	for _, tx := range txs {
		fees += tx.Fee
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()
	s.setMining(true)

	bits := s.TargetBits
	coinbase := core.NewCoinbaseTX(miner, "", s.Subsidy+fees)
	transactions := append([]*core.Transaction{coinbase}, txs...)

	go func() {
		block, err := bc.MineBlock(ctx, transactions, bits, &s.Progress)
		if block != nil {
			err = errors.Join(err, core.UTXOSet{Blockchain: bc}.Update(block))
		}
		cancel()

		s.mining.Store(false)
		fyne.Do(func() {
			s.setMining(false)
			done(block, err)
		})
		s.Refresh()
	}()
}

func (s *AppState) CancelMining() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (s *AppState) setMining(active bool) {
	for _, fn := range s.miningListeners {
		fn(active)
	}
}

func (s *AppState) OnMiningChanged(fn func(bool)) {
	s.miningListeners = append(s.miningListeners, fn)
}

func (s *AppState) Snapshot() *Snapshot {
	return s.snapshot
}

func (s *AppState) Subscribe(fn func(*Snapshot)) {
	s.listeners = append(s.listeners, fn)
	fn(s.snapshot)
}

func (s *AppState) Refresh() {
	if !s.loading.CompareAndSwap(false, true) {
		s.pending.Store(true)
		return
	}

	go func() {
		for {
			snap := s.load()
			fyne.Do(func() {
				s.snapshot = snap
				for _, fn := range s.listeners {
					fn(snap)
				}
			})

			if !s.pending.Swap(false) {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		s.loading.Store(false)

		if s.pending.Swap(false) {
			s.Refresh()
		}
	}()
}

func (s *AppState) load() *Snapshot {
	snap := &Snapshot{Loaded: true, MempoolMax: s.MempoolSize}

	var addresses []string
	if wallets, err := core.NewWallets(); err != nil {
		snap.Err = err
	} else {
		addresses = wallets.GetAddresses()
	}

	bc := s.Blockchain()
	var utxo core.UTXOSet
	if bc != nil {
		snap.HasChain = true
		snap.Blocks = bc.Blocks()
		snap.Mempool = bc.Mempool.GetTransactions()
		snap.MempoolMax = bc.Mempool.MaxSize()
		for _, block := range snap.Blocks {
			snap.TxCount += len(block.Transactions)
		}

		utxo = core.UTXOSet{Blockchain: bc}
		count, err := utxo.CountTransactions()
		snap.UTXOCount = count
		snap.Err = errors.Join(snap.Err, err)
		balances, err := utxo.Balances()
		snap.Err = errors.Join(snap.Err, err)
		for _, value := range balances {
			snap.Supply += value
		}
	}

	for _, address := range addresses {
		info := WalletInfo{Address: address, PubKeyHash: core.AddressToPubKeyHash(address)}
		if bc != nil {
			outputs, err := utxo.FindUTXO(info.PubKeyHash)
			snap.Err = errors.Join(snap.Err, err)
			for _, out := range outputs {
				info.Balance += out.Value
				info.Outputs++
				if !out.Pending {
					info.Available += out.Value
				}
			}
		}
		snap.Wallets = append(snap.Wallets, info)
	}

	return snap
}

func (s *AppState) CreateWallet() (string, error) {
	wallets, err := core.NewWallets()
	if err != nil {
		return "", err
	}

	address := wallets.CreateWallet()
	if err := wallets.SaveToFile(); err != nil {
		return "", err
	}

	return address, nil
}

func (s *AppState) Send(from, to string, amount, fee int) (*core.Transaction, error) {
	bc := s.Blockchain()
	if bc == nil {
		return nil, core.ErrNoBlockchain
	}

	return core.NewUTXOTransaction(from, to, amount, fee, &core.UTXOSet{Blockchain: bc})
}

func (s *AppState) Reindex() (int, error) {
	bc := s.Blockchain()
	if bc == nil {
		return 0, core.ErrNoBlockchain
	}

	utxo := core.UTXOSet{Blockchain: bc}
	if err := utxo.Reindex(); err != nil {
		return 0, err
	}

	return utxo.CountTransactions()
}

func (s *AppState) Go(screen, arg string) {
	if s.Navigate != nil {
		s.Navigate(screen, arg)
	}
}

func (s *AppState) Toast(title, message string, level Level) {
	if s.Notify != nil {
		s.Notify(title, message, level)
	}
}
