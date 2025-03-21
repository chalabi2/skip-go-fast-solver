package metrics

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/go-kit/kit/metrics"
	prom "github.com/go-kit/kit/metrics/prometheus"
	stdprom "github.com/prometheus/client_golang/prometheus"
)

const (
	chainIDLabel            = "chain_id"
	sourceChainIDLabel      = "source_chain_id"
	destinationChainIDLabel = "destination_chain_id"
	successLabel            = "success"
	orderStatusLabel        = "order_status"
	transferStatusLabel     = "transfer_status"
	settlementStatusLabel   = "settlement_status"
	transactionTypeLabel    = "transaction_type"
	gasBalanceLevelLabel    = "gas_balance_level"
	gasTokenSymbolLabel     = "gas_token_symbol"
	chainNameLabel          = "chain_name"

	// Connection types
	ConnectionTypeWebSocket = "websocket"
	ConnectionTypeRPC       = "rpc"
)

type Metrics interface {
	IncTransactionSubmitted(success bool, chainID, transactionType string)
	IncTransactionVerified(success bool, chainID string)

	IncFillOrderStatusChange(sourceChainID, destinationChainID, orderStatus string)
	ObserveFillLatency(sourceChainID, destinationChainID string, orderStatus string, latency time.Duration)

	IncOrderSettlementStatusChange(sourceChainID, destinationChainID, settlementStatus string)
	ObserveSettlementLatency(sourceChainID, destinationChainID string, settlementStatus string, latency time.Duration)

	IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID string, transferStatus string)

	IncHyperlaneCheckpointingErrors()
	IncHyperlaneMessages(sourceChainID, destinationChainID string, messageStatus string)
	ObserveHyperlaneLatency(sourceChainID, destinationChainID, transferStatus string, latency time.Duration)
	IncHyperlaneRelayTooExpensive(sourceChainID, destinationChainID string)

	ObserveTransferSizeOutOfRange(sourceChainID, destinationChainID string, amountOutOfRange int64)
	ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBpsExceededBy int64)
	ObserveInsufficientBalanceError(chainID string, amountInsufficientBy uint64)

	SetGasBalance(chainID, chainName, gasTokenSymbol string, gasBalance, warningThreshold, criticalThreshold big.Int, gasTokenDecimals uint8)

	IncExcessiveOrderFulfillmentLatency(sourceChainID, destinationChainID, orderStatus string)
	IncExcessiveOrderSettlementLatency(sourceChainID, destinationChainID, settlementStatus string)
	IncExcessiveHyperlaneRelayLatency(sourceChainID, destinationChainID string)

	SetConnectionType(chainID, monitor, connType string)
	IncrementBlocksReceived(chainID, monitor string)
	RecordSubscriptionError(chainID, monitor, errorType string)
	RecordConnectionSwitch(chainID, monitor, fromType, toType string)
}

type metricsContextKey struct{}

func ContextWithMetrics(ctx context.Context, metrics Metrics) context.Context {
	return context.WithValue(ctx, metricsContextKey{}, metrics)
}

func FromContext(ctx context.Context) Metrics {
	metricsFromContext := ctx.Value(metricsContextKey{})
	if metricsFromContext == nil {
		return NewNoOpMetrics()
	} else {
		return metricsFromContext.(Metrics)
	}
}

var _ Metrics = (*PromMetrics)(nil)

type PromMetrics struct {
	totalTransactionSubmitted metrics.Counter
	totalTransactionsVerified metrics.Counter

	fillOrderStatusChange metrics.Counter
	fillLatency           metrics.Histogram

	orderSettlementStatusChange      metrics.Counter
	settlementLatency                metrics.Histogram
	excessiveOrderSettlementLatency  metrics.Counter
	excessiveOrderFulfillmentLatency metrics.Counter

	fundRebalanceTransferStatusChange metrics.Counter

	hplMessageStatusChange         metrics.Counter
	hplCheckpointingErrors         metrics.Counter
	hplLatency                     metrics.Histogram
	hplRelayTooExpensive           metrics.Counter
	excessiveHyperlaneRelayLatency metrics.Counter

	transferSizeOutOfRange    metrics.Histogram
	feeBpsRejections          metrics.Histogram
	insufficientBalanceErrors metrics.Histogram

	gasBalance      metrics.Gauge
	gasBalanceState metrics.Gauge

	// New WebSocket metrics using go-kit metrics types
	connectionType     metrics.Gauge
	blocksReceived     metrics.Counter
	subscriptionErrors metrics.Counter
	connectionSwitches metrics.Counter
}

