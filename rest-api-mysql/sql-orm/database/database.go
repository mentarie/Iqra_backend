package database

import (
	"log"
	"gorm.io/gorm"
)

type User struct {
	Id       int    `json:"id" gorm:"primary_key"`
	Id_user  string `json:"id_user"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func GetUsers(db *gorm.DB) ([]User, error) {
	var users []User
	if err := db.Find(&users).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return nil, err
	}

	return users, nil
}

func GetUser(id int, db *gorm.DB) (User, error) {
	var user User
	if err := db.Where(&User{Id: id}).Find(&user).Error; err != nil {
		log.Println("failed to get data :", err.Error())
		return User{}, err
	}

	return user, nil
}

func CreateUser(user User, db *gorm.DB) error {
	if err := db.Create(&user).Error; err != nil {
		return err
	}
	log.Println("Success insert data")
	return nil
}

func UpdateUser(id int, user User, db *gorm.DB) error {
	if err := db.Model(&User{}).Where(&User{
		Id: id,
	}).Updates(user).Error; err != nil {
		return err
	}
	log.Println("Success update data")
	return nil
}

func DeleteUser(id int, db *gorm.DB) error {
	var user User
	if err := db.Where(&User{Id: id}).Find(&user).Error; err != nil {
		return err
	}
	if err := db.Delete(user).Error; err != nil {
		return err
	}
	return nil
}
