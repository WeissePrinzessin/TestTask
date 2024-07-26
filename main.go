package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	 _ "github.com/WeissePrinzessin/TestTask/docs"
)

// @title People Info API
// @version 1.0
// @description This is a sample server Petstore server.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@swagger.io
// @host localhost:8080
// @BasePath /
// @schemes http

type User struct {
	ID             int    `json:"id" db:"id"`
	PassportNumber string `json:"passportNumber" db:"passport_number"`
}

type Task struct {
	ID          int    `json:"id"`
	UserID      int    `json:"user_id"`
	Description string `json:"description"`
}

type TimeLog struct {
	ID        int        `json:"id"`
	UserID    int        `json:"user_id"`
	TaskID    int        `json:"task_id"`
	StartTime time.Time  `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`
}

type People struct {
	Surname    string `json:"surname"`
	Name       string `json:"name"`
	Patronymic string `json:"patronymic"`
	Address    string `json:"address"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

var db *sqlx.DB
var timeLogs []TimeLog
var users []User

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	// Подключение к БД
	dbUser := os.Getenv("db_user")
	dbPassword := os.Getenv("db_password")
	dbName := os.Getenv("db_name")
	dbHost := os.Getenv("db_host")
	dbPort := os.Getenv("db_port")
	dsn := fmt.Sprintf("user=%s password=%s dbname=%s host=%s port=%s sslmode=disable", dbUser, dbPassword, dbName, dbHost, dbPort)
	db, err = sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(files.Handler))

	r.GET("/users", getUsers)  // получение данных о пользователе
	r.POST("/users", createUser) // добавление нового пользователя
	r.PUT("/users/:id", updateUser) // изменение данных пользователя
	r.DELETE("/users/:id", deleteUser) // удаление пользователя

	r.GET("/users/:id/worklogs", getUserWorklogs) // получение трудозатрат по пользователю
	r.POST("/users/:id/tasks/:task_id/start", startTask) // начать отсчет времени по задаче для пользователя
	r.POST("/users/:id/tasks/:task_id/stop", stopTask) // закончить отсчет времени по задаче для пользователя

	r.Run()
}

// getUsers godoc
// @Summary Get users
// @Description Get users with optional passport number filter
// @Produce json
// @Param passportNumber query string false "Passport Number"
// @Param skip query int false "Number of users to skip"
// @Param limit query int false "Number of users to return"
// @Success 200 {array} User
// @Failure 400 {object} ErrorResponse
// @Router /users [get]
func getUsers(c *gin.Context) {
	logrus.Info("Fetching users")

	passportNumber := c.Query("passportNumber")
	skip, _ := strconv.Atoi(c.DefaultQuery("skip", "0"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var filteredUsers []User
	for _, user := range users {
		if passportNumber == "" || user.PassportNumber == passportNumber {
			filteredUsers = append(filteredUsers, user)
		}
	}

	start := skip
	end := skip + limit
	if end > len(filteredUsers) {
		end = len(filteredUsers)
	}

	c.JSON(http.StatusOK, filteredUsers[start:end])
}

// createUser godoc
// @Summary Create user
// @Description Create a new user
// @Accept json
// @Produce json
// @Param user body User true "User"
// @Success 200 {object} User
// @Failure 400 {object} ErrorResponse
// @Router /users [post]
func createUser(c *gin.Context) {
	logrus.Info("Creating new user")

	var newUser User
	if err := c.ShouldBindJSON(&newUser); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	newUser.ID = len(users) + 1
	users = append(users, newUser)
	c.JSON(http.StatusOK, newUser)
}

// updateUser godoc
// @Summary Update user
// @Description Update an existing user by ID
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param user body User true "User"
// @Success 200 {object} User
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{id} [put]
func updateUser(c *gin.Context) {
	logrus.Info("Updating user")

	id, _ := strconv.Atoi(c.Param("id"))
	var updatedUser User
	if err := c.ShouldBindJSON(&updatedUser); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}
	for i, user := range users {
		if user.ID == id {
			users[i].PassportNumber = updatedUser.PassportNumber
			c.JSON(http.StatusOK, users[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
}

// deleteUser godoc
// @Summary Delete user
// @Description Delete a user by ID
// @Param id path int true "User ID"
// @Success 200 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /users/{id} [delete]
func deleteUser(c *gin.Context) {
	logrus.Info("Deleting user")

	id, _ := strconv.Atoi(c.Param("id"))
	for i, user := range users {
		if user.ID == id {
			users = append(users[:i], users[i+1:]...)
			c.JSON(http.StatusOK, ErrorResponse{Error: "User deleted"})
			return
		}
	}
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "User not found"})
}

// getUserWorklogs godoc
// @Summary Get user worklogs
// @Description Get worklogs for a user
// @Produce json
// @Param id path int true "User ID"
// @Param start query string false "Start date in RFC3339 format"
// @Param end query string false "End date in RFC3339 format"
// @Success 200 {array} TimeLog
// @Failure 404 {object} ErrorResponse
// @Router /users/{id}/worklogs [get]
func getUserWorklogs(c *gin.Context) {
	logrus.Info("Fetching user worklogs")

	userID, _ := strconv.Atoi(c.Param("id"))
	start := c.Query("start")
	end := c.Query("end")

	var filteredLogs []TimeLog
	for _, log := range timeLogs {
		if log.UserID == userID {
			if start != "" && log.StartTime.Before(parseTime(start)) {
				continue
			}
			if end != "" && log.EndTime != nil && log.EndTime.After(parseTime(end)) {
				continue
			}
			filteredLogs = append(filteredLogs, log)
		}
	}

	if len(filteredLogs) == 0 {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "No worklogs found"})
		return
	}

	// Сортировка по убыванию времени затрат
	sort.Slice(filteredLogs, func(i, j int) bool {
		return getDuration(filteredLogs[i]) > getDuration(filteredLogs[j])
	})

	c.JSON(http.StatusOK, filteredLogs)
}

// startTask godoc
// @Summary Start task
// @Description Start a task for a user
// @Produce json
// @Param id path int true "User ID"
// @Param task_id path int true "Task ID"
// @Success 200 {object} TimeLog
// @Failure 404 {object} ErrorResponse
// @Router /users/{id}/tasks/{task_id}/start [post]
func startTask(c *gin.Context) {
	logrus.Info("Starting task")

	userID, _ := strconv.Atoi(c.Param("id"))
	taskID, _ := strconv.Atoi(c.Param("task_id"))

	newLog := TimeLog{
		ID:        len(timeLogs) + 1,
		UserID:    userID,
		TaskID:    taskID,
		StartTime: time.Now(),
	}
	timeLogs = append(timeLogs, newLog)
	c.JSON(http.StatusOK, newLog)
}

// stopTask godoc
// @Summary Stop task
// @Description Stop a task for a user
// @Produce json
// @Param id path int true "User ID"
// @Param task_id path int true "Task ID"
// @Success 200 {object} TimeLog
// @Failure 404 {object} ErrorResponse
// @Router /users/{id}/tasks/{task_id}/stop [post]
func stopTask(c *gin.Context) {
	logrus.Info("Stopping task")

	userID, _ := strconv.Atoi(c.Param("id"))
	taskID, _ := strconv.Atoi(c.Param("task_id"))

	for i, log := range timeLogs {
		if log.UserID == userID && log.TaskID == taskID && log.EndTime == nil {
			endTime := time.Now()
			timeLogs[i].EndTime = &endTime
			c.JSON(http.StatusOK, timeLogs[i])
			return
		}
	}
	c.JSON(http.StatusNotFound, ErrorResponse{Error: "Active time log not found"})
}

func parseTime(timeStr string) time.Time {
	parsedTime, _ := time.Parse(time.RFC3339, timeStr)
	return parsedTime
}

func getDuration(log TimeLog) time.Duration { // продолжительность времени, затраченное на выполнение задачи
	if log.EndTime == nil {
		return 0
	}
	return log.EndTime.Sub(log.StartTime)
}
