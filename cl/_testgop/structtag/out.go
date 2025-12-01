package main

type Start struct {
	_ struct {
	} `_:"Start recording meeting minutes"`
}
type MultiTag struct {
	_ struct {
	} `_:"First tag"`
	_ struct {
	} `_:"Second tag"`
	name string
	_    struct {
	} `_:"Third tag"`
}
type MixedStruct struct {
	ID int
	_  struct {
	} `_:"Documentation tag"`
	Name string `json:"name"`
}
