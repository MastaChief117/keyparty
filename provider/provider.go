package provider

type Provider struct {
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
}

var Supported = []Provider{
	{Name: "openai", BaseURL: "https://api.openai.com/v1"},
	{Name: "anthropic", BaseURL: "https://api.anthropic.com/v1"},
	{Name: "gemini", BaseURL: "https://generativelanguage.googleapis.com/v1beta/openai/"},
	{Name: "mistral", BaseURL: "https://api.mistral.ai/v1"},
	{Name: "groq", BaseURL: "https://api.groq.com/openai/v1"},
	{Name: "together", BaseURL: "https://api.together.xyz/v1"},
	{Name: "deepseek", BaseURL: "https://api.deepseek.com/v1"},
	{Name: "openrouter", BaseURL: "https://openrouter.ai/api/v1"},
	{Name: "fireworks", BaseURL: "https://api.fireworks.ai/inference/v1"},
	{Name: "nvidia", BaseURL: "https://integrate.api.nvidia.com/v1"},
	{Name: "custom", BaseURL: ""},
}

func GetByName(name string) (Provider, bool) {
	for _, p := range Supported {
		if p.Name == name {
			return p, true
		}
	}
	return Provider{}, false
}