func NewPromMetrics() Metrics {
	m := &PromMetrics{
		fillOrderStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "fill_order_status_change_counter",
			Help:      "numbers of fill order status changes, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, orderStatusLabel}),
		excessiveOrderFulfillmentLatency: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "excessive_order_fulfillment_latency_counter",
			Help:      "number of observations of excessive order fulfillment latency, paginated by source and destination chain and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, orderStatusLabel}),
		excessiveOrderSettlementLatency: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "excessive_order_settlement_latency_counter",
			Help:      "number of observations of excessive order settlement latency, paginated by source and destination chain and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, settlementStatusLabel}),
		orderSettlementStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "order_settlement_status_change_counter",
			Help:      "numbers of order settlement status changes, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, settlementStatusLabel}),
		fundRebalanceTransferStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "funds_rebalance_transfer_status_change_counter",
			Help:      "numbers of funds rebalance transfer status changes, paginated by source and destination chain, and status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),
		totalTransactionSubmitted: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "total_transactions_submitted_counter",
			Help:      "number of transactions submitted, paginated by success status and source and destination chain id",
		}, []string{successLabel, chainIDLabel, transactionTypeLabel}),
		totalTransactionsVerified: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "total_transactions_verified_counter",
			Help:      "number of transactions verified, paginated by success status and chain id",
		}, []string{successLabel, chainIDLabel}),
		fillLatency: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "latency_per_fill_seconds",
			Help:      "latency from source transaction to fill completion, paginated by source and destination chain id (in seconds)",
			Buckets:   []float64{1, 5, 10, 15, 20, 30, 40, 50, 60, 120, 300, 600},
		}, []string{sourceChainIDLabel, destinationChainIDLabel, orderStatusLabel}),
		settlementLatency: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "latency_per_settlement_minutes",
			Help:      "latency from source transaction to fill completion, paginated by source and destination chain id (in minutes)",
			Buckets:   []float64{1, 5, 15, 30, 60, 120, 180, 240, 300},
		}, []string{sourceChainIDLabel, destinationChainIDLabel, settlementStatusLabel}),
		hplMessageStatusChange: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "hyperlane_message_status_change_counter",
			Help:      "number of hyperlane messages status changes, paginated by source and destination chain, and message status",
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),

		hplCheckpointingErrors: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "hyperlane_checkpointing_errors",
			Help:      "number of hyperlane checkpointing errors",
		}, []string{}),
		hplLatency: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "latency_per_hyperlane_message_seconds",
			Help:      "latency for hyperlane message relaying, paginated by status, source and destination chain id (in seconds)",
			Buckets:   []float64{1, 5, 10, 15, 20, 30, 40, 50, 60, 120, 300, 600},
		}, []string{sourceChainIDLabel, destinationChainIDLabel, transferStatusLabel}),
		hplRelayTooExpensive: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "hyperlane_relay_too_expensive_counter",
			Help:      "counter of relay attempts that were aborted due to being too expensive",
		}, []string{sourceChainIDLabel, destinationChainIDLabel}),
		excessiveHyperlaneRelayLatency: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "excessive_hyperlane_relay_latency_counter",
			Help:      "number of observations of excessive hyperlane relay latency, paginated by source and destination chain",
		}, []string{sourceChainIDLabel, destinationChainIDLabel}),

		transferSizeOutOfRange: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "transfer_size_out_of_range",
			Help:      "histogram of transfer sizes that were out of min/max fill size constraints",
			Buckets: []float64{
				-1000000000,   // -1,000 USDC
				-100000000,    // -100 USDC
				-10000000,     // -10 USDC
				100000000,     // 100 USDC
				1000000000,    // 1,000 USDC
				10000000000,   // 10,000 USDC
				100000000000,  // 100,000 USDC
				1000000000000, // 1,000,000 USDC
			},
		}, []string{sourceChainIDLabel, destinationChainIDLabel}),
		feeBpsRejections: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "fee_bps_rejections",
			Help:      "histogram of fee bps that were rejected for being too low",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 200, 500, 1000},
		}, []string{sourceChainIDLabel, destinationChainIDLabel}),
		insufficientBalanceErrors: prom.NewHistogramFrom(stdprom.HistogramOpts{
			Namespace: "solver",
			Name:      "insufficient_balance_errors",
			Help:      "histogram of fill orders that exceeded available balance",
			Buckets: []float64{
				100000000,     // 100 USDC
				1000000000,    // 1,000 USDC
				10000000000,   // 10,000 USDC
				100000000000,  // 100,000 USDC
				1000000000000, // 1,000,000 USDC
			},
		}, []string{chainIDLabel}),
		gasBalance: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "gas_balance_gauge",
			Help:      "gas balances, paginated by chain id",
		}, []string{chainIDLabel, chainNameLabel, gasTokenSymbolLabel}),
		gasBalanceState: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "gas_balance_state_gauge",
			Help:      "gas balance states (0=ok 1=warning 2=critical), paginated by chain id",
		}, []string{chainIDLabel, chainNameLabel}),

		// New WebSocket metrics using go-kit constructors
		connectionType: prom.NewGaugeFrom(stdprom.GaugeOpts{
			Namespace: "solver",
			Name:      "chain_connection_type",
			Help:      "Current connection type (0=RPC, 1=WebSocket) for each chain and monitor",
		}, []string{"chain_id", "monitor"}),

		blocksReceived: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "websocket_blocks_received_total",
			Help:      "Number of blocks received via WebSocket per chain and monitor",
		}, []string{"chain_id", "monitor"}),

		subscriptionErrors: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "websocket_subscription_errors_total",
			Help:      "Number of WebSocket subscription errors per chain, monitor, and error type",
		}, []string{"chain_id", "monitor", "error_type"}),

		connectionSwitches: prom.NewCounterFrom(stdprom.CounterOpts{
			Namespace: "solver",
			Name:      "connection_type_switches_total",
			Help:      "Number of switches between connection types per chain and monitor",
		}, []string{"chain_id", "monitor", "from_type", "to_type"}),
	}

	return m
}

