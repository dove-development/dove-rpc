package src

type ErrorResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Id interface{} `json:"id"`
}

func ErrorResponseNew(e string) ErrorResponse {
	return ErrorResponse{
		JsonRpc: "2.0",
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    -32000,
			Message: e,
		},
		Id: nil,
	}
}
