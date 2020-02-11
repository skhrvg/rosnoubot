package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tealeg/xlsx"
)

// Тип данных обозначающий одну запись в БД, т.е. одно занятие
type entry struct {
    group string
    subgroup int
    dayoff bool
    date time.Time
    subject, time, workType, professor, campus, lectureHall, comment, correction string
}

// Преобразует слайс из занятий одной группы в .sql файл
func writeSQL(entries []entry, group string, institute string, form string, level string) string {
    var rawSQL string
    rawSQL = "DELETE FROM `schedule` WHERE `gr0up` = '" + group + "';\n"
    rawSQL += fmt.Sprintf("INSERT IGNORE INTO `groups` (`gr0up`, `institute`, `form`, `los`) VALUES ('%s', '%s', '%s', '%s');\n", group, institute, form, level)
    for _, e := range entries {
        var dayoff int
        if e.dayoff {
            dayoff = 1
        }
        rawSQL += fmt.Sprintf("INSERT INTO `schedule` (`gr0up`, `subgroup`, `date`, `dayoff`, `subject`, `time`, `type`, `professor`, `comment`, `campus`) VALUES ('%s', '%d', '%04d-%02d-%02d', '%d', '%s', '%s', '%s', '%s', '%s', '%s');\n", 
            e.group, e.subgroup, e.date.Year(), e.date.Month(), e.date.Day(), dayoff, e.subject, e.time, e.workType, e.professor, e.comment, e.campus)
    }
    return rawSQL
} 

// Рассчитывает индексы стартовых ячеек по дням недели
func calculateWeekdays(xlFile *xlsx.File) ([6]int, [6]bool) {
    var (
        weekdayStart [6]int
        weekdayDayoff [6]bool
    )
    for index, row := range xlFile.Sheets[0].Rows {
        switch row.Cells[1].Value {
        case "ПОНЕДЕЛЬНИК": weekdayStart[0] = index + 1
        case "ВТОРНИК": weekdayStart[1] = index + 1
        case "СРЕДА": weekdayStart[2] = index + 1
        case "ЧЕТВЕРГ": weekdayStart[3] = index + 1
        case "ПЯТНИЦА": weekdayStart[4] = index + 1
        case "СУББОТА": weekdayStart[5] = index + 1
        }
    }
    for i := 0 ; i < 5; i++ { if weekdayStart[i+1]-weekdayStart[i] < 4 { weekdayDayoff[i] = true } }
    for _, row := range xlFile.Sheets[0].Rows[weekdayStart[5]:] {
        if row.Cells[1].Value != "" { return weekdayStart, weekdayDayoff }
    }
    weekdayDayoff[5] = true
    return weekdayStart, weekdayDayoff
}

