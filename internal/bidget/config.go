package bidget

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

var RequiredKeys = []string{"API_KEY", "API_SECRET", "API_PASSPHRASE", "PERIOD"}

var (
	apiKey        string
	apiSecret     string
	apiPassphrase string
	timeout       time.Duration
)

const (
	baseURL = "https://api.bitget.com"
)

func LoadEnvInteractive() map[string]string {
	env, _ := godotenv.Read()
	if env == nil {
		env = make(map[string]string)
	}

	iscreated := false
	reader := bufio.NewReader(os.Stdin)
	for _, key := range RequiredKeys {
		if env[key] == "" {
			iscreated = true
			if key == "PERIOD" {
				fmt.Println("the time interval between balance enquiries")
				fmt.Println("Type Hint: 30s, 2m, 1h")
			}
			fmt.Printf("Enter value for %s: ", key)
			input, _ := reader.ReadString('\n')
			env[key] = strings.TrimSpace(input)
			godotenv.Write(env, ".env")
		}
	}

	if iscreated {
		godotenv.Write(env, ".env")
	}

	godotenv.Load(".env")
	apiKey = os.Getenv("API_KEY")
	apiSecret = os.Getenv("API_SECRET")
	apiPassphrase = os.Getenv("API_PASSPHRASE")

	timeout, _ = time.ParseDuration(os.Getenv("PERIOD"))

	return env
}

func SaveEnv(env map[string]string) error {
	return godotenv.Write(env, ".env")
}