func (m *PromMetrics) IncTransactionSubmitted(success bool, chainID, transactionType string) {
	m.totalTransactionSubmitted.With(successLabel, fmt.Sprint(success), chainIDLabel, chainID, transactionTypeLabel, transactionType).Add(1)
}

func (m *PromMetrics) IncTransactionVerified(success bool, chainID string) {
	m.totalTransactionsVerified.With(successLabel, fmt.Sprint(success), chainIDLabel, chainID).Add(1)
}

func (m *PromMetrics) ObserveFillLatency(sourceChainID, destinationChainID, orderStatus string, latency time.Duration) {
	m.fillLatency.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, orderStatusLabel, orderStatus).Observe(latency.Seconds())
}

func (m *PromMetrics) ObserveSettlementLatency(sourceChainID, destinationChainID, settlementStatus string, latency time.Duration) {
	m.settlementLatency.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, settlementStatusLabel, settlementStatus).Observe(latency.Minutes())
}

func (m *PromMetrics) ObserveHyperlaneLatency(sourceChainID, destinationChainID, transferStatus string, latency time.Duration) {
	m.hplLatency.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, transferStatus).Observe(latency.Seconds())
}

func (m *PromMetrics) IncFillOrderStatusChange(sourceChainID, destinationChainID, orderStatus string) {
	m.fillOrderStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, orderStatusLabel, orderStatus).Add(1)
}

func (m *PromMetrics) IncOrderSettlementStatusChange(sourceChainID, destinationChainID, settlementStatus string) {
	m.orderSettlementStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, settlementStatusLabel, settlementStatus).Add(1)
}

func (m *PromMetrics) IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID, transferStatus string) {
	m.fundRebalanceTransferStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, transferStatus).Add(1)
}

func (m *PromMetrics) IncHyperlaneCheckpointingErrors() {
	m.hplCheckpointingErrors.Add(1)
}

func (m *PromMetrics) IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {
	m.hplMessageStatusChange.With(sourceChainIDLabel, sourceChainID, destinationChainIDLabel, destinationChainID, transferStatusLabel, messageStatus).Add(1)
}

func (m *PromMetrics) IncHyperlaneRelayTooExpensive(sourceChainID, destinationChainID string) {
	m.hplRelayTooExpensive.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
	).Add(1)
}

func (m *PromMetrics) ObserveTransferSizeOutOfRange(sourceChainID, destinationChainID string, amountOutOfRange int64) {
	m.transferSizeOutOfRange.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
	).Observe(float64(amountOutOfRange))
}

func (m *PromMetrics) ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBps int64) {
	m.feeBpsRejections.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
	).Observe(float64(feeBps))
}

func (m *PromMetrics) ObserveInsufficientBalanceError(chainID string, difference uint64) {
	m.insufficientBalanceErrors.With(
		chainIDLabel, chainID,
	).Observe(float64(difference))
}

func (m *PromMetrics) SetGasBalance(chainID, chainName, gasTokenSymbol string, gasBalance, warningThreshold, criticalThreshold big.Int, gasTokenDecimals uint8) {
	// We compare the gas balance against thresholds locally rather than in the grafana alert definition since
	// the prometheus metric is exported as a float64 and the thresholds reach Wei amounts where precision is lost.
	gasBalanceFloat, _ := gasBalance.Float64()
	gasTokenAmount := gasBalanceFloat / (math.Pow10(int(gasTokenDecimals)))
	gasBalanceState := 0
	if gasBalance.Cmp(&criticalThreshold) < 0 {
		gasBalanceState = 2
	} else if gasBalance.Cmp(&warningThreshold) < 0 {
		gasBalanceState = 1
	}
	m.gasBalanceState.With(chainIDLabel, chainID, chainNameLabel, chainName).Set(float64(gasBalanceState))
	m.gasBalance.With(chainIDLabel, chainID, chainNameLabel, chainName, gasTokenSymbolLabel, gasTokenSymbol).Set(gasTokenAmount)
}

