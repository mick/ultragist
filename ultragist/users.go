package ultragist

import (
	"fmt"
)

func GetUserById(userId string) {

	fmt.Printf("user by userId for %s \n", userId)
}

func UpdateUser(key string) {
	// fingerprint := key.Fingerprint

}

func CreateUser(key string) {
	// fingerprint := key.Fingerprint

}

type User struct {
	UserId   string `json:"userId"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
