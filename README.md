# Lossless media bot for Telegram

### Demo: [https://t.me/lossless_media_bot](@lossless_media_bot)

If you've used Telegram mobile client to upload photos or videos, you know it compresses media and usually it looks creepy and video quality is very low. This bot will help you to upload videos and photos in the original quality.

This bot uses Telegram user_api to download and upload files bigger than 50mb.

## How to run own copy of the bot

To run your own bot it is required:
1. to create dev api via my.telegram.org
2. to create a bot via [https://t.me/BotFather](@BotFather)
3. to have got a linux server with docker (and docker-compose) installed. 

When you have all requirements, do the following:
1. Copy docker-compose.yml and .env.example to any folder
2. Update credentials in .env.example and rename it to .env
3. Run bot via ```docker-compose up -d```

## How to use

### Via dialog

1. Send video or photo as a file to the bot
2. The bot will convert it into a video/photo message and send it back to you

### Via channel

1. Add bot to the channel
2. Send video or photo as a file to the channel
3. The bot will convert it into a video/photo message and send it back to the channel
