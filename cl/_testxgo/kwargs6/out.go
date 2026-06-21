package main

type CompletionParams interface {
	MaxOutputTokens(int64) CompletionParams
	Set(name string, val interface{}) CompletionParams
	System(prompt ...string) CompletionParams
}
type completionParamsImpl struct {
	data map[string]interface{}
}
type Client struct {
}

func (p *completionParamsImpl) Set(name string, val interface{}) CompletionParams {
	p.data[name] = val
	return p
}
func (p *completionParamsImpl) MaxOutputTokens(n int64) CompletionParams {
	p.data["maxOutputTokens"] = n
	return p
}
func (p *completionParamsImpl) System(prompt ...string) CompletionParams {
	p.data["system"] = prompt
	return p
}
func (c *Client) CompletionParams() CompletionParams {
	return &completionParamsImpl{data: make(map[string]interface{})}
}
func (c *Client) Complete(prompt string, params CompletionParams) {
}

var c Client
// Known typed keywords + unknown keyword falling back to Set
func main() {
	c.Complete("hello", c.CompletionParams().MaxOutputTokens(1024).System("You are helpful.").Set("topP", 0.9))
}
