package transfermonitor

import (
	"context"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"cosmossdk.io/math"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	ethereumrpc "github.com/ethereum/go-ethereum/rpc"
	dbtypes "github.com/skip-mev/go-fast-solver/db"
	"github.com/skip-mev/go-fast-solver/db/gen/db"
	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/contracts/fast_transfer_gateway"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"github.com/skip-mev/go-fast-solver/shared/tmrpc"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	maxBlocksProcessedPerIteration = 100000
	destinationChainID             = "osmosis-1"
	orderSubmittedEventSignature   = "0x59f858504f8d8ad967dd7453df850e265270474e364b7e2fbd3333e06efdbfc0"
)

type MonitorDBQueries interface {
	InsertTransferMonitorMetadata(ctx context.Context, arg db.InsertTransferMonitorMetadataParams) (db.TransferMonitorMetadatum, error)
	GetTransferMonitorMetadata(ctx context.Context, chainID string) (db.TransferMonitorMetadatum, error)
	InsertOrder(ctx context.Context, arg db.InsertOrderParams) (db.Order, error)
}

type TransferMonitor struct {
	db            MonitorDBQueries
	clients       map[string]*ethclient.Client
	wsClients     map[string]*ethereumrpc.Client
	tmRPCManager  tmrpc.TendermintRPCClientManager
	quickStart    bool
	didQuickStart map[string]bool
	ticker        *time.Ticker
	useWebSocket  bool
}

func NewTransferMonitor(db MonitorDBQueries, quickStart bool, pollInterval *time.Duration, useWebSocket bool) *TransferMonitor {
	if pollInterval == nil {
		pollInterval = &[]time.Duration{5 * time.Second}[0]
	}
	return &TransferMonitor{
		db:            db,
		clients:       make(map[string]*ethclient.Client),
		wsClients:     make(map[string]*ethereumrpc.Client),
		tmRPCManager:  tmrpc.NewTendermintRPCClientManager(),
		quickStart:    quickStart,
		didQuickStart: make(map[string]bool),
		ticker:        time.NewTicker(*pollInterval),
		useWebSocket:  useWebSocket,
	}
}