func (m *PromMetrics) IncExcessiveOrderFulfillmentLatency(sourceChainID, destinationChainID, orderStatus string) {
	m.excessiveOrderFulfillmentLatency.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
		orderStatusLabel, orderStatus,
	).Add(1)
}

func (m *PromMetrics) IncExcessiveOrderSettlementLatency(sourceChainID, destinationChainID, settlementStatus string) {
	m.excessiveOrderSettlementLatency.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
		settlementStatusLabel, settlementStatus,
	).Add(1)
}

func (m *PromMetrics) IncExcessiveHyperlaneRelayLatency(sourceChainID, destinationChainID string) {
	m.excessiveHyperlaneRelayLatency.With(
		sourceChainIDLabel, sourceChainID,
		destinationChainIDLabel, destinationChainID,
	).Add(1)
}

func (m *PromMetrics) SetConnectionType(chainID, monitor, connType string) {
	value := 0.0
	if connType == ConnectionTypeWebSocket {
		value = 1.0
	}
	m.connectionType.With("chain_id", chainID, "monitor", monitor).Set(value)
}

func (m *PromMetrics) IncrementBlocksReceived(chainID, monitor string) {
	m.blocksReceived.With("chain_id", chainID, "monitor", monitor).Add(1)
}

func (m *PromMetrics) RecordSubscriptionError(chainID, monitor, errorType string) {
	m.subscriptionErrors.With("chain_id", chainID, "monitor", monitor, "error_type", errorType).Add(1)
}

func (m *PromMetrics) RecordConnectionSwitch(chainID, monitor, fromType, toType string) {
	m.connectionSwitches.With("chain_id", chainID, "monitor", monitor, "from_type", fromType, "to_type", toType).Add(1)
}

type NoOpMetrics struct{}

func (n NoOpMetrics) IncExcessiveOrderFulfillmentLatency(sourceChainID, destinationChainID, orderStatus string) {
}
func (n NoOpMetrics) IncExcessiveOrderSettlementLatency(sourceChainID, destinationChainID, settlementStatus string) {
}
func (n NoOpMetrics) IncExcessiveHyperlaneRelayLatency(sourceChainID, destinationChainID string) {
}
func (n NoOpMetrics) IncHyperlaneRelayTooExpensive(sourceChainID, destinationChainID string) {
}
func (n NoOpMetrics) ObserveInsufficientBalanceError(chainID string, amountInsufficientBy uint64) {
}
func (n NoOpMetrics) IncTransactionSubmitted(success bool, chainID, transactionType string) {
}
func (n NoOpMetrics) IncTransactionVerified(success bool, chainID string) {
}
func (n NoOpMetrics) ObserveFillLatency(sourceChainID, destinationChainID, orderStatus string, latency time.Duration) {
}
func (n NoOpMetrics) ObserveSettlementLatency(sourceChainID, destinationChainID, settlementStatus string, latency time.Duration) {
}
func (n NoOpMetrics) ObserveHyperlaneLatency(sourceChainID, destinationChainID, orderstatus string, latency time.Duration) {
}
func (n NoOpMetrics) IncFillOrderStatusChange(sourceChainID, destinationChainID, orderStatus string) {
}
func (n NoOpMetrics) IncOrderSettlementStatusChange(sourceChainID, destinationChainID, settlementStatus string) {
}
func (n NoOpMetrics) IncFundsRebalanceTransferStatusChange(sourceChainID, destinationChainID, transferStatus string) {
}
func (n NoOpMetrics) IncHyperlaneCheckpointingErrors()                                             {}
func (n NoOpMetrics) IncHyperlaneMessages(sourceChainID, destinationChainID, messageStatus string) {}
func (n NoOpMetrics) ObserveTransferSizeOutOfRange(sourceChainID, destinationChainID string, amountExceededBy int64) {
}
func (n *NoOpMetrics) SetGasBalance(chainID, chainName, gasTokenSymbol string, gasBalance, warningThreshold, criticalThreshold big.Int, gasTokenDecimals uint8) {
}
func (n NoOpMetrics) ObserveFeeBpsRejection(sourceChainID, destinationChainID string, feeBps int64) {}
func (n NoOpMetrics) SetConnectionType(chainID, monitor, connType string)                           {}
func (n NoOpMetrics) IncrementBlocksReceived(chainID, monitor string)                               {}
func (n NoOpMetrics) RecordSubscriptionError(chainID, monitor, errorType string)                    {}
func (n NoOpMetrics) RecordConnectionSwitch(chainID, monitor, fromType, toType string)              {}
func NewNoOpMetrics() Metrics {
	return &NoOpMetrics{}
}
