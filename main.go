package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/dghubble/oauth1"
	"github.com/g8rswimmer/go-twitter/v2"
	"github.com/joho/godotenv"
)

type authorize struct {
	Token string
}

func (a authorize) Add(_ *http.Request) {}

type UserData struct {
	Username  string
	Timestamp time.Time
}

type UsernameMonitor struct {
	client     *twitter.Client
	userID     string
	historical map[string][]UserData
	logFile    *os.File
}

func NewUsernameMonitor(apiKey, apiKeySecret, accessToken, accessTokenSecret, userID string) (*UsernameMonitor, error) {
	config := oauth1.NewConfig(apiKey, apiKeySecret)
	token := oauth1.NewToken(accessToken, accessTokenSecret)
	httpClient := config.Client(context.Background(), token)

	client := &twitter.Client{
		Authorizer: authorize{Token: accessToken},
		Client:     httpClient,
		Host:       "https://api.twitter.com",
	}

	logFile, err := os.OpenFile("username_changes.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %v", err)
	}

	return &UsernameMonitor{
		client:     client,
		userID:     userID,
		historical: make(map[string][]UserData),
		logFile:    logFile,
	}, nil
}

func (m *UsernameMonitor) MonitorUsername(ctx context.Context) {
	// ticker := time.NewTicker(checkInterval)
	// defer ticker.Stop()

	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		return
	// 	case <-ticker.C:
	if err := m.checkUsername(); err != nil {
		log.Printf("Error checking username: %v", err)
	}
	// 	}
	// }
}

func (m *UsernameMonitor) checkUsername() error {
	opts := twitter.UserLookupOpts{
		UserFields: []twitter.UserField{"username", "created_at"},
	}

	user, err := m.client.UserNameLookup(context.Background(), []string{m.userID}, opts)
	if err != nil {
		return fmt.Errorf("failed to lookup user: %v", err)
	}

	if len(user.Raw.Users) == 0 {
		return fmt.Errorf("no user found")
	}
	for _, usr := range user.Raw.Users {
		fmt.Println("user: ", usr)
	}
	currentUsername := user.Raw.Users[0].UserName
	userData := UserData{
		Username:  currentUsername,
		Timestamp: time.Now(),
	}
	fmt.Println("userData: ", userData)
	// Check if username changed
	history := m.historical[m.userID]
	fmt.Println("history: ", len(history))
	if len(history) > 0 && history[len(history)-1].Username != currentUsername {
		change := fmt.Sprintf("[%s] Username changed from %s to %s\n",
			userData.Timestamp.Format(time.RFC3339),
			history[len(history)-1].Username,
			currentUsername)
		fmt.Println(change)
		if _, err := m.logFile.WriteString(change); err != nil {
			return fmt.Errorf("failed to write to log: %v", err)
		}
	}

	// Update historical data
	m.historical[m.userID] = append(m.historical[m.userID], userData)
	return nil
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	apiKey := os.Getenv("TWITTER_API_KEY")
	apiKeySecret := os.Getenv("TWITTER_API_SECRET")
	accessToken := os.Getenv("TWITTER_ACCESS_TOKEN")
	accessTokenSecret := os.Getenv("TWITTER_ACCESS_TOKEN_SECRET")
	userID := os.Getenv("TARGET_USER_ID")

	if apiKey == "" || apiKeySecret == "" || accessToken == "" || accessTokenSecret == "" || userID == "" {
		fmt.Println("env credentials missing")
		return
	}
	monitor, err := NewUsernameMonitor(apiKey, apiKeySecret, accessToken, accessTokenSecret, userID)
	if err != nil {
		log.Fatalf("Failed to create monitor: %v", err)
	}

	ctx := context.Background()
	monitor.MonitorUsername(ctx)
}
