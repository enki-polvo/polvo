package model

type CommonLogWrapper struct {
	// Common log Header
	EventName string      `json:"eventname"`
	Source    string      `json:"source"`
	Timestmp  string      `json:"timestamp"`
	Log       string      `json:"log"`
	MetaData  interface{} `json:"metadata"`
	// Reference count is used to track the number of references to the log sync pool
	RefCount int32  `json:"-"`
	Tag      string `json:"-"`
}
