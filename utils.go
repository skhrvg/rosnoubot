package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"
)

var (
	tgPort          = os.Getenv("ROSNOUBOT_TGPORT")
	tgURL           = os.Getenv("ROSNOUBOT_TGURL")
	tgToken         = os.Getenv("ROSNOUBOT_TGTOKEN")
	tgAdminID, _    = strconv.Atoi(os.Getenv("ROSNOUBOT_TGADMINID"))
	mysqlConnection = os.Getenv("ROSNOUBOT_MYSQL")
)

// Проверка наличия строки в слайсе или массиве
func sliceContains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

// Запись в файл из потока
func writeFile(filepath string, f io.ReadCloser) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, f)
	return err
}

// Удаление файла
func removeFile(filepath string) error {
	err := os.Remove(filepath)
	return err
}

// Проверка на ошибки
func checkerr(err error) {
	if err != nil {
		log.Fatal("[FATAL]", err)
	}
}

// Возвращает день недели по номеру (time pkg)
func weekdayString(date string) string {
	weekday, _ := time.Parse("02.01.2006", date)
	switch weekday.Weekday() {
	case 0:
		return "Воскресенье"
	case 1:
		return "Понедельник"
	case 2:
		return "Вторник"
	case 3:
		return "Среда"
	case 4:
		return "Четверг"
	case 5:
		return "Пятница"
	case 6:
		return "Суббота"
	}
	return "impossible error"
}

// Создает сообщение с расписанием на день из слайса work-ов
func dayToMsg(day []work, group string, date string) string {
	var msg string

	datet, _ := time.Parse("2006-01-02", date)
	date = datet.Format("02.01.2006")
	msg = fmt.Sprintf("📅  *Расписание на %s, %s %s*\n\n", weekdayString(date), date, "["+group+"]")
	if len(day) == 0 {
		msg += "Ничего не найдено. Возможно, произошла ошибка."
		return msg
	}
	if day[0].dayoff {
		msg += "🤟  Выходной!"
		return msg
	}
	for _, w := range day {
		msg += fmt.Sprintf("_%s_\n*%s*\n%s ", w.time, w.subject, w.workType)
		if w.comment != "" {
			msg += fmt.Sprintf("(%s)", w.comment)
		}
		if w.campus != "" {
			msg += fmt.Sprintf("\n%s", w.campus)
		}
		msg += "\n\n"
	}
	return msg
}
