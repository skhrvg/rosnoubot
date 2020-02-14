package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

func main() {
	log.Println("[BOT]", "Подготовка к запуску...")
	log.Println("[CONFIG]", "Локальный порт:", tgPort)
	log.Println("[CONFIG]", "Webhook URL:", tgURL)
	log.Println("[CONFIG]", "Telegram token:", tgToken)
	log.Println("[CONFIG]", "Telegram admin userid:", tgAdminID)
	log.Println("[CONFIG]", "БД MySQL:", mysqlConnection)

	// Создание вебхука
	b, err := tb.NewBot(tb.Settings{
		Token: tgToken,
		Poller: &tb.Webhook{
			Listen:   ":" + tgPort,
			Endpoint: &tb.WebhookEndpoint{PublicURL: tgURL},
		},
	})
	if err != nil {
		log.Fatal("[FATAL]", err)
	}

	// Реплай-клавиатуры
	keyboardSetup1 := [][]tb.ReplyButton{
		[]tb.ReplyButton{tb.ReplyButton{Text: "ИИСиИКТ"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "ИГТ"}, tb.ReplyButton{Text: "ИБТ"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "ИЭУиФ"}, tb.ReplyButton{Text: "ЮИ"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "ИПиП"}, tb.ReplyButton{Text: "НИ"}},
	}
	keyboardSetup2 := [][]tb.ReplyButton{
		[]tb.ReplyButton{tb.ReplyButton{Text: "Очная"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "Очно-заочная"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "Заочная"}},
	}
	keyboardSetup3 := [][]tb.ReplyButton{
		[]tb.ReplyButton{tb.ReplyButton{Text: "Бакалавриат"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "Магистратура"}},
	}
	runSQLBtn := tb.InlineButton{Text: "Run SQL", Unique: "runSQL"}
	keyboardParse := [][]tb.InlineButton{
		[]tb.InlineButton{runSQLBtn},
	}
	broadcastConfirmBtn := tb.InlineButton{Text: "Отправить", Unique: "broadcastConfirmBtn"}
	keyboardBroadcast := [][]tb.InlineButton{
		[]tb.InlineButton{broadcastConfirmBtn},
	}
	keyboardMain := [][]tb.ReplyButton{
		[]tb.ReplyButton{tb.ReplyButton{Text: "📅  Сегодня"}, tb.ReplyButton{Text: "📅  Завтра"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "📅  Эта неделя"}, tb.ReplyButton{Text: "📅  След. неделя"}},
		[]tb.ReplyButton{tb.ReplyButton{Text: "🔄  Изм. группу"}, tb.ReplyButton{Text: "⚠️  Нашли ошибку?"}},
	}

	// Отправка сообщения об ошибке пользователям с screen = ResetRequired
	cmdResetRequired := func(m *tb.Message) {
		b.Send(m.Sender, "⚠️  *ОШИБКА ЧТЕНИЯ ПРОФИЛЯ ИЗ БАЗЫ ДАННЫХ*\n\nК сожалению, бот не может найти ваши настройки. Сбросить настройки можно командой /reset.",
			&tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
	}

	// Добавление нового пользователя или сброс настроек пользователя
	cmdResetUserSettings := func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		userSetScreen(m.Sender.ID, "setup1")
		b.Send(m.Sender, "🤖  *Привет!*\n\nС помощью этого бота ты сможешь узнать расписание для своей группы РосНОУ. Для начала тебе нужно ответить на 4 простых вопроса.\n\n_Этот бот не является \"официальным\" и никаким образом не связан с администрацией РосНОУ. Информация, предоставляемая ботом, может быть недостоверной или неактуальной. По любым вопросам ты можешь обращаться к админу в ЛС (@skhrvg) или в главном меню бота с помощью специальной кнопки._",
			tb.ParseMode("Markdown"))
		b.Send(m.Sender, "🛠️  *Настройка бота [1/4]*\n\nВыбери свой институт.",
			&tb.ReplyMarkup{ReplyKeyboard: keyboardSetup1}, tb.ParseMode("Markdown"))
		b.Send(m.Sender, "ℹ️  *Подсказка:*\n\n_Если у тебя не отображаются кнопки бота, нажми на иконку с 4 квадратами справа от поля ввода._",
			tb.ParseMode("Markdown"))
	}
	b.Handle("/start", cmdResetUserSettings)
	b.Handle("/reset", cmdResetUserSettings)
	b.Handle("🔄  Изм. группу", cmdResetUserSettings)

	// Восстановления screen по-умолчанию (main) по запросу пользователя
	b.Handle("/cancel", func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if sliceContains([]string{"parse", "report"}, userGetScreen(m.Sender.ID)) {
			userSetScreen(m.Sender.ID, "main")
			b.Send(m.Sender, "⛔️  *Действие отменено.*", &tb.ReplyMarkup{ReplyKeyboard: keyboardMain, ResizeReplyKeyboard: true},
				tb.ParseMode("Markdown"))
		}
	})

	// WIP: Парсинг всех файлов в директории
	b.Handle("/parseall", func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if m.Sender.ID == tgAdminID {
			institute := strings.Split(m.Text, " ")[1]
			folder, _ := os.Open("parser/")
			names, _ := folder.Readdir(-1)
			folder.Close()
			for _, file := range names {
				filename := "parser/" + file.Name()
				group := strings.Split(file.Name(), ".")[0]
				log.Printf(group)
				logs, rawSQL, _ := parse(filename, group, institute, "Очная", "Бакалавриат",
					3, time.Date(2020, 2, 3, 0, 0, 0, 0, time.Now().UTC().Location()))
				log.Printf(logs)
				f, _ := os.Create("temp/" + group + ".sql")
				f.WriteString(rawSQL)
				f.Close()
			}
		}
	})
	b.Handle("/confirmall", func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if m.Sender.ID == tgAdminID {
			folder, _ := os.Open("temp/")
			names, _ := folder.Readdir(-1)
			folder.Close()
			for _, file := range names {
				group := strings.Split(file.Name(), ".")[0]
				log.Println("[DB]", "Uploading", group)
				fileExec("temp/" + group + ".sql")
			}
		}
	})

	// Парсинг отправленного файла и внесение изменений в БД
	// Только для администратора
	var parseArgs []string
	b.Handle("/parse", func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if m.Sender.ID == tgAdminID {
			parseArgs = strings.Split(m.Text, " ")[1:]
			userSetScreen(m.Sender.ID, "parse")
			b.Send(m.Sender, fmt.Sprintf("*Режим загрузки расписания:*\n%s (%s, %s, %s)", parseArgs[0], parseArgs[1], "Очная", "Бакалавриат"),
				tb.ParseMode("Markdown"))
		}
	})
	b.Handle(&runSQLBtn, func(c *tb.Callback) {
		err := fileExec("temp/" + parseArgs[0] + ".sql")
		b.Respond(c, &tb.CallbackResponse{Text: fmt.Sprintln(err), ShowAlert: true})
		removeFile("temp/" + parseArgs[0] + ".sql")
	})
	b.Handle(tb.OnDocument, func(m *tb.Message) {
		if userGetScreen(m.Sender.ID) == "parse" {
			userSetScreen(m.Sender.ID, "main")
			xlfile, _ := b.GetFile(m.Document.MediaFile())
			writeFile("temp/"+parseArgs[0]+".xlsx", xlfile)
			logs, rawSQL, _ := parse("temp/"+parseArgs[0]+".xlsx", parseArgs[0], parseArgs[1], "Очная", "Бакалавриат",
				3, time.Date(2020, 2, 3, 0, 0, 0, 0, time.Now().UTC().Location()))
			log.Printf(logs)
			f, err := os.Create("temp/" + parseArgs[0] + ".sql")
			f.WriteString(rawSQL)
			f.Close()
			b.Send(m.Sender, "`"+logs+"`", tb.ParseMode("Markdown"))
			_, err = b.Send(m.Sender, &tb.Document{File: tb.FromDisk("temp/" + parseArgs[0] + ".sql"), FileName: parseArgs[0] + ".sql"},
				&tb.ReplyMarkup{InlineKeyboard: keyboardParse})
			if err != nil {
				log.Println("[WARN]", err)
			}
			removeFile("temp/" + parseArgs[0] + ".xlsx")
			parseArgs = nil
		}
	})

	// Отправка сообщения пользователю по id
	// Только для администратора
	b.Handle("/send", func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if m.Sender.ID == tgAdminID {
			args := strings.SplitN(m.Text, " ", 3)[1:]
			intID, _ := strconv.Atoi(args[0])
			b.Send(&tb.User{ID: intID}, fmt.Sprintf("💬  *Сообщение от администратора:*\n\n%s", args[1]), tb.ParseMode("Markdown"))
			b.Send(&tb.User{ID: tgAdminID}, fmt.Sprintf("*Сообщение отправлено пользователю %s:*\n\n%s", args[0], args[1]), tb.ParseMode("Markdown"))
		}
	})

	// Рассылка группе пользователей
	// Только для администратора
	var broadcastUserList []string
	var broadcastMessage string
	var broadcastTarget string
	b.Handle("/broadcast", func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if m.Sender.ID == tgAdminID {
			args := strings.SplitN(m.Text, " ", 4)[1:]
			broadcastUserList = getUserIDs(args[0], args[1])
			broadcastMessage = args[2]
			broadcastTarget = args[1]
			b.Send(&tb.User{ID: tgAdminID}, fmt.Sprintf("*Подтердите рассылку %d пользователям (*`WHERE %s = %s`*):*\n\n%s", len(broadcastUserList), args[0], args[1], broadcastMessage),
				&tb.ReplyMarkup{InlineKeyboard: keyboardBroadcast}, tb.ParseMode("Markdown"))
		}
	})
	b.Handle(&broadcastConfirmBtn, func(c *tb.Callback) {
		counter := 0
		for _, uID := range broadcastUserList {
			intID, _ := strconv.Atoi(uID)
			_, err := b.Send(&tb.User{ID: intID}, fmt.Sprintf("💬  *Рассылка для %s:*\n\n%s", broadcastTarget, broadcastMessage), tb.ParseMode("Markdown"))
			log.Println("[CHAT]", "Рассылка пользователю", uID)
			if err != nil {
				log.Println("[WARN]", err)
			} else {
				counter++
			}
		}
		b.Respond(c, &tb.CallbackResponse{Text: fmt.Sprintf("Сообщение отправлено %d пользователям.", counter), ShowAlert: true})
		b.Send(&tb.User{ID: tgAdminID}, "Рассылка завершена", &tb.ReplyMarkup{ReplyKeyboard: keyboardMain, ResizeReplyKeyboard: true}, tb.ParseMode("Markdown"))
		broadcastUserList = nil
		broadcastMessage = ""
		broadcastTarget = ""
	})

	// Получение из БД, сборка и отправка расписания
	sendSchedule := func(m *tb.Message) {
		var iStart, iLimit int
		switch m.Text {
		case "📅  Сегодня":
			iStart, iLimit = 0, 1
		case "📅  Завтра":
			iStart, iLimit = 1, 2
		case "📅  Эта неделя":
			iStart, iLimit = 0, 7-int(time.Now().Weekday())
		case "📅  След. неделя":
			iStart, iLimit = 8-int(time.Now().Weekday()), 14-int(time.Now().Weekday())
		default:
			iStart, iLimit = 0, 0
		}
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		if userGetScreen(m.Sender.ID) == "main" {
			for i := iStart; i < iLimit; i++ {
				b.Send(m.Sender,
					dayToMsg(getDay(userGet(m.Sender.ID, "gr0up"), time.Now().AddDate(0, 0, i).Format("2006-01-02")), userGet(m.Sender.ID, "gr0up"), time.Now().AddDate(0, 0, i).Format("2006-01-02")),
					&tb.ReplyMarkup{ReplyKeyboard: keyboardMain, ResizeReplyKeyboard: true}, tb.ParseMode("Markdown"),
				)
			}
		} else {
			log.Println("[WARN]", "Пользователь", m.Sender.ID, "вызвал расписание не с экрана 'main'. Текущий экран:", userGetScreen(m.Sender.ID))
		}
	}
	b.Handle("📅  Сегодня", sendSchedule)
	b.Handle("📅  Завтра", sendSchedule)
	b.Handle("📅  Эта неделя", sendSchedule)
	b.Handle("📅  След. неделя", sendSchedule)

	// Хендлер для кастомного ввода
	b.Handle(tb.OnText, func(m *tb.Message) {
		log.Println("[CHAT]", m.Sender.ID, m.Sender.FirstName, m.Sender.LastName, "@"+m.Sender.Username, ">>>", m.Text)
		switch userGetScreen(m.Sender.ID) {

		// Первоначальная настройка
		case "setup1":
			if sliceContains([]string{"ИИСиИКТ", "ИГТ", "ИБТ", "ИЭУиФ", "ЮИ", "ИПиП", "НИ"}, m.Text) {
				userSetScreen(m.Sender.ID, "setup2")
				userSet(m.Sender.ID, "institute", m.Text)
				b.Send(m.Sender, "🛠️  *Настройка бота [2/4]*\n\nВыбери форму обучения.", &tb.ReplyMarkup{ReplyKeyboard: keyboardSetup2}, tb.ParseMode("Markdown"))
			} else {
				b.Send(m.Sender, "⚠️  *Неверный институт.*\n\nВыбери свой институт используя кнопки ниже.", &tb.ReplyMarkup{ReplyKeyboard: keyboardSetup1}, tb.ParseMode("Markdown"))
			}
		case "setup2":
			if sliceContains([]string{"Очно-заочная", "Заочная"}, m.Text) {
				userSetScreen(m.Sender.ID, "WIP-form")
				userSet(m.Sender.ID, "form", m.Text)
				b.Send(m.Sender, "⚠️  *К сожалению, сейчас бот не работает с расписанием для данной формы обучения.*\n\nТы можешь выбрать другую форму обучения сбросив настройки командой /reset.", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
				break
			}
			if sliceContains([]string{"Очная", "Очно-заочная", "Заочная"}, m.Text) {
				userSetScreen(m.Sender.ID, "setup3")
				userSet(m.Sender.ID, "form", m.Text)
				b.Send(m.Sender, "🛠️  *Настройка бота [3/4]*\n\nВыбери свой уровень обучения.", &tb.ReplyMarkup{ReplyKeyboard: keyboardSetup3}, tb.ParseMode("Markdown"))
			} else {
				b.Send(m.Sender, "⚠️  *Неверная форма обучения.*\n\nВыбери форму обучения используя кнопки ниже.", &tb.ReplyMarkup{ReplyKeyboard: keyboardSetup2}, tb.ParseMode("Markdown"))
			}
		case "setup3":
			if sliceContains([]string{"Магистратура"}, m.Text) {
				userSetScreen(m.Sender.ID, "WIP-level")
				userSet(m.Sender.ID, "los", m.Text)
				b.Send(m.Sender, "⚠️  *К сожалению, сейчас бот не работает с расписанием для данной формы обучения.*\n\nТы можешь выбрать другую форму обучения сбросив настройки командой /reset.", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
				break
			}
			if sliceContains([]string{"Бакалавриат", "Магистратура"}, m.Text) {
				userSetScreen(m.Sender.ID, "setup4")
				userSet(m.Sender.ID, "los", m.Text)
				groups := strings.Join(groupsList(userGet(m.Sender.ID, "institute"), userGet(m.Sender.ID, "form"), userGet(m.Sender.ID, "los")), ", ")
				if groups != "" {
					b.Send(m.Sender, "🛠️ *Настройка бота [4/4]*\n\nВведи свой номер группы.\n\n_Доступные группы:\n"+groups+"_", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
				} else {
					b.Send(m.Sender, "⚠️  *К сожалению, бот не нашел группы для тебя.*\n\nСбросить настройки можно командой /reset.", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
				}
			} else {
				b.Send(m.Sender, "⚠️  Неверный уровень обучения.\n\nВыбери свой уровень обучения с помощью кнопок ниже.", &tb.ReplyMarkup{ReplyKeyboard: keyboardSetup3}, tb.ParseMode("Markdown"))
			}
		case "setup4":
			if sliceContains(groupsList(userGet(m.Sender.ID, "institute"), userGet(m.Sender.ID, "form"), userGet(m.Sender.ID, "los")), m.Text) {
				userSetScreen(m.Sender.ID, "main")
				userSet(m.Sender.ID, "gr0up", m.Text)
				b.Send(m.Sender, "✅  *Настройка бота завершена!*\n\nКстати, чтобы узнать расписание на определённый день, ты можешь отправить дату в формате ДД.ММ.ГГГГ.", &tb.ReplyMarkup{ReplyKeyboard: keyboardMain, ResizeReplyKeyboard: true}, tb.ParseMode("Markdown"))
			} else {
				b.Send(m.Sender, "⚠️  *К сожалению, бот не нашел расписание для твоей группы "+m.Text+".*\n\nПроверь номер группы и отправь его снова или используй команду /reset чтобы выбрать другой институт или форму обучения.", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
			}

		// Главный экран
		case "main":
			if m.Text == "⚠️  Нашли ошибку?" {
				userSetScreen(m.Sender.ID, "report")
				b.Send(m.Sender, "*Спасибо, что помогаешь сделать бота лучше!*\n\nПодробно опиши свою проблему. Администратор увидит твое сообщение и отправит ответ через бота или в личку.\n\n_Отменить отправку отчета можно командой _/cancel_._",
					&tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
			} else {
				t, err := time.Parse("02.01.2006", m.Text)
				if err == nil {
					b.Send(m.Sender,
						dayToMsg(getDay(userGet(m.Sender.ID, "gr0up"), t.Format("2006-01-02")), userGet(m.Sender.ID, "gr0up"), t.Format("2006-01-02")),
						&tb.ReplyMarkup{ReplyKeyboard: keyboardMain, ResizeReplyKeyboard: true}, tb.ParseMode("Markdown"),
					)
				} else {
					log.Println("[WARN]", err)
				}
			}

		// Экран отправки отчета
		case "report":
			userSetScreen(m.Sender.ID, "main")
			b.Send(&tb.User{ID: tgAdminID}, fmt.Sprintf("⚠️  *REPORT*\nИмя: `%s`\nФамилия: `%s`\nUsername: @%s\nID: `%d`\n\nГруппа: `%s (%s | %s | %s)`", m.Sender.FirstName, m.Sender.LastName, m.Sender.Username, m.Sender.ID, userGet(m.Sender.ID, "gr0up"), userGet(m.Sender.ID, "institute"), userGet(m.Sender.ID, "form"), userGet(m.Sender.ID, "los")), tb.ParseMode("Markdown"))
			b.Forward(&tb.User{ID: tgAdminID}, m, tb.ParseMode("Markdown"))
			b.Send(m.Sender, "✅  *Отчет отправлен.*", &tb.ReplyMarkup{ReplyKeyboard: keyboardMain, ResizeReplyKeyboard: true}, tb.ParseMode("Markdown"))

		// Экраны "Not Implemented"
		case "WIP-form":
			b.Send(m.Sender, "⚠️  *К сожалению, сейчас бот не работает с расписанием для данной формы обучения.*\n\nТы можешь выбрать другую форму обучения сбросив настройки командой /reset.", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))
		case "WIP-level":
			b.Send(m.Sender, "⚠️  *К сожалению, сейчас бот не работает с расписанием для данного уровня обучения.*\n\nТы можешь выбрать другую форму обучения сбросив настройки командой /reset.", &tb.ReplyMarkup{ReplyKeyboardRemove: true}, tb.ParseMode("Markdown"))

		// Экран неоходимости пересоздания профиля
		case "ResetRequired":
			cmdResetRequired(m)
		default:
			cmdResetRequired(m)
			userSetScreen(m.Sender.ID, "ResetReqired")
		}

	})

	// СТАРТУЕМ!
	log.Println("[BOT]", "Запуск бота...")
	b.Start()
}