func (t *TransferMonitor) Start(ctx context.Context) error {
	lmt.Logger(ctx).Info("Starting transfer monitor")
	var chains []config.ChainConfig
	evmChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_EVM)
	if err != nil {
		return fmt.Errorf("error getting EVM chains: %w", err)
	}
	for _, chain := range evmChains {
		if chain.FastTransferContractAddress != "" {
			chains = append(chains, chain)
		}
	}

	if t.useWebSocket {
		eg, ctx := errgroup.WithContext(ctx)
		for _, chain := range chains {
			chain := chain
			eg.Go(func() error {
				return t.startWebSocketMonitor(ctx, chain)
			})
		}
		go func() {
			if err := eg.Wait(); err != nil {
				lmt.Logger(ctx).Error("WebSocket monitor error", zap.Error(err))
			}
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.ticker.C:
			for _, chain := range chains {
				chainID, err := getChainID(chain)
				if err != nil {
					lmt.Logger(ctx).Error("Error getting chain id", zap.Error(err))
					continue
				}
				var startBlockHeight uint64
				transferMonitorMetadata, err := t.db.GetTransferMonitorMetadata(ctx, chainID)
				if err != nil && !strings.Contains(err.Error(), "no rows in result set") {

					lmt.Logger(ctx).Error("Error getting transfer monitor metadata", zap.Error(err))
					continue
				} else if err == nil {
					startBlockHeight = uint64(transferMonitorMetadata.HeightLastSeen)
				}

				if t.quickStart && !t.didQuickStart[chainID] {
					latestBlock, err := t.getLatestBlockHeight(ctx, chain)
					if err != nil {
						lmt.Logger(ctx).Error("Error getting latest block height", zap.Error(err))
						continue
					}
					quickStartBlockHeight := latestBlock - chain.QuickStartNumBlocksBack
					if quickStartBlockHeight > startBlockHeight {
						startBlockHeight = quickStartBlockHeight
					}

					// Mark this chain as having been quickstarted
					t.didQuickStart[chainID] = true
				}

				lmt.Logger(ctx).Debug("Processing new blocks", zap.String("chain_id", chainID), zap.Uint64("height", startBlockHeight))
				var orders []Order
				var endBlockHeight uint64
				var fastTransferGatewayContractAddress string
				switch chain.Type {
				case config.ChainType_EVM:
					fastTransferGatewayContractAddress = chain.FastTransferContractAddress
					orders, endBlockHeight, err = t.findNewTransferIntentsOnEVMChain(ctx, chain, startBlockHeight)
					if err != nil {
						lmt.Logger(ctx).Error("Error finding burn transactions", zap.Error(err))
						continue
					}
				default:
					lmt.Logger(ctx).Error("Unsupported chain type", zap.String("chain_type", string(chain.Type)))
					continue
				}

				errorInsertingOrder := false
				if len(orders) > 0 {
					lmt.Logger(ctx).Info("Found burn transactions", zap.Int("count", len(orders)), zap.String("chain_id", chainID))
					for _, order := range orders {
						toInsert := db.InsertOrderParams{
							SourceChainID:                     order.ChainID,
							DestinationChainID:                order.DestinationChainID,
							SourceChainGatewayContractAddress: fastTransferGatewayContractAddress,
							Sender:                            order.OrderEvent.Sender[:],
							Recipient:                         order.OrderEvent.Recipient[:],
							AmountIn:                          order.OrderEvent.AmountIn.String(),
							AmountOut:                         order.OrderEvent.AmountOut.String(),
							Nonce:                             int64(order.OrderEvent.Nonce),
							OrderCreationTx:                   order.TxHash,
							OrderCreationTxBlockHeight:        int64(order.TxBlockHeight),
							OrderID:                           order.OrderID,
							OrderStatus:                       dbtypes.OrderStatusPending,
							TimeoutTimestamp:                  time.Unix(order.TimeoutTimestamp, 0).UTC(),
						}
						if len(order.OrderEvent.Data) > 0 {
							toInsert.Data = sql.NullString{String: hex.EncodeToString(order.OrderEvent.Data), Valid: true}
						}

						_, err := t.db.InsertOrder(ctx, toInsert)
						if err != nil && !strings.Contains(err.Error(), "sql: no rows in result set") {

							lmt.Logger(ctx).Error("Error inserting order", zap.Error(err))
							errorInsertingOrder = true
							break
						}
						metrics.FromContext(ctx).IncFillOrderStatusChange(order.ChainID, order.DestinationChainID, dbtypes.OrderStatusPending)
					}
				}
				lmt.Logger(ctx).Debug("num orders found while processing blocks", zap.Int("numOrders", len(orders)))
				if errorInsertingOrder {
					continue
				}

				_, err = t.db.InsertTransferMonitorMetadata(ctx, db.InsertTransferMonitorMetadataParams{
					ChainID:        chainID,
					HeightLastSeen: int64(endBlockHeight),
				})
				if err != nil {

					lmt.Logger(ctx).Error("Error inserting transfer monitor metadata", zap.Error(err))
					continue
				}
			}
		}
	}
}

func (t *TransferMonitor) findNewTransferIntentsOnEVMChain(ctx context.Context, chain config.ChainConfig, startBlockHeight uint64) ([]Order, uint64, error) {
	client, err := t.getClient(ctx, chain.ChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Error getting client", zap.Error(err))
		return nil, 0, err
	}

	header, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		lmt.Logger(ctx).Error("Error fetching latest block", zap.Error(err))
		return nil, 0, err
	}

	endBlockHeight := math.Min(header.Number.Uint64(), startBlockHeight+maxBlocksProcessedPerIteration)

	fastTransferContractAddress := chain.FastTransferContractAddress
	fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(
		common.HexToAddress(fastTransferContractAddress),
		client,
	)
	if err != nil {
		lmt.Logger(ctx).Error("Error creating MessageTransmitter object", zap.Error(err))
		return nil, 0, err
	}

	orders, err := t.findTransferIntents(ctx, startBlockHeight, endBlockHeight, fastTransferGateway, client, chain.Environment, chain.ChainID)
	if err != nil {
		lmt.Logger(ctx).Error("Error finding burn transactions", zap.Error(err))
		return nil, 0, err
	}

	if orders != nil {
		orderCounts := make(map[string]int)
		for _, order := range orders {
			key := fmt.Sprintf("%s->%s", order.ChainID, order.DestinationChainID)
			orderCounts[key]++
		}

		for chainPair, numOfOrders := range orderCounts {
			lmt.Logger(ctx).Info("Fast transfer orders found",
				zap.String("source->destination", chainPair),
				zap.Int("numOfOrders", numOfOrders))
		}
	}
	return orders, endBlockHeight, nil
}

