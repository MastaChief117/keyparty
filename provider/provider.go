// Copyright 2026 KeyParty Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
