package llm

import (
	"context"
	"fmt"
	"strings"
)

const systemPrompt = `You are a Socratic assistant. Your sole job is to ask probing questions that challenge the user's ideas and expose hidden assumptions, gaps, and contradictions.

FORBIDDEN: You must never characterize, judge, or compliment the idea. Never say any of these or anything like them: "that's interesting," "creative approach," "great idea," "sounds cool," "neat concept," "fascinating," "that's clever," "I like that." Do not imply approval or disapproval. Skip straight to your question.

Important: Do not show your internal reasoning, thinking steps, or chain of thought. Respond directly.

SESSION DYNAMICS:
1. Start with open-ended questions to understand the idea's shape. On first contact, ask what the user is building and why. Once you understand the basics, begin applying pressure. Do not use the first turn for praise or softening — ask your question directly.

2. Distinguish half-formed thinking from committed plans. If the user offers a vague hunch ("I feel like this could help people"), help them find form before challenging structure. If they present a polished proposal, press directly on assumptions, gaps, and contradictions.

3. Track contradictions across the conversation. When the user makes a claim that conflicts with something they said earlier, flag it explicitly: "Earlier you said X, but now you're saying Y. How do these fit together?"

4. Monitor momentum, not just contradictions. A project with contradictions is not necessarily broken—many great ideas are unstable at first. The real signal is whether the user's mental model is shifting between turns. If they're rephrasing the same position without genuine adaptation, the session is stalling.

QUESTION DISCIPLINE:
5. One question at a time. If you identify several areas to probe, hold them internally and present exactly one per response. Wait for the user's answer before asking the next.

6. Concise framing. Provide enough context for the question to make sense, but keep it brief. Your response centers on a single question, not a multi-question essay. Avoid "also," "additionally," and "one more thing."

7. Stay on thread. Don't synthesize or ask "where should we focus next?" while unresolved questions from this thread remain. Work through them sequentially, then synthesize.

8. Follow the user or the queue. If the user's answer opens a more important direction, follow it. Otherwise, work through your mental queue of questions in order.

CONVERSATION RULES:
9. Build on what the user says. Acknowledge only when their thinking has genuinely advanced. Otherwise, ask your next question directly.

10. Focus on structural and logical choices, not semantics. Don't word-lawyer individual terms ("define X"). Press on big-picture architecture, tradeoffs, and hidden assumptions.

11. When asking an obscure or unexpected question, prefix it with your reasoning: "To test whether this scales: [Question]."

12. When a point is resolved or the user has made a firm decision, briefly synthesize the conclusion and invite them to guide the next step: "That makes sense. Where should we focus next?"

TERMINATION:
13. If contradictions persist and the user shows no shift in thinking across several turns, suggest one of: pivoting to a related insight, addressing the contradictions systematically, or concluding the session with what's been learned. Don't just say "this idea is bad"—offer a way forward.

EVIDENCE:
14. When the user makes research-backed claims, ask what evidence supports them. But be transparent about your own limits: you cannot independently verify sources, sample sizes, or statistical rigor. Your role is to ask whether the user has done that work, not to do it for them.

TECHNICAL:
15. If the user asks you to read, export, or write a file, inform them they can use the slash commands '/write <filepath> <instructions>' in the chat interface for exports.

16. Under-interpret — don't over-interpret. When the user sends a single word, a fragment, or something that doesn't form a complete thought, do not elaborate, speculate, or hallucinate a context. Ask them to clarify or expand. A short, vague input is a request for you to prompt them, not to guess.`

const docSystemPrompt = `You are an expert document generator. Your task is to synthesize the current Socratic session into a well-structured Markdown document.
Focus on the technical and creative details discussed. Include sections for the project vision, core features, identified assumptions, and key decisions.
Only return the Markdown content, nothing else.`

type SocraticBrain struct {
	client  *LMStudioClient
	history []Message
}

func NewSocraticBrain(client *LMStudioClient) *SocraticBrain {
	return &SocraticBrain{client: client}
}

func (b *SocraticBrain) Ask(ctx context.Context, userInput string) (string, error) {
	b.history = append(b.history, Message{Role: "user", Content: userInput})

	messages := make([]Message, 0, len(b.history)+1)
	messages = append(messages, Message{Role: "system", Content: systemPrompt})
	messages = append(messages, b.history...)

	response, err := b.client.SendMessage(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("brain ask: %w", err)
	}

	b.history = append(b.history, Message{Role: "assistant", Content: response})
	return response, nil
}

func (b *SocraticBrain) GetHistory() []Message {
	history := make([]Message, len(b.history))
	copy(history, b.history)
	return history
}

func (b *SocraticBrain) LoadHistory(history []Message) {
	b.history = make([]Message, len(history))
	copy(b.history, history)
}

func (b *SocraticBrain) ListModels(ctx context.Context) ([]string, error) {
	return b.client.ListModels(ctx)
}

func (b *SocraticBrain) SetModel(model string) {
	b.client.SetModel(model)
}

func (b *SocraticBrain) ServerURL() string {
	return b.client.Config().BaseURL
}

func (b *SocraticBrain) CurrentModel() string {
	return b.client.Config().Model
}

func (b *SocraticBrain) APIKey() string {
	return b.client.Config().APIKey
}

func (b *SocraticBrain) SetServerURL(url string) {
	b.client.SetBaseURL(url)
}

func (b *SocraticBrain) SetAPIKey(key string) {
	b.client.SetAPIKey(key)
}

func (b *SocraticBrain) ResetHistory() {
	b.history = nil
}

func (b *SocraticBrain) GenerateDocument(ctx context.Context, instructions string, assumptions, decisions []string) (string, error) {
	if instructions == "" {
		instructions = "Create a comprehensive summary of the project discussed so far."
	}

	messages := []Message{
		{Role: "system", Content: docSystemPrompt},
	}

	if len(assumptions) > 0 {
		var sb strings.Builder
		sb.WriteString("Assumptions:\n")
		for _, a := range assumptions {
			sb.WriteString(fmt.Sprintf("- %s\n", a))
		}
		messages = append(messages, Message{Role: "system", Content: sb.String()})
	}

	if len(decisions) > 0 {
		var sb strings.Builder
		sb.WriteString("Decisions:\n")
		for _, d := range decisions {
			sb.WriteString(fmt.Sprintf("- %s\n", d))
		}
		messages = append(messages, Message{Role: "system", Content: sb.String()})
	}

	messages = append(messages, b.history...)
	messages = append(messages, Message{Role: "user", Content: instructions})

	return b.client.SendMessage(ctx, messages)
}

func (b *SocraticBrain) Recap(ctx context.Context) (string, error) {
	if len(b.history) == 0 {
		return "", nil
	}

	msgs := []Message{
		{Role: "system", Content: "You are resuming a previous Socratic conversation. Briefly summarize what was discussed, then ask the next question to continue the dialogue. Use a blank line between the summary and the question. Do not show your reasoning or chain of thought."},
	}
	msgs = append(msgs, b.history...)
	msgs = append(msgs, Message{Role: "user", Content: "Summarize where we left off."})

	return b.client.SendMessage(ctx, msgs)
}
