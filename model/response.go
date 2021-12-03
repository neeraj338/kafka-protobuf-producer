package model

type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error"`
}

func OfSuccess(data interface{}) Response {
	return Response{Success: true, Data: data}
}
func OfFailure(err string) Response {
	return Response{Success: false, Data: nil, Error: err}
}
