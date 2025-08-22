# Gemini Chat Telegram Bot

This is a Telegram bot that integrates with the Gemini API to provide chat functionalities.

## Getting Started

Follow these steps to set up and run the bot using Docker.

### 1. Clone the Repository

First, clone the project from GitHub:

```bash
git clone https://github.com/vasyvasilie/gemini-chat-tg-bot.git
cd gemini-chat-tg-bot
```

### 2. Build the Docker Image

Build the Docker image for the bot. This process will compile the Go application inside the Docker image.

```bash
docker build -t gemini-chat-tg-bot:latest .
```

### 3. Configure Environment Variables

The bot requires three environment variables to function:

*   `BOT_API_TOKEN`: Your Telegram Bot API Token. You can get this by talking to [BotFather on Telegram](https://t.me/botfather).
*   `GEMINI_API_KEY`: Your Google Gemini API Key. Obtain this from [Google AI Studio](https://aistudio.google.com/app/apikey) or the [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
*   `ALLOWED_USERS`: A comma-separated list of Telegram User IDs (numeric) who are allowed to use the bot.

It's recommended to add these to your `.bashrc` (or equivalent shell configuration file like `.zshrc`) so they are automatically loaded when you start your terminal session.

Open your `.bashrc` file:

```bash
nano ~/.bashrc
# or use your preferred text editor like vim, gedit, etc.
```

Add the following lines, replacing the placeholder values with your actual tokens and user IDs:

```bash
export BOT_API_TOKEN="YOUR_TELEGRAM_BOT_API_TOKEN"
export GEMINI_API_KEY="YOUR_GOOGLE_GEMINI_API_KEY"
export ALLOWED_USERS="YOUR_USER_ID_1,YOUR_USER_ID_2"
export STORAGE_PATH="/tmp/bolt.db"
```

After adding the variables, save the file and apply the changes by sourcing your `.bashrc`:

```bash
source ~/.bashrc
```

### 4. Run the Docker Container

Once the environment variables are set in your shell, you can run the Docker container. The bot will run in the background.

```bash
docker run --name gemini-chat-tg-bot -d -v /tmp/bolt.db:/tmp/bolt.db -e STORAGE_PATH -e BOT_API_TOKEN -e GEMINI_API_KEY -e ALLOWED_USERS gemini-chat-tg-bot:latest
```

To check the logs of the running container, you can use:

```bash
docker logs <container_id_or_name>
```

You can find the container ID or name using `docker ps`.
