package dto

// IsMiniMaxH3ImageModel matches the models using MiniMax's OpenAI Images API,
// rather than the legacy image-01 /v1/image_generation protocol.
func IsMiniMaxH3ImageModel(model string) bool {
	return model == "MiniMax-H3-Image" || model == "MiniMax-H3-Image-Highspeed"
}
