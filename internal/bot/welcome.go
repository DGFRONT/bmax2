package bot

import "strings"

// welcomeText describes the service. Keep it in step with seed/catalog.json.
const welcomeText = `**Сервисы региона** — официальные каналы, чаты, боты и сайты РСО-Алания в одном приложении.

Что можно сделать:
⛽ проверить наличие топлива на АЗС
🩺 записаться к врачу
👨‍💼 обратиться в чат-бот МФЦ
🏛️ найти канал министерства или ведомства
🏥 🎓 🎭 ⚽ подписаться на больницы, вузы, колледжи, учреждения культуры и спорта
🚌 💧 узнать новости транспорта и Водоканала

Нажмите «Открыть приложение» ниже или кнопку приложения рядом с полем ввода. Чтобы сразу перейти в раздел, выберите его кнопкой.`

// welcomeMessage greets name (may be empty) and links to the mini app of the
// bot with the given username.
func welcomeMessage(name, username string) newMessage {
	greeting := "Здравствуйте!"
	if name = strings.TrimSpace(name); name != "" {
		greeting = "Здравствуйте, " + name + "!"
	}
	app := func(text, startParam string) button {
		return button{Type: "open_app", Text: text, WebApp: username, Payload: startParam}
	}
	return newMessage{
		Text:   greeting + " 👋\n\n" + welcomeText,
		Format: "markdown",
		Attachments: []attachment{{
			Type: "inline_keyboard",
			Payload: keyboardPayload{Buttons: [][]button{
				{app("📱 Открыть приложение", "")},
				{app("🏛️ Органы власти", "category_government"), app("🏥 Здравоохранение", "category_health")},
				{app("🎓 Вузы", "category_universities"), app("📚 Колледжи", "category_colleges")},
				{app("🎭 Культура", "category_culture"), app("⚽ Спорт", "category_sport")},
			}},
		}},
	}
}
