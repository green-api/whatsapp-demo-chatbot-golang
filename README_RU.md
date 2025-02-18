# whatsapp-demo-chatbot-golang

- [Documentation in English](README.md).

Пример чатбота написанного на Golang с использованием API сервиса для Whatsapp [green-api.com](https://green-api.com/).
Чатбот наглядно демонстрирует использование API для отправки текстовых сообщений, файлов, картинок, локаций и контактов.


## Содержание

* [Установка среды для запуска чатбота](#установка-среды-для-запуска-чатбота)
* [Запуск чатбота](#запуск-чатбота)
* [Настройка чатбота](#настройка-чатбота)
* [Использование](#использование)
* [Структура кода](#структура-кода)
* [Управление сообщениями](#управление-сообщениями)


## Установка среды для запуска чатбота

Для запуска чатбота необходимо произвести установку среды Golang. Загрузите последний релиз, подходящий для вашей операционной системы, с [официального вебсайта](https://go.dev/dl/). Следуйте настройкам по умолчанию и завершите установку среды.

После завершения необходимо проверить была ли среда развернута корректно. Для этого откройте командную строку (например, cmd или bash) и введите запрос:
```
    go version
```
Для корректной работы, ответом на введеный запрос должна быть версия Go не ниже:
```
    go version go 1.20
```

Скачайте и разархивируйте [zip-архив](https://github.com/green-api/whatsapp-demo-chatbot-golang) проекта или клонируйте его командой системы контроля версий:

<details>
<summary>Как установить систему контроля версий Git?</summary>

Скачайте и установите систему контроля версий Git, подходящую для используемой операционной системы, с [официального вебсайта](https://git-scm.com/downloads).
 
</details>
```
git clone https://github.com/green-api/whatsapp-demo-chatbot-golang
```

Откройте проект в любой IDE.

Среда для запуска чатбота готова, теперь необходимо произвести настройку и запустить чатбот на вашем аккаунте Whatsapp.

## Запуск чатбота

Для того, чтобы настроить чатбот на своем аккаунте Whatsapp, Вам необходимо перейти в [личный кабинет](https://console.green-api.com/) и зарегистрироваться. Для новых пользователей предоставлена [инструкция](https://green-api.com/docs/before-start/) для настройки аккаунта и получения необходимых для работы чатбота параметров, а именно:
```
    idInstance
    apiTokenInstance
```

Не забудьте включить все уведомления в настройках инстанса, чтобы чатбот мог сразу начать принимать сообщения.
После получения данных параметров, найдите класс [`main.go`](main.go) и введите `idInstance` и `apiTokenInstance` в значения констант.
Инициализация данных необходима для связывания бота с Вашим Whatsapp аккаунтом:

```
    const (
        idInstance       = "{INSTANCE}"
        apiTokenInstance = "{TOKEN}"
    )
```

Далее можно запускать программу, для этого нажмите пуск в интерфейсе IDE или введите следующий запрос в командной строке:
```
go run main.go
```
Данный запрос запустит работу чатбота. Процесс начинается с инициализации чатбота, которая включает в себя изменение настроек связанного инстанса.

В библиотеке [whatsapp-chatbot-golang](https://github.com/green-api/whatsapp-chatbot-golang) прописан механизм изменения настроек инстанса методом [SetSettings](https://green-api.com/docs/api/account/SetSettings/), который запускается при включении чатбота.

Все настройки по получению уведомлений выключены по умолчанию, чатбот включит следующие настройки:
```
    "incomingWebhook": "yes",
    "outgoingMessageWebhook": "yes",
    "outgoingAPIMessageWebhook": "yes",
```
которые отвечают за получение уведомлений о входящих и исходящих сообщениях.

Процесс изменения настроек занимает несколько минут, в течении этого времени инстанс будет недоступен. Сообщения отправленные чатботу в это время не будут обработаны.

После того, как будут применены настройки, произойдет удаление уведомлений о полученных ранее входящих сообщениях. Этот процесс так же прописан в библиотеке [whatsapp-chatbot-golang](https://github.com/green-api/whatsapp-chatbot-golang) и автоматически запускается после изменения настроек.

Это необходимо для того, чтобы чатбот не начал обрабатывать сообщения со старых чатов.

После того, как изменения настроек и удаление входящих уведомлений будут исполнены, чатбот начнет стандартно отвечать на сообщения. Суммарно этот процесс занимает не больше 5 минут.

Чтобы остановить работу чатбота, используйте сочетание клавиш `Ctrl + C` в командной строке.

## Настройка чатбота

По умолчанию чатбот использует ссылки для выгрузки файлов из сети, однако пользователи могут добавить свои ссылки на файлы, одну для файла любого расширения pdf / docx /... и одну для картинки.

Ссылки должны вести на файлы из облачного хранилища или открытого доступа. В файле [`endpoints.go`](scenes/endpoints.go) есть следующий код для отправки файла:
```go
case "2":
    message.SendUrlFile(
    "https://storage.yandexcloud.net/sw-prod-03-test/ChatBot/corgi.pdf",
    "corgi.pdf",
    util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))
```
Добавьте ссылку на файл любого расширения в качестве первого параметра метода `answerWithUrlFile` и задайте имя файлу во втором параметре. Имя файла должно содержать расширение, например "somefile.pdf".
Данная строка после изменения будет в следующем формате:
```go
case "2":
    message.SendUrlFile(
    "https://...somefile.pdf",
    "corgi.pdf",
    util.GetString([]string{"send_file_message", lang})+util.GetString([]string{"links", lang, "send_file_documentation"}))
```

Все изменения должны быть сохранены, после чего можно запускать чатбот. Для запуска чатбота вернитесь к [пункту 2](#запуск-чатбота).

## Использование

Если предыдущие шаги были выполнены, то на вашем аккаунте Whatsapp должен работать чатбот. Важно помнить, что пользователь должен быть авторизован в [личном кабинете](https://console.green-api.com/).

Теперь вы можете отправлять сообщения чатботу!

Чатбот откликнется на любое сообщение отправленное на аккаунт.
Так как чатбот поддерживает 2 языка - русский и английский - то прежде чем поприветствовать собеседника, чатбот попросит выбрать язык общения:
```
1 - English
2 - Русский
```
Ответьте 1 или 2, чтобы выбрать язык для дальнейшего общения. После того как вы отправите 2, чатбот пришлет приветственное сообщение на русском языке:
```
Добро пожаловать в GREEN-API чатбот, пользователь! GREEN-API предоставляет отправку данных следующих видов. Выберите цифру из списка, чтобы проверить как работает метод отправки

1. Текстовое сообщение 📩
2. Файл 📋
3. Картинка 🖼
4. Контакт 📱
5. Геолокация 🌎
6. ...

Чтобы вернуться в начало напишите стоп
```
Выбрав число из списка и отправив его, чатбот ответит каким API был отправлен данный тип сообщения и поделится ссылкой на информацию об API.

Например, отправив 1, пользователь получит в ответ:
```
Это сообщение отправлено через sendMessage метод

Чтобы узнать как работает метод, пройдите по ссылке
https://green-api.com/docs/api/sending/SendMessage/
```
Если отправить что-то помимо чисел 1-11, то чатбот лаконично ответит:
```
Извините, я не совсем вас понял, напишите меню, чтобы посмотреть возможные опции
```
Так же пользователь может вызвать меню, отправив сообщение содержащее "меню". И отправив "стоп", пользователь завершит беседу с чатботом и получит сообщение:
```
Спасибо за использование чатбота GREEN-API, пользователь!
```

## Структура кода

Основной файл чатбота это [`main.go`](main.go), в нем находится функция `main` и с него начинается выполнение программы. В этом классе происходит инициализация объекта бота при помощи класса `BotFactory`, установка первой сцены и запуск бота.

```go
func main() {
    const (
		// idInstance = '1101123456'
		// apiTokenInstance = 'abcdefghjklmn1234567890oprstuwxyz'
		idInstance       = "{INSTANCE}"
		apiTokenInstance = "{TOKEN}"
	)

    bot := chatbot.NewBot(idInstance, apiTokenInstance)      //Инициализация бота с параметрами INSTANCE и TOKEN из констант

    go func() {                                 //Обработчик ошибок
        select {
		    case err := <-bot.ErrorChannel:
            if err != nil {
                log.Println(err)
            }
        }
    }()
	
    if _, err := bot.GreenAPI.Methods().Account().SetSettings(map[string]interface{}{      //Установка настроек инстанса
 		"incomingWebhook":           "yes",
 		"outgoingMessageWebhook":    "yes",
 		"outgoingAPIMessageWebhook": "yes",
        "pollMessageWebhook":         "yes",
        "markIncomingMessagesReaded": "yes",
 	}); err != nil {
 		log.Fatalln(err)
 	}   

    bot.SetStartScene(scenes.StartScene{})      //Установка стартовой сцены бота

    bot.StartReceivingNotifications()       //Запуск бота
}
```

Данный бот использует паттерн сцен для организации кода. Это значит, что логика чатбота разделена на фрагменты (сцены), сцена соответствует определенному состоянию диалога и отвечает за обработку ответа.

Для каждого диалога одновременно активна может быть только одна сцена.

Например, первая сцена [`start.go`](scenes/start.go) отвечает за приветственное сообщение. Вне зависимости от текста сообщения, бот спрашивает какой язык удобен пользователю и включает следующую сцену, которая отвечает за обработку ответа.

Всего в боте 4 сцены:

- Сцена [`start.go`](scenes/start.go) - отвечает на любое входящее сообщение, отправляет список доступных языков. Запускает сцену `MainMenu`.
- Сцена [`mainMenu.go`](scenes/mainMenu.go) - обрабатывает выбор пользователя и отправляет текст главного меню на выбранном языке. Запускает сцену `Endpoints`
- Сцена [`endpoints.go`](scenes/mainMenu.go) - выполняет выбранный пользователем метод и отправляет описание метода на выбранном языке.
- Сцена [`createGroup.go`](scenes/createGroup.go) - Сцена создает группу, если пользователь сказал, что добавил бота в свои контакты. Если нет, возвращается к сцене «конечные точки».

Файл [`util.go`](util/util.go) содержит метод `IsSessionExpired()` который используется, чтобы снова устанавливать стартовую сцену, если боту не пишут более 2 минут.

Файл [`ymlReader.go`](util/ymlReader.go) содержит метод `getString()` который возвращает строки из файла `strings.xml` по ключам. Этот файл используется для хранения текстов ответов бота.

## Управление сообщениями

Как и указывает чатбот в ответах, все сообщения отправлены через API. Документацию по методам отправки сообщений можно найти на сайте [green-api.com/docs/api/sending](https://green-api.com/docs/api/sending/).

Что касается получения сообщений, то сообщения вычитываются через HTTP API. Документацию по методам получения сообщений можно найти на сайте [green-api.com/docs/api/receiving/technology-http-api](https://green-api.com/docs/api/receiving/technology-http-api/).

Чатбот использует библиотеку [whatsapp-chatbot-golang](https://github.com/green-api/whatsapp-chatbot-golang), где уже интегрированы методы отправки и получения сообщений, поэтому сообщения вычитываются автоматически, а отправка обычных текстовых сообщений упрощена.

Например, чатбот автоматически отправляет сообщение контакту, от которого получил сообщение:
```go
    message.AnswerWithText(util.GetString([]string{"select_language"}))
```
Однако другие методы отправки можно вызвать напрямую из библиотеки [whatsapp-api-client-golang](https://github.com/green-api/whatsapp-api-client-golang). Как, например, при получении аватара:
```go
    message.GreenAPI.Methods().Service().GetAvatar(chatId)
```

## Лицензия

Лицензировано на условиях [Creative Commons Attribution-NoDerivatives 4.0 International (CC BY-ND 4.0)](https://creativecommons.org/licenses/by-nd/4.0/).

[LICENSE](https://github.com/green-api/whatsapp-demo-chatbot-golang/blob/master/LICENCE).
