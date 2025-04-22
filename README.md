# whatsapp-demo-chatbot-golang

- [Документация на русском](README_RU.md).

An example of a chatbot written in Go using the API service for Whatsapp [green-api.com](https://green-api.com/en/).
The chatbot clearly demonstrates the use of the API to send text messages, files, pictures, locations, contacts, and integrates OpenAI GPT for intelligent conversations.


## Content

* [Installing the environment for running the chatbot](#setting-up-the-environment-for-running-the-chatbot)
* [Launch chatbot](#launch-a-chatbot)
* [Chatbot setup](#setting-up-a-chatbot)
* [Usage](#usage)
* [Code structure](#code-structure)
* [Message management](#message-management)
* [GPT functionality](#gpt-functionality)


## Setting up the environment for running the chatbot

To run the chatbot, you need to install the Golang environment. Download the latest release suitable for your operating system from [official website](https://go.dev/dl/). Follow the default settings and complete the environment installation.

After completion, you need to check whether the environment was deployed correctly. To do this, open a command line (for example, cmd or bash) and enter the query:
```
    go version
```
To work correctly, the response to the entered request must be a version of Go no lower than:
```
    go version go 1.19
```

Download and unzip the [zip-archive](https://github.com/green-api/whatsapp-demo-chatbot-golang) of the project or clone it with the version control system command:

<details>
<summary>How to install Git version control?</summary>

Download and install the Git version control system appropriate for your operating system from [official website](https://git-scm.com/downloads).
</details>

```
    git clone https://github.com/green-api/whatsapp-demo-chatbot-golang
```

Open the project in any IDE.

The environment for launching the chatbot is ready, now you need to configure and launch the chatbot on your Whatsapp account.

## Launch a chatbot

In order to set up a chatbot on your Whatsapp account, you need to go to [your personal account](https://console.greenapi.com/) and register. For new users, [instructions](https://greenapi.com/en/docs/before-start/) are provided for setting up an account and obtaining the parameters necessary for the chatbot to work, namely:
```
    idInstance
    apiTokenInstance
```

You'll also need an OpenAI API key to use the GPT functionality. You can obtain one from the [OpenAI platform](https://platform.openai.com/).

Create a `.env` file in the project root with the following variables:
```
ID_INSTANCE=your_instance_id
AUTH_TOKEN=your_api_token
OPENAI_API_KEY=your_openai_api_key
```

Don't forget to enable all notifications in your instance settings, so that the chatbot can immediately start receiving messages.

You can then run the program by clicking start in the IDE interface or entering the following query on the command line:
```
    go run main.go
```
This request will start the chatbot. The process begins with chatbot initialization, which includes changing the settings of the associated instance.

The library [whatsapp-chatbot-golang](https://github.com/green-api/whatsapp-chatbot-golang) contains a mechanism for changing instance settings using the [SetSettings](https://green-api.com/en/docs/api/account/SetSettings/) method, which is launched when the chatbot is turned on.

All settings for receiving notifications are disabled by default; the chatbot will enable the following settings:
```
     "incomingWebhook": "yes",
     "pollMessageWebhook": "yes",
     "markIncomingMessagesReaded": "yes"
```
which are responsible for receiving notifications about incoming messages and polls.

The process of changing settings takes several minutes, during which time the instance will be unavailable. Messages sent to the chatbot during this time will not be processed.

After the settings are applied, notifications about previously received incoming messages will be deleted. This process is also written in the library [whatsapp-chatbot-golang](https://github.com/green-api/whatsapp-chatbot-golang) and starts automatically after changing the settings.

This is necessary so that the chatbot does not start processing messages from old chats.

After changing the settings and deleting incoming notifications, the chatbot will begin to respond to messages as standard. In total, this process takes no more than 5 minutes.

To stop the chatbot, use the keyboard shortcut `Ctrl + C` in the command line.

## Setting up a chatbot

By default, the chatbot uses links to download files from the network, but users can add their own links to files, one for a file of any extension pdf / docx /... and one for a picture.

Links must lead to files from cloud storage or public access. In the file [`endpoints.go`](scenes/endpoints.go) there is the following code to send the file:
```go
case "2":
    message.SendUrlFile(
    "https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/corgi.pdf",
    "corgi.pdf",
    util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))
```
Add a link to a file of any extension as the first parameter of the `answerWithUrlFile` method and specify the file name in the second parameter. The file name must contain an extension, for example "somefile.pdf".
This line after modification will be in the following format:

```go
case "2":
    message.SendUrlFile(
    "https://...somefile.pdf",
    "corgi.pdf",
    util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))
```

All changes must be saved, after which you can launch the chatbot. To launch the chatbot, return to [step 2](#launch-chatbot).

## Usage

If the previous steps have been completed, then the chatbot should be working on your Whatsapp account. It is important to remember that the user must be authorized in [personal account](https://console.green-api.com/).

Now you can send messages to the chatbot!

The chatbot will respond to any message sent to your account.
Since the chatbot supports 2 languages - Russian and English - before greeting the interlocutor, the chatbot will ask you to select a language of communication:
```
1 - English
2 - Russian
```
Answer 1 or 2 to select the language for further communication. After you send 1, the chatbot will send a welcome message in English:
```
Welcome to GREEN-API chatbot, user! GREEN-API provides the following types of data sending. Select a number from the list to check how the sending method works

1. Text message 📩
2. File 📋
3. Picture 🖼
4. Contact 📱
5. Geolocation 🌎
6. ...
14. 🔥 Conversation with ChatGPT 🤖

To return to the beginning write stop or 0
```
By selecting a number from the list and sending it, the chatbot will answer which API sent this type of message and share a link to information about the API.

For example, by sending 1, the user will receive in response:
```
This message was sent via the sendMessage method

To find out how the method works, follow the link
https://greenapi.com/en/docs/api/sending/SendMessage/
```

If you send something other than numbers 1-14, the chatbot will succinctly answer:
```
Sorry, I didn't quite understand you, write a menu to see the possible options
```
The user can also call up the menu by sending a message containing "menu". And by sending "stop", the user will end the conversation with the chatbot and receive the message:
```
Thank you for using the GREEN-API chatbot, user!
```

### GPT Chat Mode

By selecting option 14, you can interact with OpenAI's GPT model:

```
🤖 You have started a conversation with ChatGPT.
Ask any questions, and ChatGPT will try to answer them.
To return to the main menu, type *menu*, *exit*, *stop*, or *back*.
```

In this mode, your messages will be processed by GPT, and you'll receive intelligent responses. The conversation history is maintained throughout your session, allowing for contextual interactions.

To exit GPT mode and return to the main menu, type any of the exit commands like "menu", "exit", "back", etc.

## Code structure

The main file of the chatbot is [`main.go`](main.go), it contains the `main` function and program execution begins from there. In this class, the bot object is initialized using the `BotFactory` class, the GPT bot is configured and registered, the first scene is set, and the bot is launched.

```go
func main() {
    // Load environment variables
    err := godotenv.Load(".env")
    
    // Initialize the base bot
    baseBot := chatbot.NewBot(idInstance, authToken)

    // Initialize and register the GPT bot
    gptConfig := gptbot.GPTBotConfig{
       IDInstance:       idInstance,
       APITokenInstance: authToken,
       OpenAIApiKey:     openaiToken,
       Model:            gptbot.ModelGPT4o,
       MaxHistoryLength: 10,
       SystemMessage:    "You are a helpful WhatsApp assistant.",
    }
    gptHelper := gptbot.NewWhatsappGptBot(gptConfig)
    registry.RegisterGptHelper(gptHelper)

    // Set the start scene and launch the bot
    baseBot.SetStartScene(scenes.StartScene{})
    baseBot.StartReceivingNotifications()
}
```

This bot uses a scene pattern to organize its code. This means that the chatbot logic is divided into fragments (scenes), the scene corresponds to a certain state of the dialogue and is responsible for processing the response.

Only one scene can be active at a time for each dialogue.

For example, the first scene [`start.go`](scenes/start.go) is responsible for the welcome message. Regardless of the text of the message, the bot asks what language is convenient for the user and includes the following scene, which is responsible for processing the response.

There are 5 scenes in the bot:

- Scene [`start.go`](scenes/start.go) - responds to any incoming message, sends a list of available languages. Launches the `MainMenu` scene.
- Scene [`mainMenu.go`](scenes/mainMenu.go) - processes the user's selection and sends the main menu text in the selected language. Launches the `Endpoints` scene
- Scene [`endpoints.go`](scenes/endpoints.go) - executes the method selected by the user and sends a description of the method in the selected language. Can transition to the GPT scene when option 14 is selected.
- Scene [`createGroup.go`](scenes/createGroup.go) - The scene creates a group if the user said that he added the bot to his contacts. If not, returns to the "endpoints" scene.
- Scene [`gptScene.go`](scenes/gptScene.go) - Handles GPT conversation mode, processing user messages through OpenAI's API and maintaining conversation context.

The file [`util.go`](util/util.go) contains the `IsSessionExpired()` method which is used to set the start scene again if the bot has not been contacted for more than 2 minutes.

The file [`ymlReader.go`](util/ymlReader.go) contains the `getString()` method which returns strings from the `strings.xml` file by key. This file is used to store the texts of the bot's responses.

A new component is the [`registry`](registry) package, which provides a global access point to the GPT bot instance using a registry pattern. This allows any scene to access the GPT functionality without tight coupling.

## Message management

As the chatbot indicates in its responses, all messages are sent via the API. Documentation on message sending methods can be found at [greenapi.com/en/docs/api/sending](https://greenapi.com/en/docs/api/sending/).

As for receiving messages, messages are read through the HTTP API. Documentation on methods for receiving messages can be found at [greenapi.com/en/docs/api/receiving/technology-http-api](https://greenapi.com/en/docs/api/receiving/technology-http-api/).

The chatbot uses the library [whatsapp-chatbot-golang](https://github.com/green-api/whatsapp-chatbot-golang), where methods for sending and receiving messages are already integrated, so messages are read automatically and sending regular text messages is simplified.

For example, a chatbot automatically sends a message to the contact from whom it received the message:
```go
     message.AnswerWithText(util.GetString([]string{"select_language"}))
```
However, other send methods can be called directly from the [whatsapp-api-client-golang](https://github.com/green-api/whatsapp-api-client-golang) library. Like, for example, when receiving an avatar:
```go
     message.GreenAPI.Methods().Service().GetAvatar(chatId)
```

## GPT Functionality

The chatbot integrates with OpenAI's GPT models using the [whatsapp-chatgpt-go](https://github.com/green-api/whatsapp-chatgpt-go) library. This enables the bot to have intelligent conversations with users.

### How it works

1. **Initialization**: The GPT bot is initialized in `main.go` with configuration including the OpenAI API key and system prompt.
2. **Registry Pattern**: The bot instance is stored in a registry to be accessible from any scene.
3. **GPT Scene**: A dedicated scene (`gptScene.go`) handles the GPT conversation mode.
4. **Session Management**: The GPT scene maintains conversation history using the session data, enabling contextual exchanges.
5. **Exit Commands**: Users can exit the GPT mode using various commands in different languages.


## License

Licensed under [Creative Commons Attribution-NoDerivatives 4.0 International (CC BY-ND 4.0)](https://creativecommons.org/licenses/by-nd/4.0/).

[LICENSE](https://github.com/green-api/whatsapp-demo-chatbot-golang/blob/master/LICENCE).