package consultation

import (

)

func buildPrompt(question string) string {
	return consultantPrompt + "\n" + question
}