func (t *TransferMonitor) getClient(ctx context.Context, chainID string) (*ethclient.Client, error) {
	if _, ok := t.clients[chainID]; !ok {
		rpc, err := config.GetConfigReader(ctx).GetRPCEndpoint(chainID)
		if err != nil {
			return nil, err
		}

		basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(chainID)
		if err != nil {
			return nil, err
		}

		conn, err := ethereumrpc.DialContext(ctx, rpc)
		if err != nil {
			return nil, err
		}
		if basicAuth != nil {
			conn.SetHeader("Authorization", fmt.Sprintf("Basic %s", *basicAuth))
		}

		client := ethclient.NewClient(conn)
		t.clients[chainID] = client
	}

	return t.clients[chainID], nil
}

type Order struct {
	TxHash             string                                  `json:"tx_hash"`
	TxBlockHeight      uint64                                  `json:"tx_block_height"`
	ChainID            string                                  `json:"chain_id"`
	DestinationChainID string                                  `json:"destination_chain_id"`
	ChainEnvironment   config.ChainEnvironment                 `json:"chain_environment"`
	OrderEvent         fast_transfer_gateway.FastTransferOrder `json:"order_event"`
	OrderID            string                                  `json:"order_id"`
	TimeoutTimestamp   int64                                   `json:"timeout_timestamp"`
}

