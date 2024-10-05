package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"strconv"
	"strings"
	"sync"

	_ "github.com/joho/godotenv/autoload"
	"github.com/nfnt/resize"
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"golang.org/x/image/draw"
)

type Response struct {
	TwitchEmoteName      string `json:"twitchEmoteName,omitempty" required:"true" description:"Twitch emote name, must be prefixed with 'strim99' such as 'strim99Example'; if possible, use an applicable meme as the name/filename rather than the literal representation of the emote, such as 'TakeMyMoney', 'Stonks', or 'Dumpy'"`
	Filename             string `json:"filename,omitempty" required:"true" description:"Recommended filename of the image based on the Twitch emote name; must omit the file extension"`
	Description          string `json:"description,omitempty" required:"true" description:"Describe the image in a short phrase"`
	Category             string `json:"category,omitempty" required:"true" description:"Emote category; must a non-exclusive single-word adjective"`
	NSFW                 bool   `json:"nsfw,omitempty" required:"true" description:"Indicate if the image is safe vs not safe for work"`
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

var (
	maxConcurrent   = 1 // Set the default maximum number of concurrent operations
	client          = openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	promptText      = os.Getenv("PROMPT_TEXT")
	emoteNamePrefix = os.Getenv("EMOTENAME_PREFIX")
	workingDir      = os.Getenv("WORKING_DIR")
	responseSchema  = &jsonschema.Definition{}
)

const (
	emoteNamePrefixPlaceholder = "strim99"
)

func init() {
	if workingDir != "" {
		if err := os.Chdir(workingDir); err != nil {
			panic(err)
		}
	}

	var err error
	responseSchema, err = jsonschema.GenerateSchemaForType(Response{})
	if err != nil {
		panic(err)
	}

	if mc := os.Getenv("OPENAI_API_MAX_CONCURRENT"); mc != "" {
		if n, err := strconv.Atoi(mc); err == nil {
			maxConcurrent = n
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Please provide a path to one or more valid image files as an argument.")
		return
	}

	var sem = make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	results := make(chan Result, len(os.Args)-1)

	wg.Add(1) // Ensure this loop is included in the WaitGroup to prevent premature exit
	go func() {
		defer wg.Done() // Remove this loop from the WaitGroup when the goroutine completes

		for _, path := range os.Args[1:] {
			sem <- struct{}{} // Wait until there is room in the semaphore
			wg.Add(1)         // Increment the WaitGroup counter
			go func() {
				defer wg.Done() // Decrement the WaitGroup counter
				processImage(path, results)
				<-sem // Release a spot in the semaphore
			}()
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.Error != nil {
			result.ErrorString = result.Error.Error()
		}
		out, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(string(out))
	}
}

func processImage(path string, results chan<- Result) {
	dataURIs := []string{}
	imagesBytes := make(map[string][]byte)

	result := &Result{
		SourceFilepath: path,
	}
	defer func() {
		results <- *result
	}()

	// Check if the file exists and is a valid image file
	img, gifImg, err := getImage(path)
	if err != nil {
		result.Error = err
		return
	}

	// Convert original size to PNG or GIF
	if gifImg != nil {
		gifBytes, err := encodeGIF(gifImg)
		if err != nil {
			result.Error = err
			return
		}
		imagesBytes["original"] = gifBytes
		dataURIs = append(dataURIs, imgBytesToDataURI(gifBytes))

		// Convert to 28x28px GIF
		img28 := resizeGIFToSquare(gifImg, 28)
		gifBytes, err = encodeGIF(img28)
		if err != nil {
			result.Error = err
			return
		}
		imagesBytes["28x28"] = gifBytes
		dataURIs = append(dataURIs, imgBytesToDataURI(gifBytes))
	} else {
		pngBytes, err := imageToPNG(img)
		if err != nil {
			result.Error = err
			return
		}
		imagesBytes["original"] = pngBytes
		dataURIs = append(dataURIs, imgBytesToDataURI(pngBytes))

		// Convert to 28x28px PNG
		img28 := resizeToSquare(img, 28)
		pngBytes, err = imageToPNG(img28)
		if err != nil {
			result.Error = err
			return
		}
		imagesBytes["28x28"] = pngBytes
		dataURIs = append(dataURIs, imgBytesToDataURI(pngBytes))
	}

	// Message Parts
	messageParts := []openai.ChatMessagePart{
		{
			Type: openai.ChatMessagePartTypeText,
			Text: promptText,
		},
	}
	for _, dataURI := range dataURIs {
		messageParts = append(messageParts, openai.ChatMessagePart{
			Type: openai.ChatMessagePartTypeImageURL,
			ImageURL: &openai.ChatMessageImageURL{
				URL: dataURI,
			},
		})
	}

	// Create a chat completion request with the image
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:         openai.ChatMessageRoleUser,
					MultiContent: messageParts,
				},
			},
			Tools: []openai.Tool{
				{
					Type: openai.ToolTypeFunction,
					Function: &openai.FunctionDefinition{
						Name:        "response",
						Description: "Return the results in a structured format",
						Parameters:  responseSchema,
					},
				},
			},
		},
	)
	if err != nil {
		result.Error = err
		return
	}

	if len(resp.Choices) == 0 {
		result.Error = fmt.Errorf("no response received")
		return
	}
	if len(resp.Choices[0].Message.ToolCalls) == 0 {
		result.Error = fmt.Errorf("no tool calls found")
		return
	}

	// Unmarshal the response
	if err := jsonschema.VerifySchemaAndUnmarshal(*responseSchema, []byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), result); err != nil {
		result.Error = err
	}

	// Replace the emote name prefix placeholder with the actual prefix
	result.TwitchEmoteName = strings.Replace(result.TwitchEmoteName, emoteNamePrefixPlaceholder, emoteNamePrefix, 1)
	result.Filename = strings.Replace(result.Filename, emoteNamePrefixPlaceholder, emoteNamePrefix, 1)

	// Save the images to disk
	for key, imgBytes := range imagesBytes {
		// Determine the file extension based on the image format
		fileExtension := "png"
		if len(imgBytes) > 3 && imgBytes[0] == 0x47 && imgBytes[1] == 0x49 && imgBytes[2] == 0x46 { // GIF magic number
			fileExtension = "gif"
		}

		// Set filename
		filename := fmt.Sprintf("%s.%s.%s", result.Filename, key, fileExtension)

		// Save the image to disk
		if err := saveBytesToDisk(imgBytes, filename); err != nil {
			result.Error = err
			return
		}

		// Append the filename to the result
		result.DestinationFiles = append(result.DestinationFiles, filename)
	}

	// Save the result to disk
	if err := saveResultToDisk(*result, fmt.Sprintf("%s.json", result.Filename)); err != nil {
		result.Error = err
		return
	}
}

