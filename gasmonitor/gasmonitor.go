package gasmonitor

import (
	"context"
	"fmt"
	"github.com/skip-mev/go-fast-solver/shared/bridges/cctp"
	"github.com/skip-mev/go-fast-solver/shared/clientmanager"
	"github.com/skip-mev/go-fast-solver/shared/metrics"
	"go.uber.org/zap"
	"strings"
	"time"

	"github.com/skip-mev/go-fast-solver/shared/config"
	"github.com/skip-mev/go-fast-solver/shared/lmt"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/core/types"
)

type GasMonitor struct {
	clientManager *clientmanager.ClientManager
	useWebSocket  bool
	pollInterval  time.Duration
}

func NewGasMonitor(clientManager *clientmanager.ClientManager, useWebSocket bool, pollInterval time.Duration) *GasMonitor {
	return &GasMonitor{
		clientManager: clientManager,
		useWebSocket:  useWebSocket,
		pollInterval:  pollInterval,
	}
}

func (gm *GasMonitor) Start(ctx context.Context) error {
	lmt.Logger(ctx).Info("Starting gas monitor",
		zap.Bool("use_websocket", gm.useWebSocket),
		zap.Duration("poll_interval", gm.pollInterval))

	var chains []config.ChainConfig
	evmChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_EVM)
	if err != nil {
		return fmt.Errorf("error getting EVM chains: %w", err)
	}
	cosmosChains, err := config.GetConfigReader(ctx).GetAllChainConfigsOfType(config.ChainType_COSMOS)
	if err != nil {
		return fmt.Errorf("error getting cosmos chains: %w", err)
	}
	chains = append(chains, evmChains...)
	chains = append(chains, cosmosChains...)

	if gm.useWebSocket {
		return gm.startWebSocketMonitor(ctx, chains)
	}
	return gm.startRPCMonitor(ctx, chains)
}

func (gm *GasMonitor) startWebSocketMonitor(ctx context.Context, chains []config.ChainConfig) error {
	eg, ctx := errgroup.WithContext(ctx)
	for _, chain := range chains {
		chain := chain // capture for goroutine
		eg.Go(func() error {
			return gm.monitorChainWebSocket(ctx, chain)
		})
	}
	return eg.Wait()
}

func (gm *GasMonitor) startRPCMonitor(ctx context.Context, chains []config.ChainConfig) error {
	ticker := time.NewTicker(gm.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			for _, chain := range chains {
				client, err := gm.clientManager.GetClient(ctx, chain.ChainID)
				if err != nil {
					return err
				}
				if err := monitorGasBalance(ctx, chain.ChainID, client); err != nil {
					lmt.Logger(ctx).Error("failed to monitor gas balance",
						zap.String("chain_id", chain.ChainID),
						zap.Error(err))
				}
			}
		}
	}
}