// Parse парсит заданный файл расписания и возвращает сгенерированный SQL
func parse(filename string, group string, institute string, form string, level string, startCol int, startDate time.Time) (logs string, rawSQL string, err error) {
    logs = fmt.Sprintln("[INFO]", "Парсинг расписания группы", group+"...")
    xlFile, err := xlsx.OpenFile(filename)
    if err != nil {log.Fatal(err)}

    weekdayStart, weekdayDayoff := calculateWeekdays(xlFile)
    logs += fmt.Sprintln("[INFO]", "Индексы строк дней недели:", weekdayStart)
    logs += fmt.Sprintln("[INFO]", "Постоянные выходные:", weekdayDayoff)
    
    lastCol := 3
    for tc := lastCol; tc < xlFile.Sheets[0].MaxCol; tc++ {
        if xlFile.Sheets[0].Rows[weekdayStart[0]-1].Cells[tc].Value != "" {
            lastCol = tc
        }
    }
    logs += fmt.Sprintln("[INFO]", "Последний столбец:", lastCol)

    var entries []entry
    var currentDate time.Time
    for weekday, row := range weekdayStart { // День недели
        logs += fmt.Sprintln("[INFO]", "Обработка дня недели:", weekday)
        currentDate = startDate.AddDate(0, 0, weekday)
        if weekdayDayoff[weekday] {
            var newEntry entry
            newEntry.group = group
            newEntry.subgroup = 0
            newEntry.dayoff = true
            for i := 0; i <= lastCol-startCol; i++ {
                newEntry.date = currentDate.AddDate(0, 0, 7*i)
                entries = append(entries, newEntry)
            }
            logs += fmt.Sprintln("[INFO]", "День недели", weekday, "заполнен выходными.")
            continue
        }
        var lastRow int
        if weekday < 5 { 
            lastRow = weekdayStart[weekday+1]-2
        } else { 
            lastRow = weekdayStart[weekday]+4
            for tc := weekdayStart[weekday]; tc < xlFile.Sheets[0].MaxRow; tc++ {
                if xlFile.Sheets[0].Rows[tc].Cells[0].Value != "" {
                    lastRow = tc+3
                }
            }
            logs += fmt.Sprintln("[INFO]", "Последняя строка:", lastRow) 
        }
        for i := row; i < lastRow; i+=3 { //Предмет
            if xlFile.Sheets[0].Rows[i].Cells[1].Value != "" {
                logs += fmt.Sprintln("[INFO]", "Найден предмет:", xlFile.Sheets[0].Rows[i].Cells[1].Value)
            } else {
                logs += fmt.Sprintln("[WARN]", "Пустой предмет. День недели:", weekday, "| Строка:", i)
            }
            currentDate = startDate.AddDate(0, 0, weekday)
            for j := startCol; j <= lastCol; j++ { // Дни
                if xlFile.Sheets[0].Rows[i].Cells[j].Value != "" { 
                    //fmt.Printf("[%d:%d] ", i, j)
                    //fmt.Println(xlFile.Sheets[0].Rows[i].Cells[j].Value)
                    var newEntry entry
                    newEntry.group = group
                    newEntry.subgroup = 0
                    switch xlFile.Sheets[0].Rows[i].Cells[j].Value { 
                    case "Л": newEntry.workType = "Лекция"
                    case "С": newEntry.workType = "Семинар"
                    case "ПЗ": newEntry.workType = "Практическое занятие"
                    case "ЗАЧ": newEntry.workType = "ЗАЧЕТ"
                    case "Л/ПЗ": newEntry.workType = "Лекция / Практическое занятие"
                    case "Л/С": newEntry.workType = "Лекция / Семинар"
                    case "Лаб": newEntry.workType = "ЛАБОРАТОРНАЯ РАБОТА"
                    case "ДИФ.ЗАЧ": newEntry.workType = "ДИФ. ЗАЧЕТ"
                    case "ЗАЩ": newEntry.workType = "ЗАЩИТА"
                    case "С/Л": newEntry.workType = "Семинар / Лекция"
                    case "ПЗ/Л": newEntry.workType = "Практическое занятие / Лекция"
                    case "Л/ЗАЧ": newEntry.workType = "Лекция / ЗАЧЕТ"
                    case "К": newEntry.workType = "КОНСУЛЬТАЦИЯ"
                    case "ЭКЗ": newEntry.workType = "ЭКЗАМЕН"
                    default: 
                        newEntry.workType = xlFile.Sheets[0].Rows[i].Cells[j].Value
                        logs += fmt.Sprintln("[WARN]", "Неизвестный тип занятия:", newEntry.workType, "| Строка:", i, "Столбец:", j)
                    }
                    newEntry.subject = xlFile.Sheets[0].Rows[i].Cells[1].Value
                    newEntry.professor = xlFile.Sheets[0].Rows[i+1].Cells[1].Value
                    newEntry.campus = xlFile.Sheets[0].Rows[i+2].Cells[1].Value
                    newEntry.time = xlFile.Sheets[0].Rows[i].Cells[0].Value
                    newEntry.date = currentDate
                    newEntry.comment = strings.TrimSpace(xlFile.Sheets[0].Rows[i+1].Cells[j].Value)
                    newEntry.comment += strings.TrimSpace(xlFile.Sheets[0].Rows[i+2].Cells[j].Value)
                    entries = append(entries, newEntry)
                } else if row == weekdayStart[weekday] {
                    isDayoff := true
                    for r := row; r < lastRow+1; r++ {
                        if xlFile.Sheets[0].Rows[r].Cells[j].Value != "" {
                            isDayoff = false
                            break
                        }
                    }
                    if isDayoff {
                        var newEntry entry
                        newEntry.group = group
                        newEntry.subgroup = 0
                        newEntry.dayoff = true
                        newEntry.date = currentDate
                        entries = append(entries, newEntry)
                    }
                }
                currentDate = currentDate.AddDate(0, 0, 7)
            }
        }
    } 
    //fmt.Println(entries)
    logs += fmt.Sprintln("[INFO]", "Успешный парсинг!")
    logs += fmt.Sprintln("[INFO]", "Генерация SQL запроса...")
    rawSQL = writeSQL(entries, group, institute, form, level)
    logs += fmt.Sprintln("[INFO]", "SQL запрос сгенерирован!")
    return logs, rawSQL, nil
}


