package src

type ErrorResponse struct {
	JsonRpc string `json:"jsonrpc"`
	Error   struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Id any `json:"id"`
}

func ErrorResponseNew(e string, id any) ErrorResponse {
	return ErrorResponse{
		JsonRpc: "2.0",
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    -32000,
			Message: e,
		},
		Id: id,
	}
}