func getImage(filename string) (image.Image, *gif.GIF, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	imageConfig, modelConfig, err := func(filename string) (image.Config, string, error) {
		return image.DecodeConfig(file)
	}(filename)
	if err != nil {
		return nil, nil, err
	}

	if imageConfig.ColorModel == nil {
		return nil, nil, fmt.Errorf("image.DecodeConfig returned a nil ColorModel")
	}

	if imageConfig.Height == 0 || imageConfig.Width == 0 {
		return nil, nil, fmt.Errorf("image.DecodeConfig returned a zero Height or Width")
	}

	if imageConfig.Width > 4096 || imageConfig.Height > 4096 {
		return nil, nil, fmt.Errorf("image.DecodeConfig returned a Width or Height greater than hardcoded maximum size of 4096")
	}

	// Try to decode as GIF first
	file.Seek(0, 0) // rewind the file
	gifImg, err := gif.DecodeAll(file)
	if err == nil {
		// It's an animated GIF
		return gifImg.Image[0], gifImg, nil
	}

	// Try to decode as still image
	file.Seek(0, 0) // rewind the file
	img, modelDecode, err := image.Decode(file)

	if modelConfig != modelDecode {
		return nil, nil, fmt.Errorf("image.DecodeConfig and image.Decode return different models")
	}

	return img, nil, err
}

func imageToPNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeGIF(g *gif.GIF) ([]byte, error) {
	var buf bytes.Buffer
	err := gif.EncodeAll(&buf, g)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func imgBytesToDataURI(imgBytes []byte) string {
	base64Img := base64.StdEncoding.EncodeToString(imgBytes)

	if len(imgBytes) > 3 && imgBytes[0] == 0x47 && imgBytes[1] == 0x49 && imgBytes[2] == 0x46 { // GIF magic number
		return fmt.Sprintf("data:image/gif;base64,%s", base64Img)
	}

	return fmt.Sprintf("data:image/png;base64,%s", base64Img)
}

func resizeImage(img image.Image, width, height uint) image.Image {
	return resize.Resize(width, height, img, resize.Lanczos3)
}

func resizeToSquare(img image.Image, size uint) image.Image {
	sizeInt := int(size)

	// Create a new RGBA image with the desired square size
	squareImg := image.NewRGBA(image.Rect(0, 0, sizeInt, sizeInt))

	// Calculate the scaling factors
	bounds := img.Bounds()
	scaleX := float64(size) / float64(bounds.Dx())
	scaleY := float64(size) / float64(bounds.Dy())
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate the new dimensions
	newWidth := int(float64(bounds.Dx()) * scale)
	newHeight := int(float64(bounds.Dy()) * scale)

	// Create a new RGBA image for the resized original
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize the original image
	draw.ApproxBiLinear.Scale(resized, resized.Rect, img, bounds, draw.Over, nil)

	// Calculate the position to center the resized image
	x := (sizeInt - newWidth) / 2
	y := (sizeInt - newHeight) / 2

	// Draw the resized image onto the square image
	draw.Draw(squareImg, image.Rect(x, y, x+newWidth, y+newHeight), resized, image.Point{}, draw.Over)

	return squareImg
}

func resizeGIFToSquare(g *gif.GIF, size uint) *gif.GIF {
	resized := &gif.GIF{
		Image:     make([]*image.Paletted, len(g.Image)),
		Delay:     make([]int, len(g.Delay)),
		LoopCount: g.LoopCount,
	}
	for i, frame := range g.Image {
		resized.Image[i] = resizeToSquarePaletted(frame, size)
		resized.Delay[i] = g.Delay[i]
	}
	return resized
}

func resizeToSquarePaletted(img *image.Paletted, size uint) *image.Paletted {
	sizeInt := int(size)
	bounds := img.Bounds()
	newImg := image.NewPaletted(image.Rect(0, 0, sizeInt, sizeInt), img.Palette)

	// Calculate scaling factors
	scaleX := float64(size) / float64(bounds.Dx())
	scaleY := float64(size) / float64(bounds.Dy())
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Calculate new dimensions
	newWidth := int(float64(bounds.Dx()) * scale)
	newHeight := int(float64(bounds.Dy()) * scale)

	// Resize
	draw.ApproxBiLinear.Scale(newImg, image.Rect(0, 0, newWidth, newHeight), img, bounds, draw.Over, nil)

	return newImg
}

func saveBytesToDisk(data []byte, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = out.Write(data)
	return err
}

func saveImageToDisk(img image.Image, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	return png.Encode(out, img)
}

func saveResultToDisk(result Result, filename string) error {
	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}

	_, err = out.Write(data)
	return err
}
