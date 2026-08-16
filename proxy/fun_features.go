package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"ai-gateway/store"
)

// ── AI FORTUNE TELLER ──────────────────────────────────────────────────────

func (p *Proxy) HandleFortune(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	fortunes := []string{
		"Your code will compile on the first try... just kidding, fix line 42.",
		"A wild nil pointer appears! Your debugging journey begins now.",
		"The stars say your PR will be approved without comments. LOL.",
		"Your next commit message will be 'fixed stuff'. Embrace mediocrity.",
		"Segfault in your future. Touch grass before it touches you.",
		"A mysterious bug will appear at 3AM. It was a semicolon all along.",
		"Your code review will include the phrase 'have you tried turning it off and on again?'",
		"The algorithm gods demand a sacrifice. Offer a console.log.",
		"Your database will migrate itself. Just kidding, run the backup first.",
		"A stack overflow approaches. Reduce your recursion, mortal.",
		"Your deployment will succeed... in the staging environment only.",
		"The CI/CD pipeline smiles upon you today. Merge with confidence.",
		"A wild dependency update appears! Your build will break for 3 days.",
		"Your code is 99% bug-free. The 1% will crash production.",
		"The rubber duck on your desk has wisdom. Ask it about your logic.",
	}

	fortune := fortunes[rand.Intn(len(fortunes))]

	if reqBody.Message != "" {
		fortune = fmt.Sprintf("You asked: '%s'\n\n%s", reqBody.Message, fortune)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"fortune":  fortune,
		"provider": "keyparty",
		"mood":     "cosmic",
		"warning":  "Side effects may include: imposter syndrome, existential dread, and sudden urges to rewrite everything in Rust.",
	})
}

// ── PROVIDER ROAST MODE ───────────────────────────────────────────────────