func (gm *GasMonitor) monitorChainWebSocket(ctx context.Context, chain config.ChainConfig) error {
	// Skip non-EVM chains
	if chain.Type != config.ChainType_EVM {
		metrics.FromContext(ctx).SetConnectionType(chain.ChainID, "gas_monitor", metrics.ConnectionTypeRPC)
		lmt.Logger(ctx).Info("Skipping WebSocket for non-EVM chain",
			zap.String("chain_id", chain.ChainID),
			zap.String("chain_name", chain.ChainName))
		return gm.monitorChainRPC(ctx, chain)
	}

	// Get WebSocket endpoint
	wsEndpoint, err := config.GetConfigReader(ctx).GetWSEndpoint(chain.ChainID)
	if err != nil {
		return fmt.Errorf("error getting websocket endpoint: %w", err)
	}

	// Create WebSocket client
	wsClient, err := rpc.DialContext(ctx, wsEndpoint)
	if err != nil {
		metrics.FromContext(ctx).SetConnectionType(chain.ChainID, "gas_monitor", metrics.ConnectionTypeRPC)
		lmt.Logger(ctx).Info("WebSocket connection failed, falling back to RPC polling",
			zap.String("chain_id", chain.ChainID),
			zap.String("chain_name", chain.ChainName),
			zap.Error(err))
		return gm.monitorChainRPC(ctx, chain)
	}

	lmt.Logger(ctx).Info("Attempting WebSocket subscription",
		zap.String("chain_id", chain.ChainID),
		zap.String("chain_name", chain.ChainName))

	// Create the headers channel before subscription
	headers := make(chan *types.Header)

	// Use EthSubscribe like the transfer monitor does
	sub, err := wsClient.EthSubscribe(ctx, headers, "newHeads")
	if err != nil {
		// Handle different failure scenarios
		switch {
		case strings.Contains(err.Error(), "not supported"):
			// Expected for Cosmos chains or non-WebSocket clients
			// Add metrics but keep existing logging
			metrics.FromContext(ctx).SetConnectionType(chain.ChainID, "gas_monitor", metrics.ConnectionTypeRPC)
			metrics.FromContext(ctx).RecordConnectionSwitch(chain.ChainID, "gas_monitor", 
				metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)
			
			lmt.Logger(ctx).Info("WebSocket not supported, falling back to RPC polling",
				zap.String("chain_id", chain.ChainID),
				zap.String("chain_name", chain.ChainName),
				zap.String("reason", "not supported"))
			return gm.monitorChainRPC(ctx, chain)
			
		case strings.Contains(err.Error(), "notifications not supported"):
			// RPC endpoint doesn't support WebSocket
			// Add metrics but keep existing logging
			metrics.FromContext(ctx).SetConnectionType(chain.ChainID, "gas_monitor", metrics.ConnectionTypeRPC)
			metrics.FromContext(ctx).RecordSubscriptionError(chain.ChainID, "gas_monitor", "notifications_not_supported")
			metrics.FromContext(ctx).RecordConnectionSwitch(chain.ChainID, "gas_monitor", 
				metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)
			
			lmt.Logger(ctx).Info("WebSocket notifications not supported, falling back to RPC polling",
				zap.String("chain_id", chain.ChainID),
				zap.String("chain_name", chain.ChainName),
				zap.String("reason", "notifications not supported"))
			return gm.monitorChainRPC(ctx, chain)
			
		default:
			// Unexpected error
			// Add metrics but keep existing logging
			metrics.FromContext(ctx).SetConnectionType(chain.ChainID, "gas_monitor", metrics.ConnectionTypeRPC)
			metrics.FromContext(ctx).RecordSubscriptionError(chain.ChainID, "gas_monitor", "unexpected_error")
			metrics.FromContext(ctx).RecordConnectionSwitch(chain.ChainID, "gas_monitor", 
				metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)
			
			lmt.Logger(ctx).Error("WebSocket subscription failed",
				zap.String("chain_id", chain.ChainID),
				zap.String("chain_name", chain.ChainName),
				zap.Error(err))
			return gm.monitorChainRPC(ctx, chain)
		}
	}

	// Add metric for successful WebSocket connection
	metrics.FromContext(ctx).SetConnectionType(chain.ChainID, "gas_monitor", metrics.ConnectionTypeWebSocket)
	
	lmt.Logger(ctx).Info("Successfully established WebSocket connection",
		zap.String("chain_id", chain.ChainID),
		zap.String("chain_name", chain.ChainName))

	blockCount := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-sub.Err():
			// Add metrics but keep existing logging
			metrics.FromContext(ctx).RecordSubscriptionError(chain.ChainID, "gas_monitor", "subscription_error")
			metrics.FromContext(ctx).RecordConnectionSwitch(chain.ChainID, "gas_monitor", 
				metrics.ConnectionTypeWebSocket, metrics.ConnectionTypeRPC)
			
			lmt.Logger(ctx).Error("WebSocket subscription error, falling back to RPC polling",
				zap.String("chain_id", chain.ChainID),
				zap.String("chain_name", chain.ChainName),
				zap.Error(err))
			return gm.monitorChainRPC(ctx, chain)
		case _ = <-headers:
			blockCount++
			metrics.FromContext(ctx).IncrementBlocksReceived(chain.ChainID, "gas_monitor")

			// Use ClientManager instead of creating new EVMBridgeClient
			client, err := gm.clientManager.GetClient(ctx, chain.ChainID)
			if err != nil {
				lmt.Logger(ctx).Error("failed to get client", 
					zap.String("chain_id", chain.ChainID),
					zap.Error(err))
				continue
			}

			if err := monitorGasBalance(ctx, chain.ChainID, client); err != nil {
				lmt.Logger(ctx).Error("failed to monitor gas balance",
					zap.String("chain_id", chain.ChainID),
					zap.Error(err))
			}
		}
	}
}

func (gm *GasMonitor) monitorChainRPC(ctx context.Context, chain config.ChainConfig) error {
	ticker := time.NewTicker(gm.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			client, err := gm.clientManager.GetClient(ctx, chain.ChainID)
			if err != nil {
				return err
			}
			if err := monitorGasBalance(ctx, chain.ChainID, client); err != nil {
				lmt.Logger(ctx).Error("failed to monitor gas balance",
					zap.String("chain_id", chain.ChainID),
					zap.Error(err))
			}
		}
	}
}


var (
	lastLowBalanceAlert = make(map[string]time.Time)
	lastBalanceCheck    = make(map[string]time.Time)
	lowBalanceAlertCooldown = 1 * time.Hour   
	balanceCheckCooldown   = 1 * time.Hour  
)

// monitorGasBalance exports a metric indicating the current gas balance of the relayer signer and whether it is below alerting thresholds
func monitorGasBalance(ctx context.Context, chainID string, chainClient cctp.BridgeClient) error {
	// Check if we should skip the balance check
	lastCheck, exists := lastBalanceCheck[chainID]
	if exists && time.Since(lastCheck) < balanceCheckCooldown {
		return nil // Skip checking balance during cooldown period
	}
	
	balance, err := chainClient.SignerGasTokenBalance(ctx)
	lastBalanceCheck[chainID] = time.Now() // Update last check time
	
	if err != nil {
		lmt.Logger(ctx).Error("failed to get gas token balance", zap.Error(err), zap.String("chain_id", chainID))
		return err
	}

	chainConfig, err := config.GetConfigReader(ctx).GetChainConfig(chainID)
	if err != nil {
		return err
	}
	warningThreshold, criticalThreshold, err := config.GetConfigReader(ctx).GetGasAlertThresholds(chainID)
	if err != nil {
		return err
	}
	if balance == nil || warningThreshold == nil || criticalThreshold == nil {
		return fmt.Errorf("gas balance or alert thresholds are nil for chain %s", chainID)
	}
	if balance.Cmp(criticalThreshold) < 0 {
		lmt.Logger(ctx).Error("low balance", zap.String("balance", balance.String()), zap.String("chainID", chainID))
	}
	metrics.FromContext(ctx).SetGasBalance(chainID, chainConfig.ChainName, chainConfig.GasTokenSymbol, *balance, *warningThreshold, *criticalThreshold, chainConfig.GasTokenDecimals)
	return nil
}
