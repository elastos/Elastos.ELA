package servers

var (
	ElaRpcInvalidHash        = responsePacking("invalid hash")
	ElaRpcInvalidBlock       = responsePacking("invalid block")
	ElaRpcInvalidTransaction = responsePacking("invalid transaction")
	ElaRpcInvalidParameter   = responsePacking("invalid parameter")

	ElaRpcUnknownBlock       = responsePacking("unknown block")
	ElaRpcUnknownTransaction = responsePacking("unknown transaction")

	ElaRpcNil           = responsePacking(nil)
	ElaRpcUnsupported   = responsePacking("Unsupported")
	ElaRpcInternalError = responsePacking("internal error")
	ElaRpcIOError       = responsePacking("internal IO error")
	ElaRpcAPIError      = responsePacking("internal API error")
	ElaRpcSuccess       = responsePacking(true)
	ElaRpcFailed        = responsePacking(false)

	ElaRpc = responsePacking
)

func responsePacking(result interface{}) map[string]interface{} {
	resp := map[string]interface{}{
		"result": result,
	}
	return resp
}