func (t *TransferMonitor) findTransferIntents(
	ctx context.Context,
	startBlock,
	endBlock uint64,
	fastTransferGateway *fast_transfer_gateway.FastTransferGateway,
	client *ethclient.Client,
	chainEnvironment config.ChainEnvironment,
	chainID string,
) (orders []Order, err error) {
	offset := uint64(0)
	limit := uint64(1000)
	m := sync.Mutex{}
	eg, egctx := errgroup.WithContext(ctx)
	eg.SetLimit(20)
OuterLoop:
	for {
		select {
		case <-egctx.Done():
			return nil, nil
		default:
			start := startBlock + offset
			end := startBlock + offset + limit
			if start > endBlock {
				break OuterLoop
			}
			if end > endBlock {
				end = endBlock
			}
			eg.Go(func() error {
				var iter *fast_transfer_gateway.FastTransferGatewayOrderSubmittedIterator
				for i := 0; i < 5; i++ {
					iter, err = fastTransferGateway.FilterOrderSubmitted(&bind.FilterOpts{
						Context: ctx,
						Start:   start,
						End:     &[]uint64{end}[0],
					}, nil)
					if err != nil && i == 4 { // TODO dont retry on context cancellation
						return err
					}
					if err == nil {
						break
					}
					time.Sleep(1 * time.Second)
				}
				if iter == nil {
					return nil
				}

				for iter.Next() {
					m.Lock()
					orderData := fast_transfer_gateway.DecodeOrder(iter.Event.Order)
					orders = append(orders, Order{
						TxHash:             iter.Event.Raw.TxHash.Hex(),
						TxBlockHeight:      iter.Event.Raw.BlockNumber,
						ChainID:            chainID,
						DestinationChainID: destinationChainID,
						OrderEvent:         orderData,
						ChainEnvironment:   chainEnvironment,
						OrderID:            hex.EncodeToString(iter.Event.OrderID[:]),
						TimeoutTimestamp:   int64(orderData.TimeoutTimestamp),
					})
					m.Unlock()
				}

				if err := iter.Error(); err != nil {
					return err
				}

				return nil
			})
			offset += limit
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err := eg.Wait(); err != nil {
		lmt.Logger(egctx).Error("Error encountered while searching for transfers", zap.Error(err))
		return nil, err
	}
	return orders, nil
}

func getChainID(chain config.ChainConfig) (string, error) {
	switch chain.Type {
	case config.ChainType_COSMOS:
		return chain.ChainID, nil
	case config.ChainType_EVM:
		return chain.ChainID, nil
	default:
		return "", fmt.Errorf("unknown chain type")
	}
}

func (t *TransferMonitor) getLatestBlockHeight(ctx context.Context, chain config.ChainConfig) (uint64, error) {
	switch chain.Type {
	case config.ChainType_EVM:
		client, err := t.getClient(ctx, chain.ChainID)
		if err != nil {
			return 0, err
		}
		header, err := client.HeaderByNumber(ctx, nil)
		if err != nil {
			return 0, err
		}
		return header.Number.Uint64(), nil
	case config.ChainType_COSMOS:
		client, err := t.tmRPCManager.GetClient(ctx, chain.ChainID)
		if err != nil {
			return 0, err
		}
		status, err := client.Status(ctx)
		if err != nil {
			return 0, err
		}
		return uint64(status.SyncInfo.LatestBlockHeight), nil
	default:
		return 0, fmt.Errorf("unsupported chain type: %s", chain.Type)
	}
}

func (t *TransferMonitor) startWebSocketMonitor(ctx context.Context, chain config.ChainConfig) error {
	chainID, err := getChainID(chain)
	if err != nil {
		return fmt.Errorf("error getting chain id: %w", err)
	}

	wsEndpoint, err := config.GetConfigReader(ctx).GetWSEndpoint(chain.ChainID)
	if err != nil {
		return fmt.Errorf("error getting websocket endpoint: %w", err)
	}

	lmt.Logger(ctx).Info("Initializing WebSocket connection",
		zap.String("chain_id", chainID),
		zap.String("ws_endpoint", wsEndpoint))

	wsClient, err := t.getWebSocketClient(ctx, chain)
	if err != nil {
		if strings.Contains(err.Error(), "dial tcp") ||
			strings.Contains(err.Error(), "not supported") {
			// Add metrics alongside existing logging
			metrics.FromContext(ctx).SetConnectionType(chainID, "transfer_monitor", metrics.ConnectionTypeRPC)
			metrics.FromContext(ctx).RecordSubscriptionError(chainID, "transfer_monitor", "connection_failed")
			metrics.FromContext(ctx).RecordConnectionSwitch(chainID, "transfer_monitor",
				metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)

			lmt.Logger(ctx).Info("WebSocket connection failed, falling back to RPC polling",
				zap.String("chain_id", chainID),
				zap.String("chain_name", chain.ChainName),
				zap.Error(err))

			return nil
		}
		return fmt.Errorf("error getting websocket client: %w", err)
	}

	// Create client for contract interaction
	ethClient := ethclient.NewClient(wsClient)

	// Create contract instance once outside the event loop
	contractAddress := common.HexToAddress(chain.FastTransferContractAddress)
	fastTransferGateway, err := fast_transfer_gateway.NewFastTransferGateway(
		contractAddress,
		ethClient,
	)
	if err != nil {
		lmt.Logger(ctx).Error("Error creating contract instance",
			zap.String("chain_id", chainID),
			zap.Error(err))
		wsClient.Close()
		return fmt.Errorf("error creating contract instance: %w", err)
	}

	// Subscribe to new heads to track latest block height
	headers := make(chan *types.Header)
	headsSub, err := ethClient.SubscribeNewHead(ctx, headers)
	if err != nil {
		lmt.Logger(ctx).Error("Error subscribing to new block headers",
			zap.String("chain_id", chainID),
			zap.Error(err))
		wsClient.Close()
		return fmt.Errorf("error subscribing to block headers: %w", err)
	}

	// Get initial block height
	var latestBlockHeight uint64
	var lastProcessedHeight uint64
	transferMonitorMetadata, err := t.db.GetTransferMonitorMetadata(ctx, chainID)
	if err == nil {
		lastProcessedHeight = uint64(transferMonitorMetadata.HeightLastSeen)
	} else if !strings.Contains(err.Error(), "no rows in result set") {
		lmt.Logger(ctx).Error("Error retrieving transfer monitor metadata",
			zap.String("chain_id", chainID),
			zap.Error(err))
	}

	// Subscribe to contract events
	query := ethereum.FilterQuery{
		Addresses: []common.Address{contractAddress},
		Topics: [][]common.Hash{{
			common.HexToHash(orderSubmittedEventSignature),
		}},
	}

	// Subscribe to event logs
	logs := make(chan types.Log)
	eventsSub, err := ethClient.SubscribeFilterLogs(ctx, query, logs)
	if err != nil {
		// Add metrics alongside existing logging
		metrics.FromContext(ctx).SetConnectionType(chainID, "transfer_monitor", metrics.ConnectionTypeRPC)
		metrics.FromContext(ctx).RecordSubscriptionError(chainID, "transfer_monitor", "subscription_failed")
		metrics.FromContext(ctx).RecordConnectionSwitch(chainID, "transfer_monitor",
			metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)

		lmt.Logger(ctx).Info("WebSocket subscription failed, falling back to RPC polling",
			zap.String("chain_id", chainID),
			zap.String("chain_name", chain.ChainName),
			zap.Error(err))
		headsSub.Unsubscribe()
		wsClient.Close()
		return nil
	}

	// Add metric for successful connection
	metrics.FromContext(ctx).SetConnectionType(chainID, "transfer_monitor", metrics.ConnectionTypeWebSocket)

	lmt.Logger(ctx).Info("Successfully established WebSocket connections",
		zap.String("chain_id", chainID),
		zap.String("chain_name", chain.ChainName),
		zap.String("contract_address", chain.FastTransferContractAddress))

	for {
		select {
		case err := <-headsSub.Err():
			metrics.FromContext(ctx).RecordSubscriptionError(chainID, "transfer_monitor", "headers_subscription_error")
			lmt.Logger(ctx).Error("WebSocket headers subscription error",
				zap.String("chain_id", chainID),
				zap.String("chain_name", chain.ChainName),
				zap.Error(err))
			eventsSub.Unsubscribe()
			wsClient.Close()
			return fmt.Errorf("headers subscription error for chain %s: %w", chainID, err)

		case header := <-headers:
			// Update latest block height from the websocket data
			latestBlockHeight = header.Number.Uint64()
			metrics.FromContext(ctx).IncrementBlocksReceived(chainID, "transfer_monitor")

			// Periodically update the metadata with the latest processed height
			if latestBlockHeight > lastProcessedHeight {
				// Only update periodically (e.g., every 100 blocks or after processing events)
				if latestBlockHeight-lastProcessedHeight >= 100 {
					_, err = t.db.InsertTransferMonitorMetadata(ctx, db.InsertTransferMonitorMetadataParams{
						ChainID:        chainID,
						HeightLastSeen: int64(latestBlockHeight),
					})
					if err != nil {
						lmt.Logger(ctx).Error("Error updating transfer monitor metadata",
							zap.String("chain_id", chainID),
							zap.Uint64("height", latestBlockHeight),
							zap.Error(err))
					} else {
						lastProcessedHeight = latestBlockHeight
						lmt.Logger(ctx).Debug("Updated last processed height from new block headers",
							zap.String("chain_id", chainID),
							zap.Uint64("height", latestBlockHeight))
					}
				}
			}

		case err := <-eventsSub.Err():
			// Add metrics alongside existing logging
			metrics.FromContext(ctx).RecordSubscriptionError(chainID, "transfer_monitor", "events_subscription_error")
			metrics.FromContext(ctx).RecordConnectionSwitch(chainID, "transfer_monitor",
				metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)

			lmt.Logger(ctx).Error("WebSocket events subscription error",
				zap.String("chain_id", chainID),
				zap.String("chain_name", chain.ChainName),
				zap.Error(err))
			headsSub.Unsubscribe()
			wsClient.Close()
			return fmt.Errorf("events subscription error for chain %s: %w", chainID, err)

		case vLog := <-logs:
			// Process the log event
			metrics.FromContext(ctx).IncrementBlocksReceived(chainID, "transfer_monitor")

			// Use the pre-created contract instance instead of creating a new one for each event
			event, err := fastTransferGateway.ParseOrderSubmitted(vLog)
			if err != nil {
				lmt.Logger(ctx).Error("Error parsing OrderSubmitted event",
					zap.String("chain_id", chainID),
					zap.Error(err))
				continue
			}

			// Create order from event data
			orderData := fast_transfer_gateway.DecodeOrder(event.Order)
			order := Order{
				TxHash:             vLog.TxHash.Hex(),
				TxBlockHeight:      vLog.BlockNumber,
				ChainID:            chainID,
				DestinationChainID: destinationChainID,
				ChainEnvironment:   chain.Environment,
				OrderEvent:         orderData,
				OrderID:            hex.EncodeToString(event.OrderID[:]),
				TimeoutTimestamp:   int64(orderData.TimeoutTimestamp),
			}

			// Insert the order into the database
			toInsert := db.InsertOrderParams{
				SourceChainID:                     order.ChainID,
				DestinationChainID:                order.DestinationChainID,
				SourceChainGatewayContractAddress: chain.FastTransferContractAddress,
				Sender:                            order.OrderEvent.Sender[:],
				Recipient:                         order.OrderEvent.Recipient[:],
				AmountIn:                          order.OrderEvent.AmountIn.String(),
				AmountOut:                         order.OrderEvent.AmountOut.String(),
				Nonce:                             int64(order.OrderEvent.Nonce),
				OrderCreationTx:                   order.TxHash,
				OrderCreationTxBlockHeight:        int64(order.TxBlockHeight),
				OrderID:                           order.OrderID,
				OrderStatus:                       dbtypes.OrderStatusPending,
				TimeoutTimestamp:                  time.Unix(order.TimeoutTimestamp, 0).UTC(),
			}

			if len(order.OrderEvent.Data) > 0 {
				toInsert.Data = sql.NullString{String: hex.EncodeToString(order.OrderEvent.Data), Valid: true}
			}

			_, err = t.db.InsertOrder(ctx, toInsert)
			if err != nil {
				if strings.Contains(err.Error(), "sql: no rows in result set") {
					// This is expected in some cases, just continue
					continue
				}

				if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "UNIQUE constraint") {
					// Log at debug level for duplicates
					lmt.Logger(ctx).Debug("Skipping duplicate order",
						zap.String("order_id", order.OrderID),
						zap.String("chain_id", chainID))
					continue
				}

				// Real error
				lmt.Logger(ctx).Error("Error inserting order",
					zap.String("order_id", order.OrderID),
					zap.String("chain_id", chainID),
					zap.Error(err))
				continue
			}

			// Also update the last processed height after processing an event
			if vLog.BlockNumber > lastProcessedHeight {
				_, err = t.db.InsertTransferMonitorMetadata(ctx, db.InsertTransferMonitorMetadataParams{
					ChainID:        chainID,
					HeightLastSeen: int64(vLog.BlockNumber),
				})
				if err != nil {
					lmt.Logger(ctx).Error("Error updating transfer monitor metadata after event",
						zap.String("chain_id", chainID),
						zap.Uint64("height", vLog.BlockNumber),
						zap.Error(err))
				} else {
					lastProcessedHeight = vLog.BlockNumber
				}
			}

			metrics.FromContext(ctx).IncFillOrderStatusChange(order.ChainID, order.DestinationChainID, dbtypes.OrderStatusPending)

			lmt.Logger(ctx).Info("Processed order from WebSocket subscription",
				zap.String("chain_id", chainID),
				zap.String("order_id", order.OrderID),
				zap.Uint64("block_number", vLog.BlockNumber))

		case <-ctx.Done():
			headsSub.Unsubscribe()
			eventsSub.Unsubscribe()
			wsClient.Close()
			lmt.Logger(ctx).Info("WebSocket subscriptions unsubscribed and connection closed",
				zap.String("chain_id", chain.ChainID),
				zap.String("chain_name", chain.ChainName))
			return nil
		}
	}
}

func (t *TransferMonitor) getWebSocketClient(ctx context.Context, chain config.ChainConfig) (*ethereumrpc.Client, error) {
	wsEndpoint, err := config.GetConfigReader(ctx).GetWSEndpoint(chain.ChainID)
	if err != nil {
		return nil, err
	}

	lmt.Logger(ctx).Info("Initializing WebSocket connection",
		zap.String("chain_id", chain.ChainID),
		zap.String("ws_endpoint", wsEndpoint))

	if client, ok := t.wsClients[chain.ChainID]; ok {
		return client, nil
	}

	basicAuth, err := config.GetConfigReader(ctx).GetBasicAuth(chain.ChainID)
	if err != nil {
		return nil, err
	}

	client, err := ethereumrpc.DialContext(ctx, wsEndpoint)
	if err != nil {
		return nil, err
	}

	if basicAuth != nil {
		client.SetHeader("Authorization", fmt.Sprintf("Basic %s", *basicAuth))
	}

	t.wsClients[chain.ChainID] = client
	return client, nil
}
