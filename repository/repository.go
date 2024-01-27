package repository

import (
	"fmt"
	"github.com/hotkimho/realworld-api/env"
	"github.com/hotkimho/realworld-api/models"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

var UserRepo *userRepository
var ArticleRepo *articleRepository

func Init() error {

	dsn := makeDSN(
		env.Config.Database.User,
		env.Config.Database.Password,
		env.Config.Database.Host,
		env.Config.Database.Name,
		env.Config.Database.Port)
	if dsn == "" {
		return fmt.Errorf("dsn is empty")
	}

	//	dsn = "dev:!gktrlagh33@tcp(127.0.0.1:3306)/real_world?charset=utf8mb4&parseTime=True&loc=Local"

	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	// Init Repositories
	initRepositories()

	err = DB.AutoMigrate(models.User{}, models.Article{})
	if err != nil {
		return err
	}

	return nil
}

func makeDSN(user, password, host, name string, port int) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, password, host, port, name)
}

func initRepositories() {
	UserRepo = NewUserRepository()
	ArticleRepo = NewArticleRepository()
}
