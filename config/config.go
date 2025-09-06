package config

type Config struct {
	PrivateKey string `json:"privateKey"`
	RPC        string `json:"rpc"`
	WSRPC      string `json:"wsRpc"`
}
