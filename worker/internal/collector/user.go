package collector

import (
	"os"

	"github.com/Zam83-AZE/zaur-test/worker/internal/models"
)

func CollectUser() models.UserInfo {
	// W-10: Full implementation per OS
	return models.UserInfo{Username: os.Getenv("USER")}
}
