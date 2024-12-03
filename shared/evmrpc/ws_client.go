package evmrpc

import (
    "context"
    "github.com/ethereum/go-ethereum"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/ethclient"
    "github.com/ethereum/go-ethereum/rpc"
)

type WSClient struct {
    client    *ethclient.Client
    rpcClient *rpc.Client
    sub       ethereum.Subscription
}

func NewWSClient(wsURL string) (*WSClient, error) {
    rpcClient, err := rpc.Dial(wsURL)
    if err != nil {
        return nil, err
    }
    
    client := ethclient.NewClient(rpcClient)
    
    return &WSClient{
        client:    client,
        rpcClient: rpcClient,
    }, nil
}

func (w *WSClient) SubscribeNewHeads(ctx context.Context) (<-chan *types.Header, error) {
    headers := make(chan *types.Header)
    sub, err := w.client.SubscribeNewHead(ctx, headers)
    if err != nil {
        return nil, err
    }
    w.sub = sub
    return headers, nil
}

func (w *WSClient) Close() {
    if w.sub != nil {
        w.sub.Unsubscribe()
    }
    w.rpcClient.Close()
}