func (p *Proxy) HandleProviderRoast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Provider1 string `json:"provider1"`
		Provider2 string `json:"provider2"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) < 2 {
		http.Error(w, `{"error":"Need at least 2 providers"}`, 400)
		return
	}

	providerKeys := make(map[string][]store.APIKey)
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			providerKeys[k.Provider] = append(providerKeys[k.Provider], k)
		}
	}

	available := make([]string, 0, len(providerKeys))
	for pname := range providerKeys {
		available = append(available, pname)
	}
	if len(available) < 2 {
		http.Error(w, `{"error":"Need at least 2 different providers"}`, 400)
		return
	}

	pickProvider := func(requested string) (string, []store.APIKey) {
		if requested != "" {
			if kList, ok := providerKeys[requested]; ok && len(kList) > 0 {
				return requested, kList
			}
		}
		for _, pname := range available {
			if kList, ok := providerKeys[pname]; ok && len(kList) > 0 {
				return pname, kList
			}
		}
		return "", nil
	}

	nameA, keysA := pickProvider(reqBody.Provider1)
	nameB, keysB := pickProvider(reqBody.Provider2)
	if nameA == nameB && len(available) >= 2 {
		for _, alt := range available {
			if alt != nameA {
				nameB = alt
				keysB = providerKeys[alt]
				break
			}
		}
	}

	systemA := fmt.Sprintf("You are %s. ROAST BATTLE against %s. Output ONLY the roast, nothing else. No preamble, no explanation, no reasoning. Just the roast. Max 100 words. Be savage but funny.", nameA, nameB)
	systemB := fmt.Sprintf("You are %s. ROAST BATTLE against %s. Output ONLY the roast, nothing else. No preamble, no explanation, no reasoning. Just the roast. Max 100 words. Be savage but funny.", nameB, nameA)

	messagesA := []map[string]string{
		{"role": "system", "content": systemA},
		{"role": "user", "content": "Open with your first roast against " + nameB + ". Go hard."},
	}

	start := time.Now()
	replyA, provA, err := p.callProviderWithFallback(keysA, messagesA, 200)
	latencyA := time.Since(start).Milliseconds()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": nameA + " failed: " + err.Error()})
		return
	}
	_ = latencyA

	messagesB := []map[string]string{
		{"role": "system", "content": systemB},
		{"role": "user", "content": nameA + " just roasted you:\n\n" + replyA + "\n\nFire back harder."},
	}

	start = time.Now()
	replyB, provB, err := p.callProviderWithFallback(keysB, messagesB, 200)
	latencyB := time.Since(start).Milliseconds()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": nameB + " failed: " + err.Error()})
		return
	}
	_ = latencyB

	messagesA = append(messagesA, map[string]string{"role": "assistant", "content": replyA})
	messagesA = append(messagesA, map[string]string{"role": "user", "content": nameB + " responded:\n\n" + replyB + "\n\nEnd them with a final roast."})

	replyA2, _, err := p.callProviderWithFallback(keysA, messagesA, 200)
	if err != nil {
		replyA2 = replyA
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"provider_a": provA,
		"provider_b": provB,
		"roast_a":    replyA,
		"roast_b":    replyB,
		"finale_a":   replyA2,
		"winner":     "you, for watching two AIs insult each other",
	})
}

// ── DEBUGGING ORACLE ──────────────────────────────────────────────────────

func (p *Proxy) HandleDebugOracle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Error string `json:"error"`
		Stack string `json:"stack"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, `{"error":"Invalid JSON"}`, 400)
		return
	}
	if reqBody.Error == "" {
		http.Error(w, `{"error":"error message required"}`, 400)
		return
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No API keys configured"}`, 400)
		return
	}

	var enabled []store.APIKey
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			enabled = append(enabled, k)
		}
	}
	if len(enabled) == 0 {
		http.Error(w, `{"error":"No enabled keys"}`, 400)
		return
	}

	prompt := fmt.Sprintf("The user encountered this error:\n\nError: %s\n\n", reqBody.Error)
	if reqBody.Stack != "" {
		prompt += "Stack trace:\n" + reqBody.Stack + "\n\n"
	}
	prompt += "You are the Debugging Oracle. Analyze this error dramatically:\n1. What went wrong (in a dramatic tone)\n2. The most likely cause\n3. A simple fix\n4. A life lesson from this bug\n\nKeep it under 200 words. Be theatrical but helpful."

	messages := []map[string]string{{"role": "user", "content": prompt}}

	start := time.Now()
	reply, provider, err := p.callProviderWithFallback(enabled, messages, 400)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"oracle":     reply,
		"provider":   provider,
		"latency_ms": latency,
		"mood":       "mystical",
	})
}

// ── COMMIT MESSAGE GENERATOR ──────────────────────────────────────────────

func (p *Proxy) HandleCommitMsg(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var reqBody struct {
		Diff  string `json:"diff"`
		Style string `json:"style"`
	}
	json.NewDecoder(r.Body).Decode(&reqBody)

	if reqBody.Style == "" {
		reqBody.Style = "dramatic"
	}

	keys, err := p.store.GetKeys()
	if err != nil || len(keys) == 0 {
		http.Error(w, `{"error":"No API keys configured"}`, 400)
		return
	}

	var enabled []store.APIKey
	for _, k := range keys {
		if k.Enabled && k.Provider != "custom" {
			enabled = append(enabled, k)
		}
	}
	if len(enabled) == 0 {
		http.Error(w, `{"error":"No enabled keys"}`, 400)
		return
	}

	stylePrompts := map[string]string{
		"dramatic":     "Write a dramatic, epic commit message as if this code change will change the world. Use metaphors and grandiose language.",
		"professional": "Write a clean, conventional commit message following best practices.",
		"meme":         "Write a hilarious meme-style commit message. Be absurd and funny.",
		"poetic":       "Write a poetic, Shakespearean commit message about this code change.",
	}

	prompt := fmt.Sprintf("Generate a commit message for these changes:\n\n%s\n\nStyle: %s\n\nOutput ONLY the commit message, nothing else.", reqBody.Diff, stylePrompts[reqBody.Style])

	messages := []map[string]string{{"role": "user", "content": prompt}}

	start := time.Now()
	reply, provider, err := p.callProviderWithFallback(enabled, messages, 150)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"commit_message": reply,
		"style":          reqBody.Style,
		"provider":       provider,
		"latency_ms":     latency,
	})
}
