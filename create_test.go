package hive

import (
	"testing"
	"time"

	"gorm.io/gorm/utils/tests"
)

func TestCreate(t *testing.T) {
	now := time.Now()
	var user = User{
		ID1:       1,
		Name:      "philhuan",
		Age:       2,
		Active:    true,
		Salary:    1.2,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		Date:      time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local),
		Score: map[string]int{
			"English": 100,
			"math":    101,
		},
	}

	if err := DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user, got error %v", err)
	}

	var result User
	if err := DB.Find(&result, user.ID1).Error; err != nil {
		t.Fatalf("failed to query user, got error %v", err)
	}

	tests.AssertEqual(t, result, user)

	//type partialUser struct {
	//	Name string
	//}
	//var partialResult partialUser
	//if err := DB.Raw("select * from users where id1 = ?", user.ID1).Scan(&partialResult).Error; err != nil {
	//	t.Fatalf("failed to query partial, got error %v", err)
	//}
}
