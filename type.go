package main

type Response struct {
	TwitchEmoteName      string `json:"twitchEmoteName,omitempty" required:"true" description:"Twitch emote name, must start with a capital letter such as 'Example'; if possible, use an applicable meme as the name/filename rather than the literal representation of the emote, such as 'TakeMyMoney', 'Stonks', or 'Dumpy'"`
	Description          string `json:"description,omitempty" required:"true" description:"Describe the image in a short phrase"`
	Category             string `json:"category,omitempty" required:"true" description:"Emote category; must a non-exclusive single-word adjective"`
	NSFW                 bool   `json:"nsfw,omitempty" required:"true" description:"Indicate if the image is safe vs not safe for work"`
	ColorDescription     string `json:"colorDescription,omitempty" required:"true" description:"Verbose description of the key colors in the image, in plain English, using accurate detailed color names"`
	MainColor            string `json:"mainColor,omitempty" required:"true" description:"Main color of the image in hexadecimal format"`
	SecondaryColor       string `json:"secondaryColor,omitempty" required:"true" description:"Secondary color of the image in hexadecimal format"`
	EmoteType            string `json:"emoteType,omitempty" required:"true" description:"Type of emote; must be one of 'static', 'animated', or 'animated-static'"`
	EmoteStyle           string `json:"emoteStyle,omitempty" required:"true" description:"Style of emote; must be one of 'cartoon', 'realistic', or 'abstract'"`
	EmoteExpression      string `json:"emoteExpression,omitempty" required:"true" description:"Emote expression; must be one of 'happy', 'sad', 'angry', 'surprised', 'disgusted', 'scared', or 'neutral'"`
	EmoteFormat          string `json:"emoteFormat,omitempty" required:"true" description:"Emote format; examples: 'png', 'gif', 'apng'"`
	EmoteSize            string `json:"emoteSize,omitempty" required:"true" description:"Emote size in pixels"`
	EmoteQuality         string `json:"emoteQuality,omitempty" required:"true" description:"Emote quality; must be one of 'low', 'medium', or 'high'"`
	EmoteSuitability28px string `json:"emoteSuitability28px,omitempty" required:"true" description:"How well does the 28x28px image work as a 28x28 chat emote; must be one of 'excellent', 'good', 'acceptable', or 'poor'; be mindful of the level of detail and complexity in the image; assume zooming is not available as a feature to the end user"`
}

type Result struct {
	Response
	SourceFilepath   string   `json:"source,omitempty"`
	DestinationFiles []string `json:"destination,omitempty"`
	Error            error    `json:"-"`
	ErrorString      string   `json:"error,omitempty"`
}
