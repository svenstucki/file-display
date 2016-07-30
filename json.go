package main

type update struct {
	File    string `json:"file"`
	Content string `json:"content",omitempty`
}
