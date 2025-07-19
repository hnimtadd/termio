package page

// The sematic prompt type. This is used when tracking a line type and r
// equires integration with the shell. By default, we mark a line as "none"
// meaning we don't know what type it is.
//
// See: https://gitlab.freedesktop.org/Per_Bothner/specifications/blob/master/proposals/semantic-prompts.md
type SemanticPromptType int

const (
	SemanticPromptTypePrompt SemanticPromptType = iota
	SemanticPromptTypeContinuation
	SemanticPromptTypeInput
	SemanticPromptTypeOutput
	SemanticPromptTypeUnknow
)

// Return trues if this is a prompt or input line type.
func (p SemanticPromptType) PromptOrInput() bool {
	return p == SemanticPromptTypePrompt ||
		p == SemanticPromptTypeContinuation ||
		p == SemanticPromptTypeInput
}
