package service

type CommonHeaderWrapper struct {
	// Common log Header
	EventName string      `json:"eventname"`
	Source    string      `json:"source"`
	Timestmp  string      `json:"timestamp"`
	Log       string      `json:"log"`
	MetaData  interface{} `json:"metadata"`
}
