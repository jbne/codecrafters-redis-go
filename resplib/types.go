package resplib

type (
	RESP2_Array []string

	RESP2_CommandRequest struct {
		Params          RESP2_Array
		ResponseChannel chan<- RESP2_CommandResponse
	}

	RESP2_CommandResponse = string
)
