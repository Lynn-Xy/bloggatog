package main

import (
	cfg "github.com/Lynn-Xy/bloggatog/internal/config"
	"fmt"
	"os"
	"errors"
	"context"
	"io"
	"html"
	"net/http"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/google/uuid"
	"github.com/Lynn-Xy/bloggatog/internal/database"
	"log"
	"time"
	"encoding/xml"
)

type state struct {
	cfg *cfg.Config
	db *database.Queries
}

type command struct {
	Name string
	Arguments []string
}

type commands struct {
	list map[string]func(*state, command) error
}

type RSSItem struct {
	Title string `xml:"title"`
	Link string `xml:"link"`
	Description string `xml:"description"`
	PubDate string `xml:"pubDate"`
}

type RSSFeed struct {
	Channel struct {
		Title string `xml:"title"`
		Link string `xml:"link"`
		Description string `xml:"description"`
		Item []RSSItem `xml:"item"`
	} `xml:"channel"`
}

func (c *commands) Run(s *state, cmd command) error {
	handler, ok := c.list[cmd.Name]
	if !ok {
		return fmt.Errorf("error running cmd: %v", cmd.Name)
	}
	err := handler(s, cmd)
	if err != nil {
		return fmt.Errorf("error executing command %v: %v", cmd.Name, err)
	}
	return nil
}

func (c *commands) Register(name string, f func(*state, command) error) {
	c.list[name] = f
}

func HandlerLogin(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("error logging in: current username not found")
}
	_, err := s.db.GetUserByName(context.Background(), cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error retrieving username from database: %v", err)
	}
	err = s.cfg.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting username: %v", err)
	}
	fmt.Printf("current user: %v has been set\n", cmd.Arguments[0])
	return nil
}

func HandlerRegister(s *state, cmd command) error {
	if len(cmd.Arguments) < 1 {
		return errors.New("error registering user: no username provided")
	}
	_, err := s.db.GetUserByName(context.Background(), cmd.Arguments[0])
	if err == nil {
		return fmt.Errorf("error registering new user: %v - user already exists in database", cmd.Arguments[0])
	}
	params := database.CreateUserParams{uuid.New(), time.Now(), time.Now(), cmd.Arguments[0]}
	_, err = s.db.CreateUser(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error creating new user: %v", err)
	}
	err = s.cfg.SetUser(cmd.Arguments[0])
	if err != nil {
		return fmt.Errorf("error setting current user: %v", err)
	}
	fmt.Printf("current user: %v has been registered\n", cmd.Arguments[0])
	return nil
}

func HandlerAddFeed(s *state, cmd command) error {
	if len(cmd.Arguments) > 2 {
		return errors.New("Feed name must be only one word, camelcase combinations are permitted")
	}
	user := s.cfg.CurrentUsername
	if user == "guest" || user == "" {
		return errors.New("error adding feed: no user is currently logged in")
	}
	dbUser, err := s.db.GetUserByName(context.Background(), user)
	if err != nil {
		return fmt.Errorf("error retrieving user from database: %v", err)
	}
	params := database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name: sql.NullString{
			String: cmd.Arguments[0],
			Valid: true,
		},
		Url: cmd.Arguments[1],
		UserID: dbUser.ID}
	feed, err := s.db.CreateFeed(context.Background(), params)
	if err != nil {
		return fmt.Errorf("error creating feed in database: %v", err)
	}
	fmt.Printf(`
	* Id: %v
	* CreatedAt: %v
	* UpdatedAt: %v
	* Name: %v
	* Url: %v
	* UserId: %v`,
	feed.ID,
	feed.CreatedAt,
	feed.UpdatedAt,
	feed.Name,
	feed.Url,
	feed.UserID)
	return nil
}

func HandlerUsers(s *state, cmd command) error {
	users, err := s.db.GetAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error retrieving users from database: %v", err)
	}
	for _, user := range users {
		if user.Name == s.cfg.CurrentUsername {
			fmt.Printf("* %v (current)\n", user.Name)
		} else {
			fmt.Printf("* %v\n", user.Name)
		}
	}
	return nil
}

func HandlerReset(s *state, cmd command) error {
	err := s.db.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("error deleting users from database: %v", err)
	}
	fmt.Print("all users deleted from database")
	return nil
}

func HandlerAgg(s *state, cmd command) error {
	URL := "https://www.wagslane.dev/index.xml"
	feed, err := fetchFeed(context.Background(), URL)
	if err != nil {
		return fmt.Errorf("error fetching rss feed: %v", err)
	}
	fmt.Printf("%+v\n", *feed)
	return nil
}

func HandlerFeeds(s *state, cmd command) error {
	feeds, err := s.db.GetAllFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("error retrieving feeds from database: %v", err)
	}
	for _, feed := range feeds {
		fmt.Printf("* Name: %v", feed.Name)
		fmt.Printf("* Url: %v", feed.Url)
		user, err := s.db.GetUserNameById(context.Background(), feed.UserID)
		if err != nil {
			return fmt.Errorf("error retrieving user name from database: %v", err)
		}
		fmt.Printf("* Username: %v", user)
	}
	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {
	newClient := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %v", err)
	}
	req.Header.Set("User-Agent", "gator")
	resp, err := newClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending http get request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error retrieving rss feed contents: server response %v", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading http get response body: %v", err)
	}
	var feed RSSFeed
	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling http get request response body data: %v", err)
	}
	for idx, item := range feed.Channel.Item {
		cleanedTitle := html.UnescapeString(item.Title)
		feed.Channel.Item[idx].Title = cleanedTitle
		cleanedDescription := html.UnescapeString(item.Description)
		feed.Channel.Item[idx].Description = cleanedDescription
	}
	cleanedTitle := html.UnescapeString(feed.Channel.Title)
	feed.Channel.Title = cleanedTitle
	cleanedDescription := html.UnescapeString(feed.Channel.Description)
	feed.Channel.Description = cleanedDescription
	return &feed, nil
}

func main() {
	config, err := cfg.Read()
	if err != nil {
		fmt.Printf("error reading config: %v", err)
		os.Exit(1)
	}
	db, err := sql.Open("postgres", config.DBURL)
	if err != nil {
		log.Printf("error opening database connection: %v", err)
	}
	dbQ := database.New(db)
	s := &state{&config, dbQ}
	commands := commands{
		list: make(map[string]func(*state, command) error),
	}

	commands.Register("login", HandlerLogin)
	commands.Register("register", HandlerRegister)
	commands.Register("reset", HandlerReset)
	commands.Register("users", HandlerUsers)
	commands.Register("agg", HandlerAgg)
	commands.Register("addfeed", HandlerAddFeed)
	commands.Register("feeds", HandlerFeeds)

	args := os.Args
	if len(args) < 2 {
		fmt.Println("not enough arguments provided")
		os.Exit(1)
	}

	cmd := command{
		Name:      args[1],
		Arguments: args[2:],
	}

	if err := commands.Run(s, cmd); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
