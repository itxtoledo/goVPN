package main

type Network struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	PIN         string `json:"pin"`
	ClientCount int    `json:"client_count"`
}
