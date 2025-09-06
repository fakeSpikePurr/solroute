package config

type Config struct {
	PrivateKey string `json:"privateKey"`
	RPC        string `json:"rpc"`
	WSRPC      string `json:"wsRpc"`
	JitoRPC    string `json:"jitoRpc"` // refer to: https://docs.jito.wtf/lowlatencytxnsend/
}
