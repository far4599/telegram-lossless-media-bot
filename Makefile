build-and-push:
	@docker buildx build -t far4599/telegram-lossless-media-bot:latest --push --platform=linux/arm64,linux/amd64 .