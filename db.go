package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

// Установка экрана пользователя в БД
func userSetScreen(userid int, newScreen string) {
	db, err := sql.Open("mysql", mysqlConnection)
	defer db.Close()
	checkerr(err)
	// Проверка наличия пользователя в БД и установка экрана
	userExists, _ := db.Query("SELECT * FROM teleusers WHERE id = ?;", userid)
	if userExists.Next() {
		log.Println("[DB]", "Смена экрана пользователя", userid, "на", newScreen)
		db.Exec("UPDATE teleusers SET screen = ? WHERE id = ?;", newScreen, userid)
		return
	}
	// Создание юзера при его отсутствии
	if newScreen == "setup1" {
		log.Println("[DB]", "Новый пользователь:", userid)
		db.Exec("INSERT INTO teleusers (id, screen) VALUES (?, 'setup1')", userid)
	} else {
		log.Println("[WARN]", "Данные о пользователе", userid, "не найдены. ResetRequired!")
		db.Exec("INSERT INTO teleusers (id, screen) VALUES (?, 'ResetRequired');", userid)
	}
}

// Получение экрана пользователя из БД
func userGetScreen(userid int) string {
	db, err := sql.Open("mysql", mysqlConnection)
	defer db.Close()
	checkerr(err)
	// Проверка наличия пользователя в БД и получение экрана
	row, err := db.Query("SELECT screen FROM teleusers WHERE id = ?;", userid)
	if row.Next() {
		var screen string
		row.Scan(&screen)
		return screen
	}
	// Создание юзера при его отсутствии
	log.Println("[WARN]", "Данные о пользователе", userid, "не найдены. ResetRequired!")
	db.Exec("INSERT INTO teleusers (id, screen) VALUES (?, 'ResetRequired');", userid)
	return "ResetRequired"
}

// Установка значения для пользователя в БД
func userSet(userid int, param string, value string) error {
	db, err := sql.Open("mysql", mysqlConnection)
	defer db.Close()
	checkerr(err)
	// Проверка наличия пользователя в БД и установка значения
	userExists, _ := db.Query("SELECT * FROM teleusers WHERE id = ?;", userid)
	if userExists.Next() {
		db.Exec("UPDATE teleusers SET `"+param+"` = ? WHERE id = ?;", value, userid)
		log.Println("[DB]", "Изменена настройка", param, "для пользователя", userid)
		return nil
	}
	// Создание юзера при его отсутствии
	log.Println("[WARN]", "Данные о пользователе", userid, "не найдены. ResetRequired!")
	db.Exec("INSERT INTO teleusers (id, screen) VALUES (?, 'ResetRequired');", userid)
	return fmt.Errorf("user not found in DB")
}

// Получение значения для пользователя из БД
func userGet(userid int, param string) string {
	db, err := sql.Open("mysql", mysqlConnection)
	defer db.Close()
	checkerr(err)
	// Проверка наличия пользователя в БД и получение значения
	row, err := db.Query("SELECT `"+param+"` FROM teleusers WHERE id = ?;", userid)
	if row.Next() {
		var result string
		row.Scan(&result)
		return result
	}
	// Создание юзера при его отсутствии
	log.Println("[WARN]", "Данные о пользователе", userid, "не найдены. ResetRequired!")
	db.Exec("INSERT INTO teleusers (id, screen) VALUES (?, 'ResetRequired');", userid)
	return "<nil>"
}

// Построчное выполнение SQL файла
func fileExec(filename string) error {
	db, err := sql.Open("mysql", mysqlConnection)
	defer db.Close()
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		_, err = db.Exec(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

// Возвращает список групп для определенного института, уровня и формы обучения
func groupsList(institute string, form string, level string) []string {
	db, err := sql.Open("mysql", mysqlConnection)
	checkerr(err)
	defer db.Close()
	var groups []string
	row, err := db.Query("SELECT `gr0up` FROM `groups` WHERE (`institute`, `form`, `los`) = (?, ?, ?);", institute, form, level)
	for row.Next() {
		var group string
		row.Scan(&group)
		groups = append(groups, group)
	}
	return groups
}

// Одно занятие, полученное из БД
type work struct {
	subgroup                                                                     int
	dayoff                                                                       bool
	subject, time, workType, professor, campus, lectureHall, comment, correction string
}

// Получение расписания на день из БД
func getDay(group string, date string) []work {
	db, err := sql.Open("mysql", mysqlConnection)
	checkerr(err)
	defer db.Close()
	var day []work
	row, err := db.Query("SELECT `subgroup`, `dayoff`, `subject`, `time`, `type`, `professor`, `campus`, `lecture_hall`, `comment`, `correction` FROM `schedule` WHERE (`gr0up`, `date`) = (?, ?);", group, date)
	for row.Next() {
		var w work
		row.Scan(&w.subgroup, &w.dayoff, &w.subject, &w.time, &w.workType, &w.professor, &w.campus, &w.lectureHall, &w.comment, &w.correction)
		day = append(day, w)
	}
	return day
}

// Получить список id пользователей по заданному параметру
func getUserIDs(p string, v string) []string {
	var userIDs []string
	db, err := sql.Open("mysql", mysqlConnection)
	checkerr(err)
	defer db.Close()
	q, err := db.Query(fmt.Sprintf("SELECT `id` FROM `teleusers` WHERE `%s` = '%s';", p, v))
	for q.Next() {
		var uID string
		q.Scan(&uID)
		userIDs = append(userIDs, uID)
	}
	return userIDs
}
