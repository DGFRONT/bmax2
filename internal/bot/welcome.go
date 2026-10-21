package bot

// welcomeText is sent after the user starts the bot.
const welcomeText = `Добро пожаловать в официальный каталог сервисов "Республика Северная Осетия - Алания". Здесь собраны полезные республиканские сервисы и ресурсы. Для запуска нажмите кнопку "Открыть".`

// welcomeMessage links to the mini app of the bot with the given username.
func welcomeMessage(username string) newMessage {
	return newMessage{
		Text: welcomeText,
		Attachments: []attachment{{
			Type: "inline_keyboard",
			Payload: keyboardPayload{Buttons: [][]button{
				{{Type: "open_app", Text: "Открыть", WebApp: username}},
			}},
		}},
	}